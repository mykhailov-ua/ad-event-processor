package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"espx/internal/database"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/pkg/coldpath"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CampaignDTO struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Status          string          `json:"status"`
	BudgetLimit     string          `json:"budget_limit"`
	CurrentSpend    string          `json:"current_spend"`
	CustomerID      string          `json:"customer_id"`
	PacingMode      string          `json:"pacing_mode"`
	DailyBudget     string          `json:"daily_budget"`
	Timezone        string          `json:"timezone"`
	FreqLimit       int32           `json:"freq_limit"`
	FreqWindow      int32           `json:"freq_window"`
	TargetCountries []string        `json:"target_countries"`
	TargetURL       string          `json:"target_url,omitempty"`
	CreativePayload json.RawMessage `json:"creative_payload,omitempty"`
	ReferrerFilter  string          `json:"referrer_filter,omitempty"`
	StartAt         string          `json:"start_at,omitempty"`
	EndAt           string          `json:"end_at,omitempty"`
	DaypartHours    []int16         `json:"daypart_hours"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

type StatusHistoryDTO struct {
	ID         int64  `json:"id"`
	CampaignID string `json:"campaign_id"`
	OldStatus  string `json:"old_status,omitempty"`
	NewStatus  string `json:"new_status"`
	Reason     string `json:"reason,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func statusHistoryToDTO(r db.CampaignStatusHistory) StatusHistoryDTO {
	var oldStatus string
	if r.OldStatus.Valid {
		oldStatus = string(r.OldStatus.CampaignStatusType)
	}
	return StatusHistoryDTO{
		ID:         r.ID,
		CampaignID: uuid.UUID(r.CampaignID.Bytes).String(),
		OldStatus:  oldStatus,
		NewStatus:  string(r.NewStatus),
		Reason:     r.Reason.String,
		CreatedAt:  r.CreatedAt.Time.Format(time.RFC3339),
	}
}

func toCampaignDTO(c db.Campaign) CampaignDTO {
	countries := c.TargetCountries
	if countries == nil {
		countries = []string{}
	}

	return CampaignDTO{
		ID:              uuid.UUID(c.ID.Bytes).String(),
		Name:            c.Name,
		Status:          string(c.Status),
		BudgetLimit:     formatMicro(c.BudgetLimit),
		CurrentSpend:    formatMicro(c.CurrentSpend),
		CustomerID:      uuid.UUID(c.CustomerID.Bytes).String(),
		PacingMode:      string(c.PacingMode),
		DailyBudget:     formatMicro(c.DailyBudget),
		Timezone:        c.Timezone,
		FreqLimit:       c.FreqLimit.Int32,
		FreqWindow:      c.FreqWindow.Int32,
		TargetCountries: countries,
		TargetURL:       c.TargetUrl,
		CreativePayload: json.RawMessage(c.CreativePayload),
		ReferrerFilter:  c.ReferrerFilter,
		StartAt:         formatOptionalTime(c.StartAt),
		EndAt:           formatOptionalTime(c.EndAt),
		DaypartHours:    daypartOrEmpty(c.DaypartHours),
		CreatedAt:       c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       c.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func formatOptionalTime(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func daypartOrEmpty(h []int16) []int16 {
	if h == nil {
		return []int16{}
	}
	return h
}

func (s *Service) ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]CampaignDTO, int64, error) {
	q := db.New(s.GetPool())

	var cid pgtype.UUID
	if customerID != uuid.Nil {
		cid = domain.ToUUID(customerID)
	}

	var st pgtype.Text
	if status != "" {
		st = pgtype.Text{String: status, Valid: true}
	}

	countParams := db.CountCampaignsParams{
		CustomerID: cid,
		Status:     st,
	}
	listParams := db.ListCampaignsParams{
		Limit:      limit,
		Offset:     offset,
		CustomerID: cid,
		Status:     st,
	}

	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountCampaigns(ctx, countParams) },
		func() ([]db.Campaign, error) { return q.ListCampaigns(ctx, listParams) },
		func(c db.Campaign) CampaignDTO { return scrubCampaignDTO(ctx, toCampaignDTO(c)) },
	)
}

