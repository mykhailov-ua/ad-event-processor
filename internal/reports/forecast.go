package reports

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	forecastLookbackDays           = 90
	forecastUnderfillAdvisoryPct   = 0.20
	forecastMaxSpendCurvePoints    = 2160
	forecastClickHouseQueryTimeout = 1500 * time.Millisecond
	forecastMinSampleImpressions   = int64(1000)
)

type HourlyImpressionSample struct {
	HourOfDay   int
	Impressions uint64
}

type forecastHourlySample = HourlyImpressionSample

func normalizeForecastPacing(mode string) string {
	switch mode {
	case "EVEN", "even":
		return "EVEN"
	default:
		return "ASAP"
	}
}

func BuildHourWeights(samples []HourlyImpressionSample) [24]float64 {
	return buildHourWeights(samples)
}

func QueryForecastHourlySamples(ctx context.Context, host ForecastHost, from, to time.Time, campaignIDs []uuid.UUID) (uint64, []HourlyImpressionSample, error) {
	return queryForecastHourlySamples(ctx, host.ForecastClickHouseQuery(), from, to, campaignIDs)
}

func buildHourWeights(samples []forecastHourlySample) [24]float64 {
	var weights [24]float64
	if len(samples) == 0 {
		for i := range weights {
			weights[i] = 1.0 / 24.0
		}
		return weights
	}
	var sum float64
	for _, s := range samples {
		if s.HourOfDay >= 0 && s.HourOfDay < 24 {
			weights[s.HourOfDay] = float64(s.Impressions)
			sum += float64(s.Impressions)
		}
	}
	if sum <= 0 {
		for i := range weights {
			weights[i] = 1.0 / 24.0
		}
		return weights
	}
	for i := range weights {
		weights[i] /= sum
	}
	return weights
}

func enumerateActiveHours(start, end time.Time, daypart []int16, timezone string) []time.Time {
	loc := time.UTC
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
	}
	daypartSet := make(map[int16]struct{}, len(daypart))
	for _, h := range daypart {
		daypartSet[h] = struct{}{}
	}
	useDaypart := len(daypartSet) > 0

	start = start.In(loc).Truncate(time.Hour)
	end = end.In(loc).Truncate(time.Hour)
	if !end.After(start) {
		return nil
	}
	var hours []time.Time
	for t := start; t.Before(end); t = t.Add(time.Hour) {
		if len(hours) >= forecastMaxSpendCurvePoints {
			break
		}
		if useDaypart {
			if _, ok := daypartSet[int16(t.Hour())]; !ok {
				continue
			}
		}
		hours = append(hours, t.UTC())
	}
	return hours
}

func projectFlightImpressions(weights [24]float64, activeHours []time.Time, totalSample uint64) int64 {
	if len(activeHours) == 0 {
		return 0
	}
	lookbackHours := float64(forecastLookbackDays * 24)
	avgPerHour := float64(totalSample) / lookbackHours
	if avgPerHour <= 0 {
		return 0
	}
	var weighted float64
	for _, h := range activeHours {
		weighted += weights[h.Hour()] * 24.0
	}
	return int64(math.Round(avgPerHour * weighted))
}

