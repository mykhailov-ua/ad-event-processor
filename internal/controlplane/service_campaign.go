package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

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

func formatOptionalText(t pgtype.Text) string {
	return campaign.FormatOptionalText(t)
}

func (s *Service) ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]CampaignDTO, int64, error) {
	return s.CampaignRuntime().ListCampaigns(ctx, customerID, status, limit, offset)
}

func (s *Service) GetCampaign(ctx context.Context, id uuid.UUID) (CampaignDTO, error) {
	return s.CampaignRuntime().GetCampaign(ctx, id)
}

func (s *Service) PatchCampaign(ctx context.Context, campaignID uuid.UUID, req PatchCampaignRequest) (CampaignDTO, error) {
	return s.CampaignRuntime().PatchCampaign(ctx, campaignID, req)
}

func (s *Service) AssignCampaignBrand(ctx context.Context, campaignID, brandID uuid.UUID) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	if campaignID == uuid.Nil {
		return fmt.Errorf("campaign id required")
	}
	camp, err := s.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return err
	}
	customerID := uuid.UUID(camp.CustomerID.Bytes)

	brandFcapKey := "fcap:c:" + campaignID.String()
	brandArg := brandIDOrNil(uuid.Nil)
	auditBrandID := ""
	if brandID != uuid.Nil {
		q := db.New(s.GetPool())
		brand, err := q.GetBrand(ctx, domain.ToUUID(brandID))
		if err != nil {
			return mapNotFound(err, ErrBrandNotFound)
		}
		if uuid.UUID(brand.CustomerID.Bytes) != customerID {
			return ErrBrandBelongsToAnotherCustomer
		}
		brandFcapKey = "fcap:b:" + brandID.String()
		brandArg = brandID
		auditBrandID = brandID.String()
	}

	tag, err := s.pool.Exec(ctx,
		`UPDATE campaigns SET brand_id = $2, brand_fcap_key = $3, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		campaignID, brandArg, brandFcapKey,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}

	var uid uuid.UUID
	if u, ok := GetUser(ctx); ok {
		uid = u.UserID
	}
	s.AuditLog(ctx, db.New(s.GetPool()), uid, "PATCH_CAMPAIGN", "campaign", &campaignID, auditCampaignBrandChange{
		BrandID: auditBrandID,
	}, nil)

	_ = s.publishCampaignUpdate(ctx, campaignID.String())
	return nil
}

func brandIDOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func (s *Service) ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]CampaignEventDTO, int64, error) {
	return s.CampaignRuntime().ListCampaignEvents(ctx, campaignID, limit, offset)
}

func (s *Service) ListStatusHistory(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]StatusHistoryDTO, int64, error) {
	return s.CampaignRuntime().ListStatusHistory(ctx, campaignID, limit, offset)
}

const clickHouseStaleThreshold = 5 * time.Minute

func scrubCampaignFields(c CampaignDTO, level authz.MaskLevel) CampaignDTO {
	if level == authz.MaskFull {
		return c
	}
	out := c
	var redacted []string
	if out.TargetURL != "" {
		redacted = append(redacted, "target_url")
		out.TargetURL = ""
	}
	if len(out.CreativePayload) > 0 {
		redacted = append(redacted, "creative_payload")
		out.CreativePayload = nil
	}
	if out.ReferrerFilter != "" {
		redacted = append(redacted, "referrer_filter")
		out.ReferrerFilter = ""
	}
	if out.BudgetLimit != "" {
		redacted = append(redacted, "budget_limit")
		out.BudgetLimit = ""
		out.BudgetLimitDisplay = redactedMoneyDisplay()
	}
	if out.DailyBudget != "" {
		redacted = append(redacted, "daily_budget")
		out.DailyBudget = ""
		out.DailyBudgetDisplay = redactedMoneyDisplay()
	}
	out.FieldsRedacted = redacted
	return out
}

func scrubCampaignDTO(ctx context.Context, c db.Campaign) CampaignDTO {
	countries := c.TargetCountries
	if countries == nil {
		countries = []string{}
	}
	dto := CampaignDTO{
		ID:                         uuid.UUID(c.ID.Bytes).String(),
		Name:                       c.Name,
		Status:                     string(c.Status),
		BudgetLimit:                formatMicro(c.BudgetLimit),
		CurrentSpend:               formatMicro(c.CurrentSpend),
		CustomerID:                 uuid.UUID(c.CustomerID.Bytes).String(),
		PacingMode:                 string(c.PacingMode),
		DailyBudget:                formatMicro(c.DailyBudget),
		Timezone:                   c.Timezone,
		FreqLimit:                  c.FreqLimit.Int32,
		FreqWindow:                 c.FreqWindow.Int32,
		TargetCountries:            countries,
		TargetURL:                  c.TargetUrl,
		SafePageURL:                c.SafePageUrl,
		SafePageEnabled:            c.SafePageEnabled,
		AttestationEnabled:         c.AttestationEnabled,
		AttestationMode:            c.AttestationMode,
		AttestationTTLSec:          c.AttestationTtlSec,
		DmrEnabled:                 c.DmrEnabled,
		CIDRBlockEnabled:           c.CidrBlockEnabled,
		ProxyVPNBlockEnabled:       c.ProxyVpnBlockEnabled,
		ModeratorIntelEnabled:      c.ModeratorIntelEnabled,
		ReviewTrafficAction:        string(domain.ParseReviewTrafficAction(c.ReviewTrafficAction)),
		TLSFingerprintBlockEnabled: c.TlsFingerprintBlockEnabled,
		ConnTypePolicy:             c.ConnTypePolicy,
		LinkSigningEnabled:         c.LinkSigningEnabled,
		LinkSigningTTLSec:          c.LinkSigningTtlSec,
		ClickDelivery:              c.ClickDelivery,
		ProxyUpstreamURL:           c.ProxyUpstreamUrl,
		ProxyRewriteAssets:         c.ProxyRewriteAssets,
		BrandID:                    formatOptionalUUID(c.BrandID),
		CreativePayload:            json.RawMessage(c.CreativePayload),
		ReferrerFilter:             c.ReferrerFilter,
		StartAt:                    formatOptionalTime(c.StartAt),
		EndAt:                      formatOptionalTime(c.EndAt),
		DaypartHours:               daypartOrEmpty(c.DaypartHours),
		OwnerUserID:                formatOptionalUUID(c.OwnerUserID),
		TrafficTemplateID:          formatOptionalText(c.TrafficTemplateID),
		ClickQueryParams:           clickQueryParamsFromRaw(c.ClickQueryParams),
		CreatedAt:                  c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:                  c.UpdatedAt.Time.Format(time.RFC3339),
		Revision:                   campaignRevision(c.UpdatedAt.Time.Format(time.RFC3339)),
	}
	if len(c.IngressCostConfig) > 0 {
		parsed := domain.ParseIngressCostConfigJSON(c.IngressCostConfig)
		if parsed.Enabled() {
			scale := "decimal"
			if parsed.ScaleMicro {
				scale = "micro"
			}
			policy := "ignore"
			if parsed.Policy == domain.IngressCostPolicyReject {
				policy = "reject"
			}
			param := ""
			switch parsed.Param {
			case domain.IngressCostParamCost:
				param = "cost"
			case domain.IngressCostParamCPC:
				param = "cpc"
			case domain.IngressCostParamBid:
				param = "bid"
			}
			dto.IngressCostConfig = &IngressCostConfigDTO{
				Param:    param,
				Scale:    scale,
				MaxMicro: parsed.MaxMicro,
				Policy:   policy,
			}
		}
	}
	if snap, ok := authz.SnapshotFromContext(ctx); ok {
		out := scrubCampaignFields(dto, snap.Mask)
		attachCampaignPresentation(ctx, &out)
		return out
	}
	attachCampaignPresentation(ctx, &dto)
	return dto
}

func (s *Service) SetClickHouse(conn driver.Conn, cfg database.ClickHouseQueryConfig) {
	if conn != nil {
		s.clickhouseQuery = database.NewClickHouseQuery(conn, cfg)
		if cr := s.campaignRuntime; cr != nil {
			cr.SetClickHouseQuery(s.clickhouseQuery)
		}
	}
}

func (s *Service) SetClickHouseWrite(conn driver.Conn) {
	s.clickhouseWriteConn = conn
}

func (s *Service) UpdateCampaignPacing(ctx context.Context, campaignID uuid.UUID, newMode string) (CampaignDTO, error) {
	return s.CampaignRuntime().UpdateCampaignPacing(ctx, campaignID, newMode)
}

func (s *Service) GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (CampaignStatsDTO, error) {
	return s.CampaignRuntime().GetCampaignStats(ctx, campaignID, from, to, granularity)
}

type clickhouseLagCache struct {
	mu      sync.Mutex
	lag     time.Duration
	updated time.Time
}

const clickhouseLagCacheTTL = 30 * time.Second

var globalClickHouseLagCache clickhouseLagCache

func (s *Service) clickHouseIngestionLag(ctx context.Context) (time.Duration, error) {
	if s.clickhouseQuery == nil {
		return 0, nil
	}

	globalClickHouseLagCache.mu.Lock()
	if time.Since(globalClickHouseLagCache.updated) < clickhouseLagCacheTTL {
		lag := globalClickHouseLagCache.lag
		globalClickHouseLagCache.mu.Unlock()
		return lag, nil
	}
	globalClickHouseLagCache.mu.Unlock()

	var latest time.Time
	err := s.clickhouseQuery.QueryRow(ctx, `
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

	globalClickHouseLagCache.mu.Lock()
	globalClickHouseLagCache.lag = lag
	globalClickHouseLagCache.updated = time.Now()
	globalClickHouseLagCache.mu.Unlock()

	return lag, nil
}