func (s *Service) GetCampaignDTO(ctx context.Context, id uuid.UUID) (CampaignDTO, error) {
	q := db.New(s.GetPool())
	c, err := q.GetCampaign(ctx, domain.ToUUID(id))
	if err != nil {
		return CampaignDTO{}, mapNotFound(err, ErrCampaignNotFound)
	}
	return scrubCampaignDTO(ctx, toCampaignDTO(c)), nil
}

func (s *Service) ListStatusHistory(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]StatusHistoryDTO, int64, error) {
	q := db.New(s.GetPool())
	cid := domain.ToUUID(campaignID)

	listParams := db.ListStatusHistoryParams{
		CampaignID: cid,
		Limit:      limit,
		Offset:     offset,
	}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountStatusHistory(ctx, cid) },
		func() ([]db.CampaignStatusHistory, error) { return q.ListStatusHistory(ctx, listParams) },
		statusHistoryToDTO,
	)
}

func (s *Service) UpdateCampaignPacing(ctx context.Context, campaignID uuid.UUID, newMode string) (CampaignDTO, error) {
	var pacing db.PacingModeType
	switch newMode {
	case "ASAP":
		pacing = db.PacingModeTypeASAP
	case "EVEN", "off", "OFF":
		pacing = db.PacingModeTypeEVEN
	case "VPP", "vpp":
		pacing = db.PacingModeTypeVPP
	default:
		return CampaignDTO{}, fmt.Errorf("%w: %s", ErrInvalidPacingMode, newMode)
	}

	var updatedCamp db.Campaign
	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)

		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapNotFound(err, ErrCampaignNotFound)
		}

		updatedCamp, err = q.UpdateCampaignPacing(ctx, db.UpdateCampaignPacingParams{
			ID:         domain.ToUUID(campaignID),
			PacingMode: pacing,
		})
		if err != nil {
			return fmt.Errorf("failed to update campaign pacing: %w", err)
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}

		s.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_PACING", "campaign", &campaignID, map[string]any{
			"old_pacing_mode": string(camp.PacingMode),
			"new_pacing_mode": string(pacing),
		}, nil)

		payloadBytes, err := coldpath.MarshalJSON(map[string]any{
			"campaign_id": campaignID.String(),
			"pacing_mode": string(pacing),
		})
		if err != nil {
			return fmt.Errorf("marshal update campaign pacing outbox payload: %w", err)
		}

		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_CAMPAIGN_PACING",
			Payload:   payloadBytes,
		})
		if err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}

		return nil
	})

	if err != nil {
		return CampaignDTO{}, err
	}

	return toCampaignDTO(updatedCamp), nil
}

const clickHouseStaleThreshold = 5 * time.Minute

type CampaignMetricsDTO struct {
	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	Conversions int64 `json:"conversions"`
}