func impressionPercentiles(samples []forecastHourlySample, activeHours []time.Time, totalSample uint64) (p50, p90 int64) {
	values := make([]int64, 0, len(activeHours))
	weights := buildHourWeights(samples)
	lookbackHours := float64(forecastLookbackDays * 24)
	avgPerHour := float64(totalSample) / lookbackHours
	for _, h := range activeHours {
		v := avgPerHour * weights[h.Hour()] * 24.0
		if v > 0 {
			values = append(values, int64(math.Round(v)))
		}
	}
	if len(values) == 0 {
		return 0, 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	p50 = values[len(values)*50/100]
	p90 = values[len(values)*90/100]
	if p90 < p50 {
		p90 = p50
	}
	return p50, p90
}

func impliedCPMMicro(budgetMicro, impressions int64) int64 {
	if impressions <= 0 {
		return budgetMicro
	}
	return budgetMicro / impressions
}

func buildSpendCurve(activeHours []time.Time, budgetMicro int64, pacing string, cpmMicro int64) []SpendCurvePoint {
	if len(activeHours) == 0 {
		return []SpendCurvePoint{}
	}
	curve := make([]SpendCurvePoint, 0, len(activeHours))
	if pacing == "EVEN" {
		perHourSpend := budgetMicro / int64(len(activeHours))
		perHourImps := int64(0)
		if cpmMicro > 0 {
			perHourImps = perHourSpend / cpmMicro
		}
		for _, h := range activeHours {
			curve = append(curve, SpendCurvePoint{
				Hour:        h.Format(time.RFC3339),
				SpendMicro:  perHourSpend,
				Impressions: perHourImps,
			})
		}
		return curve
	}

	frontCount := len(activeHours) * 30 / 100
	if frontCount < 1 {
		frontCount = 1
	}
	frontBudget := budgetMicro * 70 / 100
	backBudget := budgetMicro - frontBudget
	frontPer := frontBudget / int64(frontCount)
	backHours := len(activeHours) - frontCount
	var backPer int64
	if backHours > 0 {
		backPer = backBudget / int64(backHours)
	}
	for i, h := range activeHours {
		spend := backPer
		if i < frontCount {
			spend = frontPer
		}
		imps := int64(0)
		if cpmMicro > 0 {
			imps = spend / cpmMicro
		}
		curve = append(curve, SpendCurvePoint{
			Hour:        h.Format(time.RFC3339),
			SpendMicro:  spend,
			Impressions: imps,
		})
	}
	return curve
}

func evenPacingAdvisory(pacing string, budgetMicro, impressionsP50, cpmMicro int64) *ForecastAdvisory {
	if pacing != "EVEN" || budgetMicro <= 0 || impressionsP50 <= 0 || cpmMicro <= 0 {
		return nil
	}
	deliverableSpend := impressionsP50 * cpmMicro
	if deliverableSpend >= budgetMicro {
		return nil
	}
	underfill := float64(budgetMicro-deliverableSpend) / float64(budgetMicro)
	if underfill <= forecastUnderfillAdvisoryPct {
		return nil
	}
	return &ForecastAdvisory{
		Code:            "PACING_UNDERFILL",
		Message:         fmt.Sprintf("EVEN pacing may under-deliver by %.0f%% of budget; consider ASAP for full spend in the flight window", underfill*100),
		SuggestedPacing: "ASAP",
	}
}

func ForecastCampaign(ctx context.Context, host ForecastHost, in CampaignForecastInput) (CampaignForecastDTO, error) {
	if host == nil || host.ForecastClickHouseQuery() == nil {
		return CampaignForecastDTO{}, ErrClickHouseNotConfigured
	}
	if in.BudgetLimitMicro <= 0 {
		return CampaignForecastDTO{}, errValidation("budget_limit_micro must be greater than zero")
	}
	if !in.EndAt.After(in.StartAt) {
		return CampaignForecastDTO{}, ErrInvalidTimeRange
	}
	pacing := normalizeForecastPacing(in.PacingMode)

	clickhouseCtx, cancel := context.WithTimeout(ctx, forecastClickHouseQueryTimeout)
	defer cancel()

	lookbackEnd := time.Now().UTC().Truncate(time.Hour)
	lookbackStart := lookbackEnd.Add(-forecastLookbackDays * 24 * time.Hour)

	campaignIDs, err := forecastCampaignIDs(clickhouseCtx, host, in.CustomerID)
	if err != nil {
		return CampaignForecastDTO{}, err
	}

	totalSample, hourlySamples, err := queryForecastHourlySamples(clickhouseCtx, host.ForecastClickHouseQuery(), lookbackStart, lookbackEnd, campaignIDs)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(clickhouseCtx.Err(), context.DeadlineExceeded) {
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

func forecastCampaignIDs(ctx context.Context, host ForecastHost, customerID *uuid.UUID) ([]uuid.UUID, error) {
	if customerID == nil || *customerID == uuid.Nil || host.ForecastPool() == nil {
		return nil, nil
	}
	q := db.New(host.ForecastPool())
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

func queryForecastHourlySamples(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, from, to time.Time, campaignIDs []uuid.UUID) (uint64, []forecastHourlySample, error) {
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

	rows, err := clickhouseQuery.Query(ctx, query, args...)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = rows.Close() }()

	var total uint64
	samples := make([]forecastHourlySample, 0, 24)
	for rows.Next() {
		var sample forecastHourlySample
		if err := rows.Scan(&sample.HourOfDay, &sample.Impressions); err != nil {
			return 0, nil, err
		}
		total += sample.Impressions
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return total, samples, nil
}
