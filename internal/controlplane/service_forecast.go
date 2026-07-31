package controlplane

import (
	"context"
	"errors"
	db "espx/internal/domain/db"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	forecastCHQueryTimeout       = 1500 * time.Millisecond
	forecastMinSampleImpressions = int64(1000)
	forecastDefaultRetryAfterSec = 30
)

var (
	ErrForecastClickHouseTimeout = errors.New("forecast clickhouse query timed out")
	ErrForecastUnavailable       = errors.New("forecast service unavailable")
	ErrClickHouseNotConfigured   = errors.New("clickhouse not configured")
)

type CampaignForecastInput struct {
	CustomerID       *uuid.UUID
	BudgetLimitMicro int64
	TargetCountries  []string
	DaypartHours     []int16
	StartAt          time.Time
	EndAt            time.Time
	PacingMode       string
	Timezone         string
}

type SpendCurvePoint struct {
	Hour        string `json:"hour"`
	SpendMicro  int64  `json:"spend_micro"`
	Impressions int64  `json:"impressions"`
}

type ForecastAdvisory struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	SuggestedPacing string `json:"suggested_pacing"`
}

type CampaignForecastDTO struct {
	ImpressionsP50 int64             `json:"impressions_p50"`
	ImpressionsP90 int64             `json:"impressions_p90"`
	SpendCurve     []SpendCurvePoint `json:"spend_curve"`
	LowConfidence  bool              `json:"low_confidence"`
	Advisory       *ForecastAdvisory `json:"advisory,omitempty"`
}

func (s *Service) ForecastCampaign(ctx context.Context, in CampaignForecastInput) (CampaignForecastDTO, error) {
	if s.chQuery == nil {
		return CampaignForecastDTO{}, ErrClickHouseNotConfigured
	}
	if in.BudgetLimitMicro <= 0 {
		return CampaignForecastDTO{}, errValidation("budget_limit_micro must be greater than zero")
	}
	if !in.EndAt.After(in.StartAt) {
		return CampaignForecastDTO{}, ErrInvalidTimeRange
	}
	pacing := normalizeForecastPacing(in.PacingMode)

	chCtx, cancel := context.WithTimeout(ctx, forecastCHQueryTimeout)
	defer cancel()

	lookbackEnd := time.Now().UTC().Truncate(time.Hour)
	lookbackStart := lookbackEnd.Add(-forecastLookbackDays * 24 * time.Hour)

	campaignIDs, err := s.forecastCampaignIDs(chCtx, in.CustomerID)
	if err != nil {
		return CampaignForecastDTO{}, err
	}

	totalSample, hourlySamples, err := s.queryForecastHourlySamples(chCtx, lookbackStart, lookbackEnd, campaignIDs)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(chCtx.Err(), context.DeadlineExceeded) {
			return CampaignForecastDTO{}, ErrForecastClickHouseTimeout
		}
		return CampaignForecastDTO{}, fmt.Errorf("%w: %w", ErrForecastUnavailable, err)
	}

	lowConfidence := int64(totalSample) < forecastMinSampleImpressions
	hourWeights := buildHourWeights(hourlySamples)
	activeHours := enumerateActiveHours(in.StartAt, in.EndAt, in.DaypartHours, in.Timezone)

	flightImpressions := projectFlightImpressions(hourWeights, activeHours, totalSample)
	p50, p90 := impressionPercentiles(hourlySamples, activeHours, totalSample)

	cpmMicro := impliedCPMMicro(in.BudgetLimitMicro, flightImpressions)
	spendCurve := buildSpendCurve(activeHours, in.BudgetLimitMicro, pacing, cpmMicro)

	out := CampaignForecastDTO{
		ImpressionsP50: p50,
		ImpressionsP90: p90,
		SpendCurve:     spendCurve,
		LowConfidence:  lowConfidence,
	}
	if advisory := evenPacingAdvisory(pacing, in.BudgetLimitMicro, p50, cpmMicro); advisory != nil {
		out.Advisory = advisory
	}
	_ = in.TargetCountries
	return out, nil
}

func (s *Service) forecastCampaignIDs(ctx context.Context, customerID *uuid.UUID) ([]uuid.UUID, error) {
	if customerID == nil || *customerID == uuid.Nil {
		return nil, nil
	}
	q := db.New(s.GetPool())
	rows, err := q.ListCampaigns(ctx, db.ListCampaignsParams{
		CustomerID: pgtype.UUID{Bytes: *customerID, Valid: true},
		Limit:      500,
		Offset:     0,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, uuid.UUID(row.ID.Bytes))
	}
	return ids, nil
}

func (s *Service) queryForecastHourlySamples(ctx context.Context, from, to time.Time, campaignIDs []uuid.UUID) (uint64, []forecastHourlySample, error) {
	var (
		query string
		args  []any
	)
	if len(campaignIDs) == 0 {
		query = `
SELECT toHour(hour) AS hr, sum(impression_count) AS impressions
FROM mv_campaign_hourly_impressions
WHERE hour >= ? AND hour < ?
GROUP BY hr
ORDER BY hr`
		args = []any{from, to}
	} else {
		query = `
SELECT toHour(hour) AS hr, sum(impression_count) AS impressions
FROM mv_campaign_hourly_impressions
WHERE hour >= ? AND hour < ? AND campaign_id IN (?)
GROUP BY hr
ORDER BY hr`
		args = []any{from, to, campaignIDs}
	}

	rows, err := s.chQuery.Query(ctx, query, args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var total uint64
	samples := make([]forecastHourlySample, 0, 24)
	for rows.Next() {
		var sample forecastHourlySample
		if err := rows.Scan(&sample.hourOfDay, &sample.impressions); err != nil {
			return 0, nil, err
		}
		total += sample.impressions
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return total, samples, nil
}

func ForecastRetryAfterSec() int {
	return forecastDefaultRetryAfterSec
}
