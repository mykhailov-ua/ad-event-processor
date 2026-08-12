package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/controlplane/authz"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/money"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type CampaignDTO = adminapi.CampaignDTO

type StatusHistoryDTO struct {
	ID         int64  `json:"id"`
	CampaignID string `json:"campaign_id"`
	OldStatus  string `json:"old_status,omitempty"`
	NewStatus  string `json:"new_status"`
	Reason     string `json:"reason,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func formatOptionalTime(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func formatOptionalUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
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

	total, err := q.CountCampaigns(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []CampaignDTO{}, 0, nil
	}
	rows, err := q.ListCampaigns(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CampaignDTO, 0, len(rows))
	for _, c := range rows {
		out = append(out, scrubCampaignDTO(ctx, c))
	}
	return out, total, nil
}

func (s *Service) GetCampaign(ctx context.Context, id uuid.UUID) (CampaignDTO, error) {
	q := db.New(s.GetPool())
	c, err := q.GetCampaign(ctx, domain.ToUUID(id))
	if err != nil {
		return CampaignDTO{}, mapNotFound(err, ErrCampaignNotFound)
	}
	return scrubCampaignDTO(ctx, c), nil
}

func (s *Service) PatchCampaign(ctx context.Context, campaignID uuid.UUID, req adminapi.PatchCampaignRequest) (CampaignDTO, error) {
	if req.PacingMode != nil {
		if _, err := s.UpdateCampaignPacing(ctx, campaignID, *req.PacingMode); err != nil {
			return CampaignDTO{}, err
		}
	}

	adminPatch := req.Name != nil || req.DailyBudgetMicro != nil || req.Timezone != nil ||
		req.FreqLimit != nil || req.FreqWindow != nil || req.TargetCountries != nil ||
		req.TargetURL != nil || req.ReferrerFilter != nil ||
		req.SafePageURL != nil || req.SafePageEnabled != nil
	if !adminPatch {
		return s.GetCampaign(ctx, campaignID)
	}

	var updated db.Campaign
	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapNotFound(err, ErrCampaignNotFound)
		}

		name := camp.Name
		if req.Name != nil {
			name = strings.TrimSpace(*req.Name)
			if name == "" {
				return fmt.Errorf("name is required")
			}
		}
		dailyBudget := camp.DailyBudget
		if req.DailyBudgetMicro != nil {
			if *req.DailyBudgetMicro < 0 {
				return fmt.Errorf("invalid daily_budget")
			}
			dailyBudget = *req.DailyBudgetMicro
		}
		timezone := camp.Timezone
		if req.Timezone != nil {
			timezone = strings.TrimSpace(*req.Timezone)
			if timezone == "" {
				timezone = "UTC"
			}
		}
		freqLimit := camp.FreqLimit
		if req.FreqLimit != nil {
			freqLimit = pgtype.Int4{Int32: *req.FreqLimit, Valid: true}
		}
		freqWindow := camp.FreqWindow
		if req.FreqWindow != nil {
			freqWindow = pgtype.Int4{Int32: *req.FreqWindow, Valid: true}
		}
		countries := camp.TargetCountries
		if req.TargetCountries != nil {
			countries = countriesOrEmpty(req.TargetCountries)
		}
		targetURL := camp.TargetUrl
		if req.TargetURL != nil {
			targetURL = *req.TargetURL
		}
		referrerFilter := camp.ReferrerFilter
		if req.ReferrerFilter != nil {
			referrerFilter = *req.ReferrerFilter
		}
		safePageURL := camp.SafePageUrl
		if req.SafePageURL != nil {
			safePageURL = *req.SafePageURL
		}
		safePageEnabled := camp.SafePageEnabled
		if req.SafePageEnabled != nil {
			safePageEnabled = *req.SafePageEnabled
		}

		updated, err = q.UpdateCampaignAdmin(ctx, db.UpdateCampaignAdminParams{
			ID:              domain.ToUUID(campaignID),
			Name:            name,
			DailyBudget:     dailyBudget,
			Timezone:        timezone,
			FreqLimit:       freqLimit,
			FreqWindow:      freqWindow,
			TargetCountries: countries,
			TargetUrl:       targetURL,
			ReferrerFilter:  referrerFilter,
			SafePageUrl:     safePageURL,
			SafePageEnabled: safePageEnabled,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, auditCampaignAdminChange{
			Name:            name,
			DailyBudget:     dailyBudget,
			Timezone:        timezone,
			TargetCountries: countries,
		}, nil)

		return nil
	})
	if err != nil {
		return CampaignDTO{}, err
	}

	if pubErr := s.publishCampaignUpdate(ctx, campaignID.String()); pubErr != nil {
		slog.Warn("campaign update publish failed after patch", "campaign_id", campaignID, "err", pubErr)
	}
	return scrubCampaignDTO(ctx, updated), nil
}

func (s *Service) ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]adminapi.CampaignEventDTO, int64, error) {
	q := db.New(s.GetPool())
	cid := domain.ToUUID(campaignID)
	total, err := q.CountCampaignEvents(ctx, cid)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []adminapi.CampaignEventDTO{}, 0, nil
	}
	rows, err := q.ListCampaignEvents(ctx, db.ListCampaignEventsParams{
		CampaignID: cid,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]adminapi.CampaignEventDTO, 0, len(rows))
	for _, row := range rows {
		var ip, ua, userID string
		if row.IpAddress.Valid {
			ip = row.IpAddress.String
		}
		if row.UserAgent.Valid {
			ua = row.UserAgent.String
		}
		if row.UserID.Valid {
			userID = row.UserID.String
		}
		createdAt := ""
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
		out = append(out, adminapi.CampaignEventDTO{
			ClickID:   row.ClickID,
			EventType: row.EventType,
			UserID:    userID,
			IP:        ip,
			UserAgent: ua,
			Payload:   json.RawMessage(row.Payload),
			CreatedAt: createdAt,
		})
	}
	return out, total, nil
}

func (s *Service) ListStatusHistory(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]StatusHistoryDTO, int64, error) {
	q := db.New(s.GetPool())
	cid := domain.ToUUID(campaignID)

	listParams := db.ListStatusHistoryParams{
		CampaignID: cid,
		Limit:      limit,
		Offset:     offset,
	}
	total, err := q.CountStatusHistory(ctx, cid)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []StatusHistoryDTO{}, 0, nil
	}
	historyRows, err := q.ListStatusHistory(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	out := make([]StatusHistoryDTO, 0, len(historyRows))
	for _, r := range historyRows {
		var oldStatus string
		if r.OldStatus.Valid {
			oldStatus = string(r.OldStatus.CampaignStatusType)
		}
		out = append(out, StatusHistoryDTO{
			ID:         r.ID,
			CampaignID: uuid.UUID(r.CampaignID.Bytes).String(),
			OldStatus:  oldStatus,
			NewStatus:  string(r.NewStatus),
			Reason:     r.Reason.String,
			CreatedAt:  r.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	return out, total, nil
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

		s.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_PACING", "campaign", &campaignID, auditPacingChange{
			OldPacingMode: string(camp.PacingMode),
			NewPacingMode: string(pacing),
		}, nil)

		payloadBytes, err := coldpath.MarshalOutbox(campaignPacingPayload{
			CampaignID: campaignID.String(),
			PacingMode: string(pacing),
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

	return scrubCampaignDTO(ctx, updatedCamp), nil
}

const clickHouseStaleThreshold = 5 * time.Minute

type CampaignMetricsDTO = adminapi.CampaignMetricsDTO
type CampaignHourlyBucketDTO = adminapi.CampaignHourlyBucketDTO
type CampaignStatsDTO = adminapi.CampaignStatsDTO

func scrubCampaignFields(c CampaignDTO, level authz.MaskLevel) CampaignDTO {
	if level == authz.MaskFull {
		return c
	}
	out := c
	out.TargetURL = ""
	out.CreativePayload = nil
	out.ReferrerFilter = ""
	return out
}

func scrubCampaignDTO(ctx context.Context, c db.Campaign) CampaignDTO {
	countries := c.TargetCountries
	if countries == nil {
		countries = []string{}
	}
	dto := CampaignDTO{
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
		SafePageURL:     c.SafePageUrl,
		SafePageEnabled: c.SafePageEnabled,
		BrandID:         formatOptionalUUID(c.BrandID),
		CreativePayload: json.RawMessage(c.CreativePayload),
		ReferrerFilter:  c.ReferrerFilter,
		StartAt:         formatOptionalTime(c.StartAt),
		EndAt:           formatOptionalTime(c.EndAt),
		DaypartHours:    daypartOrEmpty(c.DaypartHours),
		CreatedAt:       c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       c.UpdatedAt.Time.Format(time.RFC3339),
	}
	if snap, ok := authz.SnapshotFromContext(ctx); ok {
		return scrubCampaignFields(dto, snap.Mask)
	}
	return dto
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

	const chStatsTimeout = 10 * time.Second
	chCtx, cancel := context.WithTimeout(ctx, chStatsTimeout)
	defer cancel()

	hourly, lag, err := s.queryClickHouseHourly(chCtx, campaignID, from, to)
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

type chLagCache struct {
	mu      sync.Mutex
	lag     time.Duration
	updated time.Time
}

const chLagCacheTTL = 30 * time.Second

var globalCHLagCache chLagCache

func (s *Service) clickHouseIngestionLag(ctx context.Context) (time.Duration, error) {
	if s.chQuery == nil {
		return 0, nil
	}

	globalCHLagCache.mu.Lock()
	if time.Since(globalCHLagCache.updated) < chLagCacheTTL {
		lag := globalCHLagCache.lag
		globalCHLagCache.mu.Unlock()
		return lag, nil
	}
	globalCHLagCache.mu.Unlock()

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
	var lag time.Duration
	if latest.IsZero() {
		lag = clickHouseStaleThreshold + time.Second
	} else {
		lag = time.Since(latest.UTC())
	}

	globalCHLagCache.mu.Lock()
	globalCHLagCache.lag = lag
	globalCHLagCache.updated = time.Now()
	globalCHLagCache.mu.Unlock()

	return lag, nil
}

type BrandDTO struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	FreqLimit  int32  `json:"freq_limit"`
	FreqWindow int32  `json:"freq_window"`
}

func (s *Service) CreateBrand(ctx context.Context, customerID uuid.UUID, name string) (uuid.UUID, error) {
	brandID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	q := db.New(s.GetPool())
	_, err = q.GetCustomerByID(ctx, domain.ToUUID(customerID))
	if err != nil {
		return uuid.Nil, mapNotFound(err, ErrCustomerNotFound)
	}

	_, err = q.CreateBrand(ctx, db.CreateBrandParams{
		ID:         domain.ToUUID(brandID),
		CustomerID: domain.ToUUID(customerID),
		Name:       name,
	})
	if err != nil {
		return uuid.Nil, err
	}

	return brandID, nil
}

func (s *Service) GetBrandDTO(ctx context.Context, id uuid.UUID) (BrandDTO, error) {
	q := db.New(s.GetPool())
	b, err := q.GetBrand(ctx, domain.ToUUID(id))
	if err != nil {
		return BrandDTO{}, mapNotFound(err, ErrBrandNotFound)
	}
	return BrandDTO{
		ID:         uuid.UUID(b.ID.Bytes).String(),
		CustomerID: uuid.UUID(b.CustomerID.Bytes).String(),
		Name:       b.Name,
		CreatedAt:  b.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  b.UpdatedAt.Time.Format(time.RFC3339),
		FreqLimit:  b.FreqLimit,
		FreqWindow: b.FreqWindow,
	}, nil
}

func (s *Service) ListBrandsByCustomer(ctx context.Context, customerID uuid.UUID) ([]BrandDTO, error) {
	q := db.New(s.GetPool())
	rows, err := q.ListBrandsByCustomer(ctx, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	out := make([]BrandDTO, len(rows))
	for i, b := range rows {
		out[i] = BrandDTO{
			ID:         uuid.UUID(b.ID.Bytes).String(),
			CustomerID: uuid.UUID(b.CustomerID.Bytes).String(),
			Name:       b.Name,
			CreatedAt:  b.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:  b.UpdatedAt.Time.Format(time.RFC3339),
			FreqLimit:  b.FreqLimit,
			FreqWindow: b.FreqWindow,
		}
	}
	return out, nil
}

func (s *Service) ConfigureBrandFcap(ctx context.Context, brandID uuid.UUID, limit, window int32) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)

		brand, err := q.GetBrandForUpdate(ctx, domain.ToUUID(brandID))
		if err != nil {
			return mapNotFound(err, ErrBrandNotFound)
		}

		err = q.ConfigureBrandFcap(ctx, db.ConfigureBrandFcapParams{
			ID:         domain.ToUUID(brandID),
			FreqLimit:  limit,
			FreqWindow: window,
		})
		if err != nil {
			return fmt.Errorf("failed to update brand fcap limits: %w", err)
		}

		payloadBytes, err := coldpath.MarshalOutbox(brandFcapOutboxPayload{
			BrandID:    brandID.String(),
			FreqLimit:  limit,
			FreqWindow: window,
		})
		if err != nil {
			return fmt.Errorf("marshal configure brand fcap outbox payload: %w", err)
		}

		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "CONFIGURE_BRAND_FCAP",
			Payload:   payloadBytes,
		})
		if err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}

		s.AuditLog(ctx, q, uuid.Nil, "CONFIGURE_BRAND_FCAP", "brand", &brandID, auditBrandFcapChange{
			OldFreqLimit:  brand.FreqLimit,
			OldFreqWindow: brand.FreqWindow,
			NewFreqLimit:  limit,
			NewFreqWindow: window,
		}, nil)

		return nil
	})
}

const (
	maxSupplyChainHops   = 10
	sellersJSONCacheTTL  = 60 * time.Second
	sellersJSONVersion   = "1.0"
	supplySettingOwner   = "supply_owner_domain"
	supplySettingManager = "supply_manager_domain"
	supplySettingContact = "supply_contact_email"
)

var (
	ErrSellerNotFound      = errors.New("seller not found")
	ErrAdsTxtEntryNotFound = errors.New("ads.txt entry not found")
	ErrInvalidSellerType   = errors.New("seller_type must be PUBLISHER, INTERMEDIARY, or BOTH")
	ErrInvalidRelationship = errors.New("relationship must be DIRECT or RESELLER")
	ErrSupplyChainTooLong  = fmt.Errorf("supply chain exceeds %d hops", maxSupplyChainHops)
	ErrSellersJSONInvalid  = errors.New("sellers.json schema validation failed")
)

type SupplyChainNode struct {
	ASI string `json:"asi"`
	SID string `json:"sid"`
	RID string `json:"rid,omitempty"`
	HP  int    `json:"hp"`
}

type SellerDTO struct {
	ID             int64  `json:"id"`
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type SellerCreateSpec struct {
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
}

type SellerUpdateSpec struct {
	SellerID       string `json:"seller_id"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	Name           string `json:"name"`
	IsConfidential bool   `json:"is_confidential"`
}