func (s *Service) CreateCampaign(ctx context.Context, spec CampaignCreateSpec) (uuid.UUID, error) {
	return s.CampaignRuntime().CreateCampaign(ctx, spec)
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
	return s.CampaignRuntime().PauseCampaign(ctx, campaignID, reason)
}

func (s *Service) PreviewPauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (MutationPreview, error) {
	return s.previewPauseCampaign(ctx, campaignID, reason)
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
	return s.CampaignRuntime().ResumeCampaign(ctx, campaignID, reason)
}

func (s *Service) PreviewResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (MutationPreview, error) {
	return s.previewResumeCampaign(ctx, campaignID, reason)
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
	if err := s.enforceCampaignPublishGate(ctx, campaignID, camp, false); err != nil {
		return MutationPreview{}, err
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
	return s.CampaignRuntime().UpdateCampaignSchedule(ctx, campaignID, startAt, endAt, daypartHours)
}

func (s *Service) transitionCampaignStatus(ctx context.Context, q db.Querier, campaignID uuid.UUID, old, newStatus db.CampaignStatusType, reason string, budget int64) error {
	_, err := q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		ID:     domain.ToUUID(campaignID),
		Status: newStatus,
	})
	if err != nil {
		return err
	}
	err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
		CampaignID: domain.ToUUID(campaignID),
		OldStatus:  db.NullCampaignStatusType{CampaignStatusType: old, Valid: true},
		NewStatus:  newStatus,
		Reason:     pgtype.Text{String: reason, Valid: true},
	})
	if err != nil {
		return err
	}
	return s.emitCampaignLifecycleOutbox(ctx, q, campaignID, newStatus, budget)
}