type CampaignHourlyBucketDTO struct {
	Hour        string `json:"hour"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
	Conversions int64  `json:"conversions"`
}

type CampaignStatsDTO struct {
	CampaignID   string                    `json:"campaign_id"`
	CurrentSpend string                    `json:"current_spend"`
	Metrics      CampaignMetricsDTO        `json:"metrics"`
	Hourly       []CampaignHourlyBucketDTO `json:"hourly"`
	Granularity  string                    `json:"granularity"`
	From         string                    `json:"from"`
	To           string                    `json:"to"`
	Stale        bool                      `json:"stale"`
	Source       string                    `json:"source"`
	Consistency  string                    `json:"consistency"`
}

func (s *Service) SetClickHouse(conn driver.Conn, cfg database.CHQueryConfig) {
	if conn != nil {
		s.chQuery = database.NewCHQuery(conn, cfg)
	}
}

func (s *Service) SetClickHouseWrite(conn driver.Conn) {
	s.chWrite = conn
}

func (s *Service) GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (CampaignStatsDTO, error) {
	if granularity != "hour" {
		return CampaignStatsDTO{}, fmt.Errorf("%w: %s", ErrUnsupportedGranularity, granularity)
	}
	if !to.After(from) {
		return CampaignStatsDTO{}, fmt.Errorf("%w: to must be after from", ErrInvalidTimeRange)
	}

	q := db.New(s.GetPool())
	camp, err := q.GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return CampaignStatsDTO{}, mapNotFound(err, ErrCampaignNotFound)
	}

	stats, err := q.SumCampaignStatsInRange(ctx, db.SumCampaignStatsInRangeParams{
		CampaignID: domain.ToUUID(campaignID),
		FromDate:   pgtype.Date{Time: from.UTC(), Valid: true},
		ToDate:     pgtype.Date{Time: to.UTC(), Valid: true},
	})
	if err != nil {
		return CampaignStatsDTO{}, err
	}

	report := CampaignStatsDTO{
		CampaignID:   campaignID.String(),
		CurrentSpend: formatMicro(camp.CurrentSpend),
		Metrics: CampaignMetricsDTO{
			Impressions: stats.Impressions,
			Clicks:      stats.Clicks,
			Conversions: stats.Conversions,
		},
		Hourly:      []CampaignHourlyBucketDTO{},
		Granularity: granularity,
		From:        from.UTC().Format(time.RFC3339),
		To:          to.UTC().Format(time.RFC3339),
		Stale:       true,
		Source:      "pg",
		Consistency: "strong",
	}

	if s.chQuery == nil {
		return report, nil
	}

	hourly, lag, err := s.queryClickHouseHourly(ctx, campaignID, from, to)
	if err != nil {
		return CampaignStatsDTO{}, err
	}
	report.Hourly = hourly
	report.Consistency = "eventual"
	report.Source = "ch"
	report.Stale = lag > clickHouseStaleThreshold
	return report, nil
}

func (s *Service) queryClickHouseHourly(ctx context.Context, campaignID uuid.UUID, from, to time.Time) ([]CampaignHourlyBucketDTO, time.Duration, error) {
	if s.chQuery == nil {
		return nil, 0, nil
	}
	type row struct {
		hour        time.Time
		impressions uint64
		clicks      uint64
		conversions uint64
	}

	query := `
SELECT
    hour,
    sum(impressions) AS impressions,
    sum(clicks) AS clicks,
    sum(conversions) AS conversions
FROM (
    SELECT hour, impression_count AS impressions, toUInt64(0) AS clicks, toUInt64(0) AS conversions
    FROM mv_campaign_hourly_impressions
    WHERE campaign_id = ? AND hour >= ? AND hour < ?
    UNION ALL
    SELECT hour, toUInt64(0), click_count, toUInt64(0)
    FROM mv_campaign_hourly_clicks
    WHERE campaign_id = ? AND hour >= ? AND hour < ?
    UNION ALL
    SELECT hour, toUInt64(0), toUInt64(0), conversion_count
    FROM mv_campaign_hourly_conversions
    WHERE campaign_id = ? AND hour >= ? AND hour < ?
)
GROUP BY hour
ORDER BY hour`

	rows, err := s.chQuery.Query(ctx, query,
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("clickhouse hourly query: %w", err)
	}
	defer rows.Close()

	buckets := make([]CampaignHourlyBucketDTO, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.hour, &r.impressions, &r.clicks, &r.conversions); err != nil {
			return nil, 0, fmt.Errorf("clickhouse hourly scan: %w", err)
		}
		buckets = append(buckets, CampaignHourlyBucketDTO{
			Hour:        r.hour.UTC().Format(time.RFC3339),
			Impressions: int64(r.impressions),
			Clicks:      int64(r.clicks),
			Conversions: int64(r.conversions),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	lag, err := s.clickHouseIngestionLag(ctx)
	if err != nil {
		return nil, 0, err
	}
	return buckets, lag, nil
}

func (s *Service) clickHouseIngestionLag(ctx context.Context) (time.Duration, error) {
	if s.chQuery == nil {
		return 0, nil
	}
	var latest time.Time
	err := s.chQuery.QueryRow(ctx, `
SELECT max(latest) FROM (
    SELECT max(created_at) AS latest FROM impressions
    UNION ALL
    SELECT max(created_at) FROM clicks
    UNION ALL
    SELECT max(created_at) FROM conversions
)`).Scan(&latest)
	if err != nil {
		return 0, fmt.Errorf("clickhouse lag probe: %w", err)
	}
	if latest.IsZero() {
		return clickHouseStaleThreshold + time.Second, nil
	}
	return time.Since(latest.UTC()), nil
}