type AdsTxtEntryDTO struct {
	ID                 int64  `json:"id"`
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id,omitempty"`
	SortOrder          int32  `json:"sort_order"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type AdsTxtEntryCreateSpec struct {
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id"`
	SortOrder          int32  `json:"sort_order"`
}

type AdsTxtEntryUpdateSpec struct {
	Domain             string `json:"domain"`
	PublisherAccountID string `json:"publisher_account_id"`
	Relationship       string `json:"relationship"`
	CertAuthorityID    string `json:"cert_authority_id"`
	SortOrder          int32  `json:"sort_order"`
}

type CampaignSupplyChainDTO struct {
	CampaignID string            `json:"campaign_id"`
	Nodes      []SupplyChainNode `json:"nodes"`
}

type SupplyFilesPayload struct {
	Trigger string `json:"trigger"`
}

type sellersJSONCacheEntry struct {
	body    []byte
	expires time.Time
}

type sellersJSONCache struct {
	mu sync.RWMutex
	v  sellersJSONCacheEntry
}

var sellersCache sellersJSONCache

func normalizeSellerType(v string) (string, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case "PUBLISHER", "INTERMEDIARY", "BOTH":
		return v, nil
	default:
		return "", ErrInvalidSellerType
	}
}

func normalizeRelationship(v string) (string, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case "DIRECT", "RESELLER":
		return v, nil
	default:
		return "", ErrInvalidRelationship
	}
}

func validateSupplyChainNodes(nodes []SupplyChainNode) error {
	if len(nodes) > maxSupplyChainHops {
		return ErrSupplyChainTooLong
	}
	for i, n := range nodes {
		if strings.TrimSpace(n.ASI) == "" || strings.TrimSpace(n.SID) == "" {
			return errValidation(fmt.Sprintf("supply chain node %d: asi and sid are required", i))
		}
		if n.HP != 0 && n.HP != 1 {
			return errValidation(fmt.Sprintf("supply chain node %d: hp must be 0 or 1", i))
		}
	}
	return nil
}

func (s *Service) enqueueSupplyFilesUpdate(ctx context.Context, q db.Querier, trigger string) error {
	invalidateSellersJSONCache()
	payload, err := coldpath.MarshalOutbox(SupplyFilesPayload{Trigger: trigger})
	if err != nil {
		return err
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "UPDATE_SUPPLY_FILES",
		Payload:   payload,
	})
	return err
}

func invalidateSellersJSONCache() {
	sellersCache.mu.Lock()
	sellersCache.v = sellersJSONCacheEntry{}
	sellersCache.mu.Unlock()
}

func (s *Service) ListSellers(ctx context.Context) ([]SellerDTO, error) {
	rows, err := db.New(s.GetPool()).ListSellers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SellerDTO, len(rows))
	for i, r := range rows {
		out[i] = SellerDTO{
			ID:             r.ID,
			SellerID:       r.SellerID,
			Domain:         r.Domain,
			SellerType:     r.SellerType,
			Name:           r.Name,
			IsConfidential: r.IsConfidential,
			CreatedAt:      r.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:      r.UpdatedAt.Time.Format(time.RFC3339),
		}
	}
	return out, nil
}

func (s *Service) GetSeller(ctx context.Context, id int64) (SellerDTO, error) {
	row, err := db.New(s.GetPool()).GetSeller(ctx, id)
	if err != nil {
		return SellerDTO{}, ErrSellerNotFound
	}
	return SellerDTO{
		ID:             row.ID,
		SellerID:       row.SellerID,
		Domain:         row.Domain,
		SellerType:     row.SellerType,
		Name:           row.Name,
		IsConfidential: row.IsConfidential,
		CreatedAt:      row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *Service) CreateSeller(ctx context.Context, spec SellerCreateSpec) (SellerDTO, error) {
	sellerType, err := normalizeSellerType(spec.SellerType)
	if err != nil {
		return SellerDTO{}, err
	}
	if strings.TrimSpace(spec.SellerID) == "" || strings.TrimSpace(spec.Domain) == "" {
		return SellerDTO{}, errValidation("seller_id and domain are required")
	}

	var out SellerDTO
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.CreateSeller(ctx, db.CreateSellerParams{
			SellerID:       strings.TrimSpace(spec.SellerID),
			Domain:         strings.TrimSpace(spec.Domain),
			SellerType:     sellerType,
			Name:           strings.TrimSpace(spec.Name),
			IsConfidential: spec.IsConfidential,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "CREATE_SELLER", "supply", nil, auditSellerCreateChange{
			SellerID: row.SellerID,
			Domain:   row.Domain,
		}, nil)

		if err := s.enqueueSupplyFilesUpdate(ctx, q, "create_seller"); err != nil {
			return err
		}
		out = SellerDTO{
			ID:             row.ID,
			SellerID:       row.SellerID,
			Domain:         row.Domain,
			SellerType:     row.SellerType,
			Name:           row.Name,
			IsConfidential: row.IsConfidential,
			CreatedAt:      row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:      row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (s *Service) UpdateSeller(ctx context.Context, id int64, spec SellerUpdateSpec) (SellerDTO, error) {
	sellerType, err := normalizeSellerType(spec.SellerType)
	if err != nil {
		return SellerDTO{}, err
	}
	if strings.TrimSpace(spec.SellerID) == "" || strings.TrimSpace(spec.Domain) == "" {
		return SellerDTO{}, errValidation("seller_id and domain are required")
	}

	var out SellerDTO
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.UpdateSeller(ctx, db.UpdateSellerParams{
			ID:             id,
			SellerID:       strings.TrimSpace(spec.SellerID),
			Domain:         strings.TrimSpace(spec.Domain),
			SellerType:     sellerType,
			Name:           strings.TrimSpace(spec.Name),
			IsConfidential: spec.IsConfidential,
		})
		if err != nil {
			return ErrSellerNotFound
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_SELLER", "supply", nil, auditSellerUpdateChange{
			ID:       id,
			SellerID: row.SellerID,
		}, nil)

		if err := s.enqueueSupplyFilesUpdate(ctx, q, "update_seller"); err != nil {
			return err
		}
		out = SellerDTO{
			ID:             row.ID,
			SellerID:       row.SellerID,
			Domain:         row.Domain,
			SellerType:     row.SellerType,
			Name:           row.Name,
			IsConfidential: row.IsConfidential,
			CreatedAt:      row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:      row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (s *Service) DeleteSeller(ctx context.Context, id int64) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetSeller(ctx, id); err != nil {
			return ErrSellerNotFound
		}
		if err := q.DeleteSeller(ctx, id); err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "DELETE_SELLER", "supply", nil, auditIdChange{ID: id}, nil)
		return s.enqueueSupplyFilesUpdate(ctx, q, "delete_seller")
	})
}

func (s *Service) ListAdsTxtEntries(ctx context.Context) ([]AdsTxtEntryDTO, error) {
	rows, err := db.New(s.GetPool()).ListAdsTxtEntries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdsTxtEntryDTO, len(rows))
	for i, r := range rows {
		out[i] = AdsTxtEntryDTO{
			ID:                 r.ID,
			Domain:             r.Domain,
			PublisherAccountID: r.PublisherAccountID,
			Relationship:       r.Relationship,
			CertAuthorityID:    r.CertAuthorityID,
			SortOrder:          r.SortOrder,
			CreatedAt:          r.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:          r.UpdatedAt.Time.Format(time.RFC3339),
		}
	}
	return out, nil
}

func (s *Service) GetAdsTxtEntry(ctx context.Context, id int64) (AdsTxtEntryDTO, error) {
	row, err := db.New(s.GetPool()).GetAdsTxtEntry(ctx, id)
	if err != nil {
		return AdsTxtEntryDTO{}, ErrAdsTxtEntryNotFound
	}
	return AdsTxtEntryDTO{
		ID:                 row.ID,
		Domain:             row.Domain,
		PublisherAccountID: row.PublisherAccountID,
		Relationship:       row.Relationship,
		CertAuthorityID:    row.CertAuthorityID,
		SortOrder:          row.SortOrder,
		CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:          row.UpdatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *Service) CreateAdsTxtEntry(ctx context.Context, spec AdsTxtEntryCreateSpec) (AdsTxtEntryDTO, error) {
	rel, err := normalizeRelationship(spec.Relationship)
	if err != nil {
		return AdsTxtEntryDTO{}, err
	}
	if strings.TrimSpace(spec.Domain) == "" || strings.TrimSpace(spec.PublisherAccountID) == "" {
		return AdsTxtEntryDTO{}, errValidation("domain and publisher_account_id are required")
	}

	var out AdsTxtEntryDTO
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.CreateAdsTxtEntry(ctx, db.CreateAdsTxtEntryParams{
			Domain:             strings.TrimSpace(spec.Domain),
			PublisherAccountID: strings.TrimSpace(spec.PublisherAccountID),
			Relationship:       rel,
			CertAuthorityID:    strings.TrimSpace(spec.CertAuthorityID),
			SortOrder:          spec.SortOrder,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "CREATE_ADS_TXT", "supply", nil, auditAdsTxtDomainChange{
			Domain: spec.Domain,
		}, nil)

		if err := s.enqueueSupplyFilesUpdate(ctx, q, "create_ads_txt"); err != nil {
			return err
		}
		out = AdsTxtEntryDTO{
			ID:                 row.ID,
			Domain:             row.Domain,
			PublisherAccountID: row.PublisherAccountID,
			Relationship:       row.Relationship,
			CertAuthorityID:    row.CertAuthorityID,
			SortOrder:          row.SortOrder,
			CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:          row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (s *Service) UpdateAdsTxtEntry(ctx context.Context, id int64, spec AdsTxtEntryUpdateSpec) (AdsTxtEntryDTO, error) {
	rel, err := normalizeRelationship(spec.Relationship)
	if err != nil {
		return AdsTxtEntryDTO{}, err
	}
	if strings.TrimSpace(spec.Domain) == "" || strings.TrimSpace(spec.PublisherAccountID) == "" {
		return AdsTxtEntryDTO{}, errValidation("domain and publisher_account_id are required")
	}

	var out AdsTxtEntryDTO
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.UpdateAdsTxtEntry(ctx, db.UpdateAdsTxtEntryParams{
			ID:                 id,
			Domain:             strings.TrimSpace(spec.Domain),
			PublisherAccountID: strings.TrimSpace(spec.PublisherAccountID),
			Relationship:       rel,
			CertAuthorityID:    strings.TrimSpace(spec.CertAuthorityID),
			SortOrder:          spec.SortOrder,
		})
		if err != nil {
			return ErrAdsTxtEntryNotFound
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_ADS_TXT", "supply", nil, auditIdChange{ID: id}, nil)

		if err := s.enqueueSupplyFilesUpdate(ctx, q, "update_ads_txt"); err != nil {
			return err
		}
		out = AdsTxtEntryDTO{
			ID:                 row.ID,
			Domain:             row.Domain,
			PublisherAccountID: row.PublisherAccountID,
			Relationship:       row.Relationship,
			CertAuthorityID:    row.CertAuthorityID,
			SortOrder:          row.SortOrder,
			CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:          row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (s *Service) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetAdsTxtEntry(ctx, id); err != nil {
			return ErrAdsTxtEntryNotFound
		}
		if err := q.DeleteAdsTxtEntry(ctx, id); err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "DELETE_ADS_TXT", "supply", nil, auditIdChange{ID: id}, nil)
		return s.enqueueSupplyFilesUpdate(ctx, q, "delete_ads_txt")
	})
}

func (s *Service) GetCampaignSupplyChain(ctx context.Context, campaignID uuid.UUID) (CampaignSupplyChainDTO, error) {
	row, err := db.New(s.GetPool()).GetCampaignFull(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return CampaignSupplyChainDTO{}, mapNotFound(err, ErrCampaignNotFound)
	}
	nodes, err := parseSupplyChainNodes(row.SupplyChainNodes)
	if err != nil {
		return CampaignSupplyChainDTO{}, err
	}
	return CampaignSupplyChainDTO{
		CampaignID: campaignID.String(),
		Nodes:      nodes,
	}, nil
}

func (s *Service) UpdateCampaignSupplyChain(ctx context.Context, campaignID uuid.UUID, nodes []SupplyChainNode) (CampaignSupplyChainDTO, error) {
	if err := validateSupplyChainNodes(nodes); err != nil {
		return CampaignSupplyChainDTO{}, err
	}

	nodesJSON, err := coldpath.MarshalJSON(nodes)
	if err != nil {
		return CampaignSupplyChainDTO{}, err
	}

	var out CampaignSupplyChainDTO
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return err
		}

		oldNodes, _ := parseSupplyChainNodes(locked.SupplyChainNodes)
		oldNodesJSON, err := coldpath.MarshalJSON(oldNodes)
		if err != nil {
			return err
		}

		updated, err := q.UpdateCampaignSupplyChain(ctx, db.UpdateCampaignSupplyChainParams{
			ID:               domain.ToUUID(campaignID),
			SupplyChainNodes: nodesJSON,
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_SUPPLY_CHAIN", "campaign", &campaignID, auditSupplyChainChange{
			OldNodes: oldNodesJSON,
			NewNodes: nodesJSON,
		}, nil)

		parsed, err := parseSupplyChainNodes(updated.SupplyChainNodes)
		if err != nil {
			return err
		}
		out = CampaignSupplyChainDTO{
			CampaignID: campaignID.String(),
			Nodes:      parsed,
		}
		return nil
	})
	return out, err
}

func parseSupplyChainNodes(raw []byte) ([]SupplyChainNode, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []SupplyChainNode{}, nil
	}
	var nodes []SupplyChainNode
	if err := coldpath.UnmarshalJSON(raw, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

type iabSellersJSON struct {
	ContactEmail string          `json:"contact_email,omitempty"`
	Version      string          `json:"version"`
	Sellers      []iabSellerJSON `json:"sellers"`
}

type iabSellerJSON struct {
	SellerID       string `json:"seller_id"`
	Name           string `json:"name,omitempty"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	IsConfidential int    `json:"is_confidential,omitempty"`
}

func validateSellersJSON(doc iabSellersJSON) error {
	if strings.TrimSpace(doc.Version) == "" {
		return fmt.Errorf("%w: version required", ErrSellersJSONInvalid)
	}
	if doc.Sellers == nil {
		return fmt.Errorf("%w: sellers array required", ErrSellersJSONInvalid)
	}
	for i, s := range doc.Sellers {
		if strings.TrimSpace(s.SellerID) == "" || strings.TrimSpace(s.Domain) == "" {
			return fmt.Errorf("%w: seller %d missing seller_id or domain", ErrSellersJSONInvalid, i)
		}
		if _, err := normalizeSellerType(s.SellerType); err != nil {
			return fmt.Errorf("%w: seller %d invalid seller_type", ErrSellersJSONInvalid, i)
		}
	}
	return nil
}

func (s *Service) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	q := db.New(s.GetPool())
	rows, err := q.ListSellers(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	settingsMap := make(map[string]string, len(settings))
	for _, r := range settings {
		settingsMap[r.Key] = r.Value
	}

	doc := iabSellersJSON{
		Version: sellersJSONVersion,
		Sellers: make([]iabSellerJSON, 0, len(rows)),
	}
	if email := strings.TrimSpace(settingsMap[supplySettingContact]); email != "" {
		doc.ContactEmail = email
	}

	for _, row := range rows {
		entry := iabSellerJSON{
			SellerID:   row.SellerID,
			Domain:     row.Domain,
			SellerType: row.SellerType,
			Name:       row.Name,
		}
		if row.IsConfidential {
			entry.IsConfidential = 1
		}
		doc.Sellers = append(doc.Sellers, entry)
	}

	if err := validateSellersJSON(doc); err != nil {
		return nil, err
	}
	return coldpath.MarshalJSON(doc)
}

func (s *Service) GetSellersJSON(ctx context.Context) ([]byte, error) {
	now := time.Now()
	sellersCache.mu.RLock()
	if len(sellersCache.v.body) > 0 && now.Before(sellersCache.v.expires) {
		body := sellersCache.v.body
		sellersCache.mu.RUnlock()
		return body, nil
	}
	sellersCache.mu.RUnlock()

	body, err := s.BuildSellersJSON(ctx)
	if err != nil {
		return nil, err
	}

	sellersCache.mu.Lock()
	sellersCache.v = sellersJSONCacheEntry{body: body, expires: now.Add(sellersJSONCacheTTL)}
	sellersCache.mu.Unlock()
	return body, nil
}

func (s *Service) BuildAdsTxt(ctx context.Context) (string, error) {
	q := db.New(s.GetPool())
	rows, err := q.ListAdsTxtEntries(ctx)
	if err != nil {
		return "", err
	}
	settings, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return "", err
	}
	settingsMap := make(map[string]string, len(settings))
	for _, r := range settings {
		settingsMap[r.Key] = r.Value
	}

	var b strings.Builder
	if owner := strings.TrimSpace(settingsMap[supplySettingOwner]); owner != "" {
		b.WriteString("OWNERDOMAIN=")
		b.WriteString(owner)
		b.WriteByte('\n')
	}
	if manager := strings.TrimSpace(settingsMap[supplySettingManager]); manager != "" {
		b.WriteString("MANAGERDOMAIN=")
		b.WriteString(manager)
		b.WriteByte('\n')
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}

	for _, row := range rows {
		b.WriteString(row.Domain)
		b.WriteString(", ")
		b.WriteString(row.PublisherAccountID)
		b.WriteString(", ")
		b.WriteString(row.Relationship)
		if cert := strings.TrimSpace(row.CertAuthorityID); cert != "" {
			b.WriteString(", ")
			b.WriteString(cert)
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func (s *Service) SupplyExportPath() string {
	if s.cfg != nil && s.cfg.Management.SupplyExportPath != "" {
		return s.cfg.Management.SupplyExportPath
	}
	return "./data/supply-export"
}

func (s *Service) CreateCampaign(ctx context.Context, spec CampaignCreateSpec) (uuid.UUID, error) {
	if err := validateDaypartHours(spec.DaypartHours); err != nil {
		return uuid.Nil, err
	}
	if err := validateSchedule(spec.StartAt, spec.EndAt); err != nil {
		return uuid.Nil, err
	}
	if err := s.enforceDeploymentLicenseCampaignCap(ctx); err != nil {
		return uuid.Nil, err
	}
	pacing := db.PacingModeTypeASAP
	if spec.PacingMode != "" {
		pacing = db.PacingModeType(spec.PacingMode)
	}

	campaignID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate campaign id: %w", err)
	}
	now := time.Now()
	initialStatus := resolveScheduleStatus(now, spec.StartAt, spec.EndAt)

	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: spec.IdempotencyKey, Valid: true})
		if err == nil {
			if existing.CampaignID.Valid {
				campaignID = uuid.UUID(existing.CampaignID.Bytes)
				return nil
			}
			return fmt.Errorf("%w ledger row for key %q", ErrIncompleteIdempotency, spec.IdempotencyKey)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("idempotency lookup failed: %w", err)
		}
		cust, err := q.GetCustomerForUpdate(ctx, domain.ToUUID(spec.CustomerID))
		if err != nil {
			return mapNotFound(err, ErrCustomerNotFound)
		}
		if cust.Balance+cust.AllowedOverdraft < spec.BudgetLimitMicro {
			return ErrInsufficientBalance
		}

		var brandIDParam pgtype.UUID
		brandFcapKey := "fcap:c:" + campaignID.String()
		if spec.BrandID != nil {
			brand, err := q.GetBrand(ctx, domain.ToUUID(*spec.BrandID))
			if err != nil {
				return mapNotFound(err, ErrBrandNotFound)
			}
			if uuid.UUID(brand.CustomerID.Bytes) != spec.CustomerID {
				return ErrBrandBelongsToAnotherCustomer
			}
			brandIDParam = domain.ToUUID(*spec.BrandID)
			brandFcapKey = "fcap:b:" + spec.BrandID.String()
		}

		var templateIDParam pgtype.UUID
		if spec.TemplateID != nil {
			templateIDParam = domain.ToUUID(*spec.TemplateID)
		}

		if _, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(spec.CustomerID),
			Balance: -spec.BudgetLimitMicro,
		}); err != nil {
			return err
		}

		_, err = q.CreateCampaign(ctx, db.CreateCampaignParams{
			ID:              domain.ToUUID(campaignID),
			Name:            spec.Name,
			BudgetLimit:     spec.BudgetLimitMicro,
			Status:          initialStatus,
			CustomerID:      domain.ToUUID(spec.CustomerID),
			PacingMode:      pacing,
			DailyBudget:     spec.DailyBudgetMicro,
			Timezone:        spec.Timezone,
			FreqLimit:       pgtype.Int4{Int32: spec.FreqLimit, Valid: true},
			FreqWindow:      pgtype.Int4{Int32: spec.FreqWindow, Valid: true},
			TargetCountries: countriesOrEmpty(spec.TargetCountries),
			BrandID:         brandIDParam,
			BrandFcapKey:    brandFcapKey,
			StartAt:         toTimestamptz(spec.StartAt),
			EndAt:           toTimestamptz(spec.EndAt),
			DaypartHours:    daypartOrEmpty(spec.DaypartHours),
			TemplateID:      templateIDParam,
		})
		if err != nil {
			return err
		}

		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(spec.CustomerID),
			CampaignID:      domain.ToUUID(campaignID),
			Amount:          spec.BudgetLimitMicro,
			Type:            db.LedgerTypeFREEZE,
			IdempotencyHash: pgtype.Text{String: spec.IdempotencyKey, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return err
		}

		err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			NewStatus:  initialStatus,
			Reason:     pgtype.Text{String: "Campaign creation", Valid: true},
		})
		if err != nil {
			return err
		}

		s.AuditLog(ctx, q, uuid.Nil, "CREATE_CAMPAIGN", "campaign", &campaignID, auditCreateCampaignChange{
			Name:         spec.Name,
			BudgetLimit:  spec.BudgetLimitMicro,
			Status:       initialStatus,
			StartAt:      spec.StartAt,
			EndAt:        spec.EndAt,
			DaypartHours: spec.DaypartHours,
		}, auditIdempotencyMeta{IdempotencyKey: spec.IdempotencyKey})

		return s.emitCampaignLifecycleOutbox(ctx, q, campaignID, initialStatus, spec.BudgetLimitMicro)
	})
	return campaignID, err
}

func (s *Service) emitCampaignLifecycleOutbox(ctx context.Context, q db.Querier, campaignID uuid.UUID, status db.CampaignStatusType, budgetLimit int64) error {
	switch status {
	case db.CampaignStatusTypeACTIVE:
		payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String(), BudgetLimit: budgetLimit})
		if err != nil {
			return fmt.Errorf("marshal create campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "CREATE_CAMPAIGN", Payload: payload})
		return err
	case db.CampaignStatusTypePAUSED:
		payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String()})
		if err != nil {
			return fmt.Errorf("marshal pause campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "PAUSE_CAMPAIGN", Payload: payload})
		return err
	default:
		return nil
	}
}

func (s *Service) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	_, err := s.pauseCampaign(ctx, campaignID, reason, false)
	return err
}

func (s *Service) PreviewPauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (MutationPreview, error) {
	return s.pauseCampaign(ctx, campaignID, reason, true)
}

func (s *Service) pauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string, dryRun bool) (MutationPreview, error) {
	if dryRun {
		return s.previewPauseCampaign(ctx, campaignID, reason)
	}
	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapNotFound(err, ErrCampaignNotFound)
		}
		if camp.Status == db.CampaignStatusTypePAUSED {
			return nil
		}
		if camp.Status != db.CampaignStatusTypeACTIVE {
			return fmt.Errorf("%w in status %s", ErrCampaignCannotBePaused, camp.Status)
		}

		_, err = q.PauseCampaign(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return err
		}
		err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: camp.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypePAUSED,
			Reason:     pgtype.Text{String: reason, Valid: reason != ""},
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "PAUSE_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)

		payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String()})
		if err != nil {
			return fmt.Errorf("marshal pause campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "PAUSE_CAMPAIGN", Payload: payload})
		return err
	})
	return MutationPreview{}, err
}

func (s *Service) previewPauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (MutationPreview, error) {
	camp, err := db.New(s.GetPool()).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return MutationPreview{}, mapNotFound(err, ErrCampaignNotFound)
	}
	if camp.Status == db.CampaignStatusTypePAUSED {
		return newMutationPreview("PAUSE_CAMPAIGN", PauseCampaignWouldChange{
			CampaignID: campaignID.String(),
			Status:     string(camp.Status),
			Noop:       true,
		})
	}
	if camp.Status != db.CampaignStatusTypeACTIVE {
		return MutationPreview{}, fmt.Errorf("%w in status %s", ErrCampaignCannotBePaused, camp.Status)
	}
	return newMutationPreview("PAUSE_CAMPAIGN", PauseCampaignWouldChange{
		CampaignID:  campaignID.String(),
		StatusFrom:  string(camp.Status),
		StatusTo:    string(db.CampaignStatusTypePAUSED),
		OutboxEvent: "PAUSE_CAMPAIGN",
		Reason:      reason,
	})
}

func (s *Service) ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	_, err := s.resumeCampaign(ctx, campaignID, reason, false)
	return err
}

func (s *Service) PreviewResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (MutationPreview, error) {
	return s.resumeCampaign(ctx, campaignID, reason, true)
}

func (s *Service) resumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string, dryRun bool) (MutationPreview, error) {
	if dryRun {
		return s.previewResumeCampaign(ctx, campaignID, reason)
	}
	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapNotFound(err, ErrCampaignNotFound)
		}
		if camp.Status != db.CampaignStatusTypePAUSED {
			return ErrCampaignNotPaused
		}

		now := time.Now()
		var startAt, endAt *time.Time
		if camp.StartAt.Valid {
			startAt = &camp.StartAt.Time
		}
		if camp.EndAt.Valid {
			endAt = &camp.EndAt.Time
		}
		if resolveScheduleStatus(now, startAt, endAt) != db.CampaignStatusTypeACTIVE {
			return ErrCampaignOutsideSchedule
		}

		_, err = q.ResumeCampaign(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return err
		}
		err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: camp.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypeACTIVE,
			Reason:     pgtype.Text{String: reason, Valid: reason != ""},
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "RESUME_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)

		payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String(), BudgetLimit: camp.BudgetLimit})
		if err != nil {
			return fmt.Errorf("marshal resume campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "RESUME_CAMPAIGN", Payload: payload})
		return err
	})
	return MutationPreview{}, err
}

func (s *Service) previewResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (MutationPreview, error) {
	camp, err := db.New(s.GetPool()).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return MutationPreview{}, mapNotFound(err, ErrCampaignNotFound)
	}
	if camp.Status != db.CampaignStatusTypePAUSED {
		return MutationPreview{}, ErrCampaignNotPaused
	}
	now := time.Now()
	var startAt, endAt *time.Time
	if camp.StartAt.Valid {
		startAt = &camp.StartAt.Time
	}
	if camp.EndAt.Valid {
		endAt = &camp.EndAt.Time
	}
	if resolveScheduleStatus(now, startAt, endAt) != db.CampaignStatusTypeACTIVE {
		return MutationPreview{}, ErrCampaignOutsideSchedule
	}
	return newMutationPreview("RESUME_CAMPAIGN", ResumeCampaignWouldChange{
		CampaignID:  campaignID.String(),
		StatusFrom:  string(camp.Status),
		StatusTo:    string(db.CampaignStatusTypeACTIVE),
		OutboxEvent: "RESUME_CAMPAIGN",
		Reason:      reason,
	})
}

func (s *Service) UpdateCampaignSchedule(ctx context.Context, campaignID uuid.UUID, startAt, endAt *time.Time, daypartHours []int16) error {
	if err := validateDaypartHours(daypartHours); err != nil {
		return err
	}
	if err := validateSchedule(startAt, endAt); err != nil {
		return err
	}

	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return err
		}
		_, err = q.UpdateCampaignSchedule(ctx, db.UpdateCampaignScheduleParams{
			ID:           domain.ToUUID(campaignID),
			StartAt:      toTimestamptz(startAt),
			EndAt:        toTimestamptz(endAt),
			DaypartHours: daypartOrEmpty(daypartHours),
		})
		if err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_SCHEDULE", "campaign", &campaignID, auditCampaignScheduleChange{
			StartAt:      startAt,
			EndAt:        endAt,
			DaypartHours: daypartHours,
		}, nil)

		payload, err := coldpath.MarshalOutbox(campaignScheduleOutboxPayload{
			CampaignID:   campaignID.String(),
			StartAt:      startAt,
			EndAt:        endAt,
			DaypartHours: daypartHours,
		})
		if err != nil {
			return fmt.Errorf("marshal update campaign schedule outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "UPDATE_CAMPAIGN_SCHEDULE", Payload: payload})
		if err != nil {
			return err
		}

		desired := resolveScheduleStatus(time.Now(), startAt, endAt)
		if desired == db.CampaignStatusTypePAUSED && locked.Status == db.CampaignStatusTypeACTIVE {
			return s.transitionCampaignStatus(ctx, q, campaignID, locked.Status, db.CampaignStatusTypePAUSED, "schedule_window", locked.BudgetLimit)
		}
		if desired == db.CampaignStatusTypeACTIVE && locked.Status == db.CampaignStatusTypePAUSED {
			return s.transitionCampaignStatus(ctx, q, campaignID, locked.Status, db.CampaignStatusTypeACTIVE, "schedule_window", locked.BudgetLimit)
		}
		return nil
	})
}

func (s *Service) transitionCampaignStatus(ctx context.Context, q db.Querier, campaignID uuid.UUID, old, new db.CampaignStatusType, reason string, budget int64) error {
	_, err := q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		ID:     domain.ToUUID(campaignID),
		Status: new,
	})
	if err != nil {
		return err
	}
	err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
		CampaignID: domain.ToUUID(campaignID),
		OldStatus:  db.NullCampaignStatusType{CampaignStatusType: old, Valid: true},
		NewStatus:  new,
		Reason:     pgtype.Text{String: reason, Valid: true},
	})
	if err != nil {
		return err
	}
	return s.emitCampaignLifecycleOutbox(ctx, q, campaignID, new, budget)
}

func (s *Service) CreateCampaignTemplate(ctx context.Context, customerID uuid.UUID, name string, budgetLimit int64, pacing db.PacingModeType, dailyBudget int64, timezone string, freqLimit, freqWindow int32, targetCountries []string, brandID *uuid.UUID, daypartHours []int16) (uuid.UUID, error) {
	if err := validateDaypartHours(daypartHours); err != nil {
		return uuid.Nil, err
	}
	templateID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	var brandParam pgtype.UUID
	if brandID != nil {
		brandParam = domain.ToUUID(*brandID)
	}

	_, err = db.New(s.GetPool()).CreateCampaignTemplate(ctx, db.CreateCampaignTemplateParams{
		ID:              domain.ToUUID(templateID),
		CustomerID:      domain.ToUUID(customerID),
		Name:            name,
		BudgetLimit:     budgetLimit,
		PacingMode:      pacing,
		DailyBudget:     dailyBudget,
		Timezone:        timezone,
		FreqLimit:       freqLimit,
		FreqWindow:      freqWindow,
		TargetCountries: countriesOrEmpty(targetCountries),
		BrandID:         brandParam,
		DaypartHours:    daypartOrEmpty(daypartHours),
	})
	return templateID, err
}

func (s *Service) ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]CampaignTemplateDTO, int64, error) {
	q := db.New(s.GetPool())
	cid := domain.ToUUID(customerID)
	listParams := db.ListCampaignTemplatesParams{
		CustomerID: cid,
		Limit:      limit,
		Offset:     offset,
	}
	total, err := q.CountCampaignTemplates(ctx, cid)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []CampaignTemplateDTO{}, 0, nil
	}
	templateRows, err := q.ListCampaignTemplates(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CampaignTemplateDTO, 0, len(templateRows))
	for _, t := range templateRows {
		countries := t.TargetCountries
		if countries == nil {
			countries = []string{}
		}
		hours := t.DaypartHours
		if hours == nil {
			hours = []int16{}
		}
		var brandID string
		if t.BrandID.Valid {
			brandID = uuid.UUID(t.BrandID.Bytes).String()
		}
		out = append(out, CampaignTemplateDTO{
			ID:              uuid.UUID(t.ID.Bytes).String(),
			CustomerID:      uuid.UUID(t.CustomerID.Bytes).String(),
			Name:            t.Name,
			BudgetLimit:     formatMicro(t.BudgetLimit),
			PacingMode:      string(t.PacingMode),
			DailyBudget:     formatMicro(t.DailyBudget),
			Timezone:        t.Timezone,
			FreqLimit:       t.FreqLimit,
			FreqWindow:      t.FreqWindow,
			TargetCountries: countries,
			BrandID:         brandID,
			DaypartHours:    hours,
			CreatedAt:       t.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:       t.UpdatedAt.Time.Format(time.RFC3339),
		})
	}
	return out, total, nil
}

func (s *Service) CreateCampaignFromTemplate(ctx context.Context, templateID uuid.UUID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error) {
	tmpl, err := db.New(s.GetPool()).GetCampaignTemplate(ctx, domain.ToUUID(templateID))
	if err != nil {
		return uuid.Nil, mapNotFound(err, ErrTemplateNotFound)
	}
	if uuid.UUID(tmpl.CustomerID.Bytes) != customerID {
		return uuid.Nil, ErrTemplateBelongsToAnotherCustomer
	}

	limit := tmpl.BudgetLimit
	if budgetLimit != nil {
		limit = *budgetLimit
	}
	if name == "" {
		name = tmpl.Name
	}

	var brandID *uuid.UUID
	if tmpl.BrandID.Valid {
		id := uuid.UUID(tmpl.BrandID.Bytes)
		brandID = &id
	}

	return s.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       customerID,
		BrandID:          brandID,
		Name:             name,
		BudgetLimitMicro: limit,
		PacingMode:       string(tmpl.PacingMode),
		DailyBudgetMicro: tmpl.DailyBudget,
		Timezone:         tmpl.Timezone,
		FreqLimit:        tmpl.FreqLimit,
		FreqWindow:       tmpl.FreqWindow,
		TargetCountries:  tmpl.TargetCountries,
		DaypartHours:     tmpl.DaypartHours,
		TemplateID:       &templateID,
		IdempotencyKey:   idempotencyKey,
	})
}

func (s *Service) SaveCampaignAsTemplate(ctx context.Context, campaignID uuid.UUID, templateName string) (uuid.UUID, error) {
	camp, err := s.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return uuid.Nil, err
	}
	if templateName == "" {
		templateName = camp.Name + " template"
	}
	var brandID *uuid.UUID
	if camp.BrandID.Valid {
		id := uuid.UUID(camp.BrandID.Bytes)
		brandID = &id
	}
	hours := camp.DaypartHours
	if hours == nil {
		hours = []int16{}
	}
	return s.CreateCampaignTemplate(ctx,
		uuid.UUID(camp.CustomerID.Bytes),
		templateName,
		camp.BudgetLimit,
		camp.PacingMode,
		camp.DailyBudget,
		camp.Timezone,
		camp.FreqLimit.Int32,
		camp.FreqWindow.Int32,
		camp.TargetCountries,
		brandID,
		hours,
	)
}

func (s *Service) UpsertBrandCreative(ctx context.Context, brandID uuid.UUID, name, landingURL string, weight int32, status string) (uuid.UUID, error) {
	if weight <= 0 {
		return uuid.Nil, ErrWeightMustBePositive
	}
	if status == "" {
		status = "ACTIVE"
	}
	if status != "ACTIVE" && status != "PAUSED" {
		return uuid.Nil, ErrCreativeStatusInvalid
	}

	creativeID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetBrand(ctx, domain.ToUUID(brandID)); err != nil {
			return mapNotFound(err, ErrBrandNotFound)
		}
		_, err := q.CreateBrandCreative(ctx, db.CreateBrandCreativeParams{
			ID:         domain.ToUUID(creativeID),
			BrandID:    domain.ToUUID(brandID),
			Name:       name,
			LandingUrl: landingURL,
			Weight:     weight,
			Status:     status,
		})
		if err != nil {
			return err
		}
		return s.emitBrandCreativesOutbox(ctx, q, brandID)
	})
	return creativeID, err
}

func (s *Service) ListBrandCreatives(ctx context.Context, brandID uuid.UUID) ([]BrandCreativeDTO, error) {
	rows, err := db.New(s.GetPool()).ListBrandCreatives(ctx, domain.ToUUID(brandID))
	if err != nil {
		return nil, err
	}
	out := make([]BrandCreativeDTO, len(rows))
	for i, c := range rows {
		out[i] = BrandCreativeDTO{
			ID:         uuid.UUID(c.ID.Bytes).String(),
			BrandID:    uuid.UUID(c.BrandID.Bytes).String(),
			Name:       c.Name,
			LandingURL: c.LandingUrl,
			Weight:     c.Weight,
			Status:     c.Status,
			CreatedAt:  c.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:  c.UpdatedAt.Time.Format(time.RFC3339),
		}
	}
	return out, nil
}

func (s *Service) UpdateBrandCreative(ctx context.Context, creativeID uuid.UUID, name, landingURL string, weight int32, status string) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		existing, err := q.GetBrandCreative(ctx, domain.ToUUID(creativeID))
		if err != nil {
			return mapNotFound(err, ErrCreativeNotFound)
		}
		_, err = q.UpdateBrandCreative(ctx, db.UpdateBrandCreativeParams{
			ID:         domain.ToUUID(creativeID),
			Name:       name,
			LandingUrl: landingURL,
			Weight:     weight,
			Status:     status,
		})
		if err != nil {
			return err
		}
		return s.emitBrandCreativesOutbox(ctx, q, uuid.UUID(existing.BrandID.Bytes))
	})
}

func (s *Service) DeleteBrandCreative(ctx context.Context, creativeID uuid.UUID) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		existing, err := q.GetBrandCreative(ctx, domain.ToUUID(creativeID))
		if err != nil {
			return mapNotFound(err, ErrCreativeNotFound)
		}
		if err := q.DeleteBrandCreative(ctx, domain.ToUUID(creativeID)); err != nil {
			return err
		}
		return s.emitBrandCreativesOutbox(ctx, q, uuid.UUID(existing.BrandID.Bytes))
	})
}

func (s *Service) emitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error {
	payload, err := coldpath.MarshalOutbox(brandIDPayload{BrandID: brandID.String()})
	if err != nil {
		return fmt.Errorf("marshal sync brand creatives outbox payload: %w", err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "SYNC_BRAND_CREATIVES", Payload: payload})
	return err
}

func (s *Service) ProcessScheduleTick(ctx context.Context) error {
	opCtx, cancel := workerContext(ctx, workerBatchTimeout)
	defer cancel()

	for i := int32(0); i < 200; i++ {
		done, err := s.processNextScheduledCampaign(opCtx)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return nil
}

func (s *Service) processNextScheduledCampaign(ctx context.Context) (done bool, err error) {
	var campID uuid.UUID
	var desired db.CampaignStatusType

	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.ClaimScheduledCampaignForUpdate(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				done = true
				return nil
			}
			return err
		}

		var startAt, endAt *time.Time
		if camp.StartAt.Valid {
			startAt = &camp.StartAt.Time
		}
		if camp.EndAt.Valid {
			endAt = &camp.EndAt.Time
		}
		desired = resolveScheduleStatus(time.Now(), startAt, endAt)
		if desired == camp.Status {
			return nil
		}
		campID = uuid.UUID(camp.ID.Bytes)
		return nil
	})
	if err != nil || done || campID == uuid.Nil {
		return done, err
	}

	var opErr error
	if desired == db.CampaignStatusTypeACTIVE {
		opErr = s.ResumeCampaign(ctx, campID, "schedule_auto_resume")
	} else {
		opErr = s.PauseCampaign(ctx, campID, "schedule_auto_pause")
	}
	if opErr != nil {
		return false, nil
	}
	return false, nil
}

const (
	outboxPriSyncBrandCreatives = 1
	outboxPriCreateCampaign     = 2
	outboxPriPacing             = 3
	outboxPriPause              = 4
)

type deliveryOutboxEntry struct {
	priority  int
	eventType string
	payload   []byte
}

type deliveryOutboxMerge map[uuid.UUID]deliveryOutboxEntry

func (m deliveryOutboxMerge) upsert(campaignID uuid.UUID, priority int, eventType string, payload []byte) {
	if m == nil {
		return
	}
	if existing, ok := m[campaignID]; ok && existing.priority >= priority {
		return
	}
	m[campaignID] = deliveryOutboxEntry{
		priority:  priority,
		eventType: eventType,
		payload:   payload,
	}
}

func (m deliveryOutboxMerge) flush(ctx context.Context, pool pgx.Tx) error {
	if len(m) == 0 {
		return nil
	}
	q := db.New(pool)
	for _, entry := range m {
		if _, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: entry.eventType,
			Payload:   entry.payload,
		}); err != nil {
			return fmt.Errorf("flush delivery optimizer outbox %s: %w", entry.eventType, err)
		}
	}
	return nil
}

func (s *Service) RunDeliveryOptimizerTick(ctx context.Context, syncWorkers []*domain.SyncWorker, runMAB bool) error {
	return s.withPgLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := workerContext(runCtx, workerBatchTimeout)
		defer cancel()

		for _, sw := range syncWorkers {
			if sw != nil {
				sw.SyncAll(opCtx)
			}
		}

		merge := make(deliveryOutboxMerge)
		var mabBrands []uuid.UUID

		err := pgx.BeginFunc(opCtx, s.GetPool(), func(tx pgx.Tx) error {
			if err := s.closedLoopPacingControllerTx(opCtx, tx, merge); err != nil {
				return err
			}
			if err := s.autoscaleBudgetsTx(opCtx, tx, merge); err != nil {
				return err
			}
			if runMAB {
				brands, err := s.optimizeBrandCreativeMABTx(opCtx, tx)
				if err != nil {
					return err
				}
				mabBrands = brands
			}
			if err := merge.flush(opCtx, tx); err != nil {
				return err
			}
			for _, brandID := range mabBrands {
				if err := s.emitBrandCreativesOutbox(opCtx, db.New(tx), brandID); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
}

const pacingLookbackDays = 90

func (s *Service) ClosedLoopPacingController(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	return s.withPgLow(ctx, func(runCtx context.Context) error {
		opCtx, cancel := workerContext(runCtx, workerBatchTimeout)
		defer cancel()

		for _, sw := range syncWorkers {
			if sw != nil {
				sw.SyncAll(opCtx)
			}
		}

		return pgx.BeginFunc(opCtx, s.GetPool(), func(tx pgx.Tx) error {
			return s.closedLoopPacingControllerTx(opCtx, tx, nil)
		})
	})
}

func (s *Service) closedLoopPacingControllerTx(ctx context.Context, tx pgx.Tx, merge deliveryOutboxMerge) error {
	q := db.New(tx)
	rows, err := q.GetAllActiveCampaignsWithStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch active campaigns for pacing: %w", err)
	}

	hourWeights := s.fetchPacingHourWeights(ctx)
	now := time.Now()

	for _, row := range rows {
		camp, err := q.GetCampaignForUpdate(ctx, row.ID)
		if err != nil {
			return fmt.Errorf("failed to lock campaign for pacing: %w", err)
		}
		if camp.Status != db.CampaignStatusTypeACTIVE {
			continue
		}

		loc := s.campaignLocation(camp.Timezone)
		localNow := now.In(loc)

		daypart := camp.DaypartHours
		if daypart == nil {
			daypart = []int16{}
		}
		timeRatio := smartPacingExpectedRatio(hourWeights, daypart, localNow)

		budgetMicro := camp.DailyBudget
		if budgetMicro == 0 {
			budgetMicro = camp.BudgetLimit
		}
		if budgetMicro == 0 {
			continue
		}

		actualSpendMicro := camp.CurrentSpend
		ratioPPM := int64(timeRatio * 1_000_000)
		expectedSpendMicro := money.ScalePPM(budgetMicro, ratioPPM)

		var targetPacing db.PacingModeType
		var shouldUpdate bool

		tolerancePPM := int64(s.cfg.PacingToleranceMargin * 1_000_000)
		overThresholdMicro := money.ScalePPM(expectedSpendMicro, 1_000_000+tolerancePPM)
		underThresholdMicro := money.ScalePPM(expectedSpendMicro, 1_000_000-tolerancePPM)

		if camp.PacingMode == db.PacingModeTypeASAP && actualSpendMicro > overThresholdMicro {
			targetPacing = db.PacingModeTypeEVEN
			shouldUpdate = true
		} else if camp.PacingMode == db.PacingModeTypeEVEN && actualSpendMicro < underThresholdMicro {
			targetPacing = db.PacingModeTypeASAP
			shouldUpdate = true
		}

		if !shouldUpdate {
			continue
		}

		campID := uuid.UUID(camp.ID.Bytes)
		_, err = q.UpdateCampaignPacing(ctx, db.UpdateCampaignPacingParams{
			ID:         camp.ID,
			PacingMode: targetPacing,
		})
		if err != nil {
			return fmt.Errorf("failed to update pacing mode: %w", err)
		}

		actualSpendStr := money.FormatDecimal(actualSpendMicro)
		expectedSpendStr := money.FormatDecimal(expectedSpendMicro)

		s.AuditLog(ctx, q, uuid.Nil, "PACING_LOOP_ADJUSTMENT", "campaign", &campID, auditPacingLoopAdjustment{
			OldPacing: string(camp.PacingMode),
			NewPacing: string(targetPacing),
			Spend:     actualSpendStr,
			Expected:  expectedSpendStr,
			Curve:     "daypart_weighted",
		}, nil)

		payloadBytes, err := coldpath.MarshalOutbox(campaignPacingPayload{
			CampaignID: campID.String(),
			PacingMode: string(targetPacing),
		})
		if err != nil {
			return fmt.Errorf("failed to marshal pacing outbox payload: %w", err)
		}

		if merge != nil {
			merge.upsert(campID, outboxPriPacing, "UPDATE_CAMPAIGN_PACING", payloadBytes)
		} else {
			_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "UPDATE_CAMPAIGN_PACING",
				Payload:   payloadBytes,
			})
			if err != nil {
				return fmt.Errorf("failed to create outbox event for pacing: %w", err)
			}
		}
	}

	return nil
}

func (s *Service) campaignLocation(timezone string) *time.Location {
	if cached, found := s.locCache.Load(timezone); found {
		return cached.(*time.Location)
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	s.locCache.Store(timezone, loc)
	return loc
}

func (s *Service) fetchPacingHourWeights(ctx context.Context) [24]float64 {
	if s.chQuery == nil {
		return uniformHourWeights()
	}
	lookbackEnd := time.Now().UTC().Truncate(time.Hour)
	lookbackStart := lookbackEnd.Add(-pacingLookbackDays * 24 * time.Hour)
	_, samples, err := s.queryForecastHourlySamples(ctx, lookbackStart, lookbackEnd, nil)
	if err != nil {
		return uniformHourWeights()
	}
	return buildHourWeights(samples)
}

func uniformHourWeights() [24]float64 {
	var weights [24]float64
	for i := range weights {
		weights[i] = 1.0 / 24.0
	}
	return weights
}

func smartPacingExpectedRatio(weights [24]float64, daypart []int16, localNow time.Time) float64 {
	daypartSet := make(map[int16]struct{}, len(daypart))
	for _, h := range daypart {
		daypartSet[h] = struct{}{}
	}
	useDaypart := len(daypartSet) > 0

	currentHour := localNow.Hour()
	minuteFrac := (float64(localNow.Minute()) + float64(localNow.Second())/60.0) / 60.0

	var totalWeight, elapsedWeight float64
	for h := 0; h < 24; h++ {
		if useDaypart {
			if _, ok := daypartSet[int16(h)]; !ok {
				continue
			}
		}
		w := weights[h]
		if w <= 0 {
			w = 1.0 / 24.0
		}
		totalWeight += w
		switch {
		case h < currentHour:
			elapsedWeight += w
		case h == currentHour:
			elapsedWeight += w * minuteFrac
		}
	}
	if totalWeight <= 0 {
		startOfDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, localNow.Location())
		elapsed := localNow.Sub(startOfDay).Seconds()
		if elapsed < 0 {
			elapsed = 0
		}
		ratio := elapsed / 86400.0
		if ratio > 1.0 {
			ratio = 1.0
		}
		return ratio
	}
	ratio := elapsedWeight / totalWeight
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	return ratio
}

type CohortVariantSpec struct {
	ID     string            `json:"id"`
	Weight uint32            `json:"weight"`
	Flags  map[string]string `json:"flags,omitempty"`
}

type ExperimentCohortSpec struct {
	ID       uuid.UUID           `json:"id"`
	Name     string              `json:"name"`
	Active   bool                `json:"active"`
	Salt     string              `json:"salt"`
	Variants []CohortVariantSpec `json:"variants"`
}

type cohortSnapshotPayload struct {
	Version int64 `json:"version"`
}

func (s *Service) UpsertExperimentCohort(ctx context.Context, spec ExperimentCohortSpec) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	if spec.ID == uuid.Nil || spec.Name == "" || spec.Salt == "" || len(spec.Variants) == 0 {
		return fmt.Errorf("invalid cohort spec")
	}
	variantsJSON, err := json.Marshal(spec.Variants)
	if err != nil {
		return fmt.Errorf("marshal cohort variants: %w", err)
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		_, err := q.UpsertExperimentCohort(ctx, db.UpsertExperimentCohortParams{
			ID:       domain.ToUUID(spec.ID),
			Name:     spec.Name,
			Active:   spec.Active,
			Salt:     spec.Salt,
			Variants: variantsJSON,
		})
		if err != nil {
			return err
		}

		payloadBytes, err := coldpath.MarshalOutbox(cohortSnapshotPayload{Version: 1})
		if err != nil {
			return err
		}
		ev, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_COHORT_SNAPSHOT",
			Payload:   payloadBytes,
		})
		if err != nil {
			return err
		}
		s.AuditLog(ctx, q, uuid.Nil, "UPDATE_COHORT_SNAPSHOT", "experiment", &spec.ID, auditCohortSnapshotChange{
			Name:     spec.Name,
			Active:   spec.Active,
			Variants: len(spec.Variants),
		}, auditOutboxEventMeta{OutboxEventID: ev.ID})
		return nil
	})
}

func (s *Service) publishRegistryFullSync(ctx context.Context) error {
	return s.publishCampaignUpdate(ctx, domain.RegistryFullSyncPayload)
}
