package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ad-event-processor/internal/campaign"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/supply"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"
	"ad-event-processor/pkg/proxyupstream"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type auditCreateCampaignChange struct {
	Name         string                `json:"name"`
	BudgetLimit  int64                 `json:"budget_limit"`
	Status       db.CampaignStatusType `json:"status"`
	StartAt      *time.Time            `json:"start_at,omitempty"`
	EndAt        *time.Time            `json:"end_at,omitempty"`
	DaypartHours []int16               `json:"daypart_hours,omitempty"`
}

func createCampaign(ctx context.Context, pool *pgxpool.Pool, fx campaign.Effects, spec campaign.CreateCampaignSpec) (uuid.UUID, error) {
	if err := campaign.ValidateDaypartHours(spec.DaypartHours); err != nil {
		return uuid.Nil, err
	}
	if err := campaign.ValidateSchedule(spec.StartAt, spec.EndAt); err != nil {
		return uuid.Nil, err
	}
	if err := fx.EnforceDeploymentLicenseCampaignCap(ctx); err != nil {
		return uuid.Nil, err
	}
	if pool == nil {
		return uuid.Nil, campaign.ErrServiceUnavailable()
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
	initialStatus := campaign.ResolveScheduleStatus(now, spec.StartAt, spec.EndAt)

	// One PG txn: idempotency ledger, customer balance freeze, campaign insert, FREEZE ledger,
	// status history, audit, and lifecycle outbox (Effects.EmitCampaignLifecycleOutbox).
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		// Idempotency: ledger hash replays return existing campaign_id; row without campaign_id fails closed.
		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: spec.IdempotencyKey, Valid: true})
		if err == nil {
			if existing.CampaignID.Valid {
				campaignID = uuid.UUID(existing.CampaignID.Bytes)
				return nil
			}
			return fmt.Errorf("%w ledger row for key %q", campaign.ErrIncompleteIdempotency, spec.IdempotencyKey)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("idempotency lookup failed: %w", err)
		}
		cust, err := q.GetCustomerForUpdate(ctx, domain.ToUUID(spec.CustomerID))
		if err != nil {
			return campaign.MapCampaignNotFound(err, campaign.ErrCustomerNotFound)
		}
		if cust.Balance+cust.AllowedOverdraft < spec.BudgetLimitMicro {
			return campaign.ErrInsufficientBalance
		}

		var brandIDParam pgtype.UUID
		// brandFcapKey is stored on the campaign row; outbox worker mirrors it to Redis fcap hash.
		brandFcapKey := "fcap:c:" + campaignID.String()
		if spec.BrandID != nil {
			brand, err := q.GetBrand(ctx, domain.ToUUID(*spec.BrandID))
			if err != nil {
				return campaign.MapCampaignNotFound(err, campaign.ErrBrandNotFound)
			}
			if uuid.UUID(brand.CustomerID.Bytes) != spec.CustomerID {
				return campaign.ErrBrandBelongsToAnotherCustomer
			}
			brandIDParam = domain.ToUUID(*spec.BrandID)
			brandFcapKey = "fcap:b:" + spec.BrandID.String()
		}

		var templateIDParam pgtype.UUID
		if spec.TemplateID != nil {
			templateIDParam = domain.ToUUID(*spec.TemplateID)
		}

		// Budget limit is prepaid: debit customer balance; FREEZE ledger row pairs with campaign row.
		if _, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(spec.CustomerID),
			Balance: -spec.BudgetLimitMicro,
		}); err != nil {
			return err
		}

		if _, err = q.CreateCampaign(ctx, db.CreateCampaignParams{
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
			TargetCountries: campaign.CountriesOrEmpty(spec.TargetCountries),
			BrandID:         brandIDParam,
			BrandFcapKey:    brandFcapKey,
			StartAt:         campaign.ToTimestamptz(spec.StartAt),
			EndAt:           campaign.ToTimestamptz(spec.EndAt),
			DaypartHours:    campaign.DaypartOrEmpty(spec.DaypartHours),
			TemplateID:      templateIDParam,
		}); err != nil {
			return err
		}

		if _, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(spec.CustomerID),
			CampaignID:      domain.ToUUID(campaignID),
			Amount:          spec.BudgetLimitMicro,
			Type:            db.LedgerTypeFREEZE,
			IdempotencyHash: pgtype.Text{String: spec.IdempotencyKey, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		}); err != nil {
			return err
		}

		if err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			NewStatus:  initialStatus,
			Reason:     pgtype.Text{String: "Campaign creation", Valid: true},
		}); err != nil {
			return err
		}

		adminID := uuid.Nil
		if user, ok := authz.GetUser(ctx); ok {
			adminID = user.UserID
		}
		fx.AuditLog(ctx, q, adminID, "CREATE_CAMPAIGN", "campaign", &campaignID, auditCreateCampaignChange{
			Name:         spec.Name,
			BudgetLimit:  spec.BudgetLimitMicro,
			Status:       initialStatus,
			StartAt:      spec.StartAt,
			EndAt:        spec.EndAt,
			DaypartHours: spec.DaypartHours,
		}, campaign.AuditIdempotencyMeta{IdempotencyKey: spec.IdempotencyKey})

		// Lifecycle outbox: CREATE_CAMPAIGN when initial status is ACTIVE; PAUSE when scheduled PAUSED.
		return fx.EmitCampaignLifecycleOutbox(ctx, q, campaignID, initialStatus, spec.BudgetLimitMicro)
	})
	return campaignID, err
}

func listCampaignEvents(
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignID uuid.UUID,
	limit, offset int32,
) ([]campaign.CampaignEventDTO, int64, error) {
	if pool == nil {
		return nil, 0, fmt.Errorf("service unavailable")
	}
	q := db.New(pool)
	cid := domain.ToUUID(campaignID)
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountCampaignEvents(ctx, cid) },
		func() ([]db.ListCampaignEventsRow, error) {
			return q.ListCampaignEvents(ctx, db.ListCampaignEventsParams{
				CampaignID: cid,
				Limit:      limit,
				Offset:     offset,
			})
		},
		campaignEventToDTO,
	)
}

func campaignEventToDTO(row db.ListCampaignEventsRow) campaign.CampaignEventDTO {
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
	return campaign.CampaignEventDTO{
		ClickID:   row.ClickID,
		EventType: row.EventType,
		UserID:    userID,
		IP:        ip,
		UserAgent: ua,
		Payload:   json.RawMessage(row.Payload),
		CreatedAt: createdAt,
	}
}

func pauseCampaign(ctx context.Context, pool *pgxpool.Pool, fx campaign.Effects, campaignID uuid.UUID, reason string) error {
	if pool == nil || fx == nil {
		return campaign.ErrServiceUnavailable()
	}
	// PAUSE_CAMPAIGN outbox row is inserted in the same PG txn as status_history (Effects.EnqueueCampaignOutbox).
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}
		if err := campaign.AssertMediaBuyerCampaignAccess(ctx, camp); err != nil {
			return err
		}
		// Idempotent pause: already PAUSED skips outbox enqueue.
		if camp.Status == db.CampaignStatusTypePAUSED {
			return nil
		}
		if camp.Status != db.CampaignStatusTypeACTIVE {
			return fmt.Errorf("%w in status %s", campaign.ErrCampaignCannotBePaused, camp.Status)
		}
		if _, err := q.PauseCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
			return err
		}
		if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: camp.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypePAUSED,
			Reason:     pgtype.Text{String: reason, Valid: reason != ""},
		}); err != nil {
			return err
		}
		adminID := uuid.Nil
		if user, ok := authz.GetUser(ctx); ok {
			adminID = user.UserID
		}
		fx.AuditLog(ctx, q, adminID, "PAUSE_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
		return fx.EnqueueCampaignOutbox(ctx, q, "PAUSE_CAMPAIGN", campaignID, camp.BudgetLimit)
	})
}

func resumeCampaign(ctx context.Context, pool *pgxpool.Pool, fx campaign.Effects, campaignID uuid.UUID, reason string, publishForce bool) error {
	if pool == nil || fx == nil {
		return campaign.ErrServiceUnavailable()
	}
	// RESUME_CAMPAIGN outbox mirrors pause: same PG txn as status row + EnqueueCampaignOutbox.
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}
		if err := campaign.AssertMediaBuyerCampaignAccess(ctx, camp); err != nil {
			return err
		}
		if camp.Status != db.CampaignStatusTypePAUSED {
			return campaign.ErrCampaignNotPaused
		}
		now := time.Now()
		var startAt, endAt *time.Time
		if camp.StartAt.Valid {
			startAt = &camp.StartAt.Time
		}
		if camp.EndAt.Valid {
			endAt = &camp.EndAt.Time
		}
		if campaign.ResolveScheduleStatus(now, startAt, endAt) != db.CampaignStatusTypeACTIVE {
			return campaign.ErrCampaignOutsideSchedule
		}
		// Publish gate (flow paths, integration blockers) before resume; force skips warnings only.
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, camp, publishForce); err != nil {
			return err
		}
		if _, err := q.ResumeCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
			return err
		}
		if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(campaignID),
			OldStatus:  db.NullCampaignStatusType{CampaignStatusType: camp.Status, Valid: true},
			NewStatus:  db.CampaignStatusTypeACTIVE,
			Reason:     pgtype.Text{String: reason, Valid: reason != ""},
		}); err != nil {
			return err
		}
		adminID := uuid.Nil
		if user, ok := authz.GetUser(ctx); ok {
			adminID = user.UserID
		}
		fx.AuditLog(ctx, q, adminID, "RESUME_CAMPAIGN", "campaign", &campaignID, auditReasonChange{Reason: reason}, nil)
		return fx.EnqueueCampaignOutbox(ctx, q, "RESUME_CAMPAIGN", campaignID, camp.BudgetLimit)
	})
}

type auditReasonChange struct {
	Reason string `json:"reason"`
}

func listCampaigns(
	ctx context.Context,
	pool *pgxpool.Pool,
	effects campaign.Effects,
	customerID uuid.UUID,
	status string,
	limit, offset int32,
) ([]campaign.CampaignDTO, int64, error) {
	if pool == nil {
		return nil, 0, fmt.Errorf("service unavailable")
	}
	q := db.New(pool)

	var cid pgtype.UUID
	if customerID != uuid.Nil {
		cid = domain.ToUUID(customerID)
	}

	var st pgtype.Text
	if status != "" {
		st = pgtype.Text{String: status, Valid: true}
	}

	countParams := db.CountCampaignsParams{
		CustomerID:  cid,
		Status:      st,
		OwnerUserID: campaign.CampaignOwnerUserFilter(ctx),
	}
	listParams := db.ListCampaignsParams{
		Limit:       limit,
		Offset:      offset,
		CustomerID:  cid,
		Status:      st,
		OwnerUserID: campaign.CampaignOwnerUserFilter(ctx),
	}

	items, total, err := coldpath.PaginatedList(
		func() (int64, error) { return q.CountCampaigns(ctx, countParams) },
		func() ([]db.Campaign, error) { return q.ListCampaigns(ctx, listParams) },
		func(c db.Campaign) campaign.CampaignDTO { return scrubCampaignDTO(ctx, c) },
	)
	if err != nil {
		return nil, 0, err
	}
	if effects != nil {
		effects.AttachCampaignListBudgetApprovalStates(ctx, items)
	}
	return items, total, nil
}

func getCampaign(
	ctx context.Context,
	pool *pgxpool.Pool,
	effects campaign.Effects,
	campaignID uuid.UUID,
) (campaign.CampaignDTO, error) {
	if pool == nil {
		return campaign.CampaignDTO{}, fmt.Errorf("service unavailable")
	}
	q := db.New(pool)
	c, err := q.GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return campaign.CampaignDTO{}, mapCampaignStoreError(err)
	}
	if err := campaign.AssertMediaBuyerCampaignAccess(ctx, c); err != nil {
		return campaign.CampaignDTO{}, err
	}
	dto := scrubCampaignDTO(ctx, c)
	if effects != nil {
		if flowID, flowErr := effects.CampaignFlowID(ctx, campaignID); flowErr == nil {
			dto.FlowID = flowID
		}
		effects.AttachCampaignBudgetApprovalState(ctx, &dto)
	}
	return dto, nil
}

func mapCampaignStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return campaign.ErrCampaignNotFound
	}
	return err
}

func formatCampaignMicro(m int64) string {
	return money.FormatFixed2(m)
}

func formatCampaignOptionalTime(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func formatCampaignOptionalUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func ScrubCampaignFields(c campaign.CampaignDTO, level authz.MaskLevel) campaign.CampaignDTO {
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
		out.BudgetLimitDisplay = RedactedMoneyDisplay()
	}
	if out.DailyBudget != "" {
		redacted = append(redacted, "daily_budget")
		out.DailyBudget = ""
		out.DailyBudgetDisplay = RedactedMoneyDisplay()
	}
	out.FieldsRedacted = redacted
	return out
}

func RedactedMoneyDisplay() string {
	return "—"
}

func scrubCampaignDTO(ctx context.Context, c db.Campaign) campaign.CampaignDTO {
	countries := c.TargetCountries
	if countries == nil {
		countries = []string{}
	}
	dto := campaign.CampaignDTO{
		ID:                         uuid.UUID(c.ID.Bytes).String(),
		Name:                       c.Name,
		Status:                     string(c.Status),
		BudgetLimit:                formatCampaignMicro(c.BudgetLimit),
		CurrentSpend:               formatCampaignMicro(c.CurrentSpend),
		CustomerID:                 uuid.UUID(c.CustomerID.Bytes).String(),
		PacingMode:                 string(c.PacingMode),
		DailyBudget:                formatCampaignMicro(c.DailyBudget),
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
		BrandID:                    formatCampaignOptionalUUID(c.BrandID),
		CreativePayload:            json.RawMessage(c.CreativePayload),
		ReferrerFilter:             c.ReferrerFilter,
		StartAt:                    formatCampaignOptionalTime(c.StartAt),
		EndAt:                      formatCampaignOptionalTime(c.EndAt),
		DaypartHours:               campaign.DaypartOrEmpty(c.DaypartHours),
		OwnerUserID:                formatCampaignOptionalUUID(c.OwnerUserID),
		TrafficTemplateID:          campaign.FormatOptionalText(c.TrafficTemplateID),
		ClickQueryParams:           campaign.ClickQueryParamsFromRaw(c.ClickQueryParams),
		CreatedAt:                  c.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:                  c.UpdatedAt.Time.Format(time.RFC3339),
		Revision:                   campaign.CampaignRevision(c.UpdatedAt.Time.Format(time.RFC3339)),
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
			dto.IngressCostConfig = &campaign.IngressCostConfigDTO{
				Param:    param,
				Scale:    scale,
				MaxMicro: parsed.MaxMicro,
				Policy:   policy,
			}
		}
	}
	// RBAC mask from request context: partial mask redacts URL/budget fields in list/get JSON.
	if snap, ok := authz.SnapshotFromContext(ctx); ok {
		out := ScrubCampaignFields(dto, snap.Mask)
		campaign.AttachCampaignPresentation(ctx, &out)
		return out
	}
	campaign.AttachCampaignPresentation(ctx, &dto)
	return dto
}

type pacingOutboxPayload struct {
	CampaignID string `json:"campaign_id"`
	PacingMode string `json:"pacing_mode"`
}

type auditPacingChange struct {
	OldPacingMode string `json:"old_pacing_mode"`
	NewPacingMode string `json:"new_pacing_mode"`
}

func updateCampaignPacing(ctx context.Context, pool *pgxpool.Pool, fx campaign.Effects, campaignID uuid.UUID, newMode string) (campaign.CampaignDTO, error) {
	pacing, err := parsePacingMode(newMode)
	if err != nil {
		return campaign.CampaignDTO{}, err
	}
	if pool == nil || fx == nil {
		return campaign.CampaignDTO{}, campaign.ErrServiceUnavailable()
	}

	var updatedCamp db.Campaign
	// Pacing mode PG update and UPDATE_CAMPAIGN_PACING outbox row share one txn for Redis mirror.
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}

		updatedCamp, err = q.UpdateCampaignPacing(ctx, db.UpdateCampaignPacingParams{
			ID:         domain.ToUUID(campaignID),
			PacingMode: pacing,
		})
		if err != nil {
			return fmt.Errorf("failed to update campaign pacing: %w", err)
		}

		adminID := uuid.Nil
		if user, ok := authz.GetUser(ctx); ok {
			adminID = user.UserID
		}

		fx.AuditLog(ctx, q, adminID, "UPDATE_CAMPAIGN_PACING", "campaign", &campaignID, auditPacingChange{
			OldPacingMode: string(camp.PacingMode),
			NewPacingMode: string(pacing),
		}, nil)

		payloadBytes, err := coldpath.MarshalOutbox(pacingOutboxPayload{
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
		return campaign.CampaignDTO{}, err
	}

	return scrubCampaignDTO(ctx, updatedCamp), nil
}

func parsePacingMode(newMode string) (db.PacingModeType, error) {
	switch newMode {
	case "ASAP":
		return db.PacingModeTypeASAP, nil
	case "EVEN", "off", "OFF":
		return db.PacingModeTypeEVEN, nil
	case "VPP", "vpp":
		return db.PacingModeTypeVPP, nil
	default:
		return "", fmt.Errorf("%w: %s", campaign.ErrInvalidPacingMode, newMode)
	}
}

func patchCampaign(ctx context.Context, pool *pgxpool.Pool, fx campaign.Effects, campaignID uuid.UUID, req campaign.PatchCampaignRequest) (campaign.CampaignDTO, error) {
	camp, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return campaign.CampaignDTO{}, err
	}
	if err := campaign.AssertMediaBuyerCampaignAccess(ctx, camp); err != nil {
		return campaign.CampaignDTO{}, err
	}
	// Optimistic concurrency: If-Match revision is campaigns.updated_at as RFC3339 (CampaignRevision).
	if req.ExpectedRevision != nil {
		currentRev := campaign.CampaignRevision(camp.UpdatedAt.Time.Format(time.RFC3339))
		if currentRev != strings.TrimSpace(*req.ExpectedRevision) {
			return campaign.CampaignDTO{}, campaign.ErrCampaignRevisionConflict
		}
	}

	// Flow, brand, ingress-cost, and click-preset patches run via Effects outside the PG txn below.
	if req.FlowID != nil {
		if err := fx.AssignCampaignFlow(ctx, campaignID, *req.FlowID); err != nil {
			return campaign.CampaignDTO{}, err
		}
	}

	if req.IngressCostConfig != nil {
		if err := fx.ApplyCampaignIngressCostPatch(ctx, campaignID, *req.IngressCostConfig); err != nil {
			return campaign.CampaignDTO{}, err
		}
	}

	clickPresetPatch := req.TrafficTemplateID != nil || req.ClickQueryParams != nil
	if clickPresetPatch {
		if err := fx.ApplyCampaignClickPresetPatch(ctx, campaignID, req.TrafficTemplateID, req.ClickQueryParams); err != nil {
			return campaign.CampaignDTO{}, err
		}
	}

	if req.BrandID != nil {
		if err := fx.AssignCampaignBrand(ctx, campaignID, *req.BrandID); err != nil {
			return campaign.CampaignDTO{}, err
		}
	}

	if req.PacingMode != nil {
		if _, err := updateCampaignPacing(ctx, pool, fx, campaignID, *req.PacingMode); err != nil {
			return campaign.CampaignDTO{}, err
		}
	}

	budgetMicro, err := campaign.ResolvePatchBudgetLimitMicro(req)
	if err != nil {
		return campaign.CampaignDTO{}, err
	}
	statusWant, statusSet, err := campaign.ParsePatchStatus(req.Status)
	if err != nil {
		return campaign.CampaignDTO{}, err
	}
	schedulePatch := req.StartAt != nil || req.EndAt != nil || req.DaypartHours != nil

	adminPatch := req.Name != nil || req.DailyBudgetMicro != nil || req.Timezone != nil ||
		req.FreqLimit != nil || req.FreqWindow != nil || req.TargetCountries != nil ||
		req.TargetURL != nil || req.ReferrerFilter != nil ||
		req.SafePageURL != nil || req.SafePageEnabled != nil || req.AttestationEnabled != nil || req.AttestationMode != nil || req.AttestationTTLSec != nil || req.DmrEnabled != nil ||
		req.CIDRBlockEnabled != nil || req.ProxyVPNBlockEnabled != nil || req.ModeratorIntelEnabled != nil ||
		req.ReviewTrafficAction != nil ||
		req.TLSFingerprintBlockEnabled != nil || req.ConnTypePolicy != nil ||
		req.LinkSigningEnabled != nil || req.LinkSigningTTLSec != nil ||
		req.ClickDelivery != nil || req.ProxyUpstreamURL != nil || req.ProxyRewriteAssets != nil
	if !adminPatch && budgetMicro == nil && !statusSet && !schedulePatch && !clickPresetPatch {
		return getCampaign(ctx, pool, fx, campaignID)
	}

	var updated db.Campaign
	// Locked patch txn: budget/schedule/status Effects helpers enqueue outbox_events on this tx
	// when Redis campaign config must change (CREATE_CAMPAIGN, PAUSE_CAMPAIGN, etc.).
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}

		if budgetMicro != nil {
			if err := fx.ApplyCampaignBudgetPatch(ctx, q, locked, *budgetMicro); err != nil {
				return err
			}
			locked, err = q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
			if err != nil {
				return err
			}
		}

		if schedulePatch {
			startAt := campaign.TimestamptzPtr(locked.StartAt)
			endAt := campaign.TimestamptzPtr(locked.EndAt)
			if req.StartAt != nil {
				startAt = req.StartAt
			}
			if req.EndAt != nil {
				endAt = req.EndAt
			}
			daypart := locked.DaypartHours
			if req.DaypartHours != nil {
				daypart = req.DaypartHours
			}
			if err := fx.ApplyCampaignSchedulePatch(ctx, q, campaignID, locked, startAt, endAt, daypart); err != nil {
				return err
			}
			locked, err = q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
			if err != nil {
				return err
			}
		}

		if adminPatch {
			name := locked.Name
			if req.Name != nil {
				name = strings.TrimSpace(*req.Name)
				if name == "" {
					return fmt.Errorf("name is required")
				}
			}
			dailyBudget := locked.DailyBudget
			if req.DailyBudgetMicro != nil {
				if *req.DailyBudgetMicro < 0 {
					return fmt.Errorf("invalid daily_budget")
				}
				dailyBudget = *req.DailyBudgetMicro
			}
			timezone := locked.Timezone
			if req.Timezone != nil {
				timezone = strings.TrimSpace(*req.Timezone)
				if timezone == "" {
					timezone = "UTC"
				}
			}
			freqLimit := locked.FreqLimit
			if req.FreqLimit != nil {
				freqLimit = pgtype.Int4{Int32: *req.FreqLimit, Valid: true}
			}
			freqWindow := locked.FreqWindow
			if req.FreqWindow != nil {
				freqWindow = pgtype.Int4{Int32: *req.FreqWindow, Valid: true}
			}
			countries := locked.TargetCountries
			if req.TargetCountries != nil {
				countries = campaign.CountriesOrEmpty(req.TargetCountries)
			}
			targetURL := locked.TargetUrl
			if req.TargetURL != nil {
				targetURL = *req.TargetURL
			}
			referrerFilter := locked.ReferrerFilter
			if req.ReferrerFilter != nil {
				referrerFilter = *req.ReferrerFilter
			}
			safePageURL := locked.SafePageUrl
			if req.SafePageURL != nil {
				safePageURL = *req.SafePageURL
			}
			safePageEnabled := locked.SafePageEnabled
			if req.SafePageEnabled != nil {
				safePageEnabled = *req.SafePageEnabled
			}
			attestationEnabled := locked.AttestationEnabled
			if req.AttestationEnabled != nil {
				attestationEnabled = *req.AttestationEnabled
			}
			attestationMode := locked.AttestationMode
			if req.AttestationMode != nil {
				parsedMode, _, err := campaign.ParsePatchAttestationMode(req.AttestationMode)
				if err != nil {
					return err
				}
				attestationMode = string(parsedMode)
			}
			if safePageEnabled && !locked.SafePageEnabled && req.AttestationMode == nil && req.AttestationEnabled == nil {
				attestationMode = string(domain.AttestationModeLight)
			}
			resolvedMode := domain.ResolveAttestationMode(domain.ParseAttestationMode(attestationMode), attestationEnabled)
			attestationMode = string(resolvedMode)
			attestationEnabled = resolvedMode.RequiresProbe()
			attestationTTL := locked.AttestationTtlSec
			if req.AttestationTTLSec != nil {
				parsed, _, err := campaign.ParsePatchAttestationTTLSec(req.AttestationTTLSec)
				if err != nil {
					return err
				}
				attestationTTL = parsed
			}
			if attestationEnabled && !safePageEnabled {
				return fmt.Errorf("attestation_enabled requires safe_page_enabled")
			}
			if resolvedMode.RequiresProbe() && !safePageEnabled {
				return fmt.Errorf("attestation_mode requires safe_page_enabled")
			}
			dmrEnabled := locked.DmrEnabled
			if req.DmrEnabled != nil {
				dmrEnabled = *req.DmrEnabled
			}
			cidrBlock := locked.CidrBlockEnabled
			if req.CIDRBlockEnabled != nil {
				cidrBlock = *req.CIDRBlockEnabled
			}
			proxyVPNBlock := locked.ProxyVpnBlockEnabled
			if req.ProxyVPNBlockEnabled != nil {
				proxyVPNBlock = *req.ProxyVPNBlockEnabled
			}
			moderatorIntel := locked.ModeratorIntelEnabled
			if req.ModeratorIntelEnabled != nil {
				moderatorIntel = *req.ModeratorIntelEnabled
			}
			reviewTrafficAction := locked.ReviewTrafficAction
			if req.ReviewTrafficAction != nil {
				parsed := domain.ParseReviewTrafficAction(*req.ReviewTrafficAction)
				if !parsed.Valid() {
					return fmt.Errorf("invalid review_traffic_action")
				}
				reviewTrafficAction = string(parsed)
			}
			tlsFingerprintBlock := locked.TlsFingerprintBlockEnabled
			if req.TLSFingerprintBlockEnabled != nil {
				tlsFingerprintBlock = *req.TLSFingerprintBlockEnabled
			}
			connTypePolicy := locked.ConnTypePolicy
			if req.ConnTypePolicy != nil {
				parsed, _, err := campaign.ParsePatchConnTypePolicy(req.ConnTypePolicy)
				if err != nil {
					return err
				}
				connTypePolicy = parsed
			}
			linkSigningEnabled := locked.LinkSigningEnabled
			if req.LinkSigningEnabled != nil {
				linkSigningEnabled = *req.LinkSigningEnabled
			}
			linkSigningTTL := locked.LinkSigningTtlSec
			if req.LinkSigningTTLSec != nil {
				parsed, _, err := campaign.ParsePatchLinkSigningTTLSec(req.LinkSigningTTLSec)
				if err != nil {
					return err
				}
				linkSigningTTL = parsed
			}
			clickDelivery := locked.ClickDelivery
			if req.ClickDelivery != nil {
				clickDelivery = strings.TrimSpace(*req.ClickDelivery)
			}
			if clickDelivery == "" {
				clickDelivery = proxyupstream.ClickDeliveryRedirect
			}
			proxyUpstream := locked.ProxyUpstreamUrl
			if req.ProxyUpstreamURL != nil {
				proxyUpstream = strings.TrimSpace(*req.ProxyUpstreamURL)
			}
			proxyRewrite := locked.ProxyRewriteAssets
			if req.ProxyRewriteAssets != nil {
				proxyRewrite = *req.ProxyRewriteAssets
			}
			allowHTTP := fx.ProxyAllowHTTPInsecure()
			if err := proxyupstream.ValidateDeliveryPair(ctx, clickDelivery, proxyUpstream, allowHTTP); err != nil {
				return err
			}

			locked, err = q.UpdateCampaignAdmin(ctx, db.UpdateCampaignAdminParams{
				ID:                         domain.ToUUID(campaignID),
				Name:                       name,
				DailyBudget:                dailyBudget,
				Timezone:                   timezone,
				FreqLimit:                  freqLimit,
				FreqWindow:                 freqWindow,
				TargetCountries:            countries,
				TargetUrl:                  targetURL,
				ReferrerFilter:             referrerFilter,
				SafePageUrl:                safePageURL,
				SafePageEnabled:            safePageEnabled,
				AttestationEnabled:         attestationEnabled,
				AttestationTtlSec:          attestationTTL,
				AttestationMode:            attestationMode,
				DmrEnabled:                 dmrEnabled,
				ClickDelivery:              clickDelivery,
				ProxyUpstreamUrl:           proxyUpstream,
				ProxyRewriteAssets:         proxyRewrite,
				TlsFingerprintBlockEnabled: tlsFingerprintBlock,
				ConnTypePolicy:             connTypePolicy,
				LinkSigningEnabled:         linkSigningEnabled,
				LinkSigningTtlSec:          linkSigningTTL,
				CidrBlockEnabled:           cidrBlock,
				ProxyVpnBlockEnabled:       proxyVPNBlock,
				ModeratorIntelEnabled:      moderatorIntel,
				ReviewTrafficAction:        reviewTrafficAction,
			})
			if err != nil {
				return err
			}

			var uid uuid.UUID
			if u, ok := authz.GetUser(ctx); ok {
				uid = u.UserID
			}
			fx.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, map[string]any{
				"name":             name,
				"daily_budget":     dailyBudget,
				"timezone":         timezone,
				"target_countries": countries,
			}, nil)
		}

		if statusSet {
			if err := fx.ApplyCampaignStatusPatch(ctx, q, locked, statusWant, "patch", req.PublishForce); err != nil {
				return err
			}
		}

		updated, err = q.GetCampaign(ctx, domain.ToUUID(campaignID))
		return err
	})
	if err != nil {
		return campaign.CampaignDTO{}, err
	}

	// Post-commit: INCR campaign epoch + PUBLISH on every Redis shard (tracker registry reload).
	// Complements outbox worker apply; not rolled back if PG txn already committed.
	fx.PublishCampaignUpdate(ctx, campaignID.String())
	return scrubCampaignDTO(ctx, updated), nil
}

func evaluateCampaignPublish(ctx context.Context, fx campaign.Effects, campaignID uuid.UUID) (campaign.CampaignPublishCheckDTO, error) {
	if fx == nil {
		return campaign.CampaignPublishCheckDTO{}, fmt.Errorf("service unavailable")
	}
	row, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return campaign.CampaignPublishCheckDTO{}, err
	}
	if err := campaign.AssertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return campaign.CampaignPublishCheckDTO{}, err
	}
	blocked, err := campaign.CollectPublishBlocked(ctx, fx, campaignID, row)
	if err != nil {
		return campaign.CampaignPublishCheckDTO{}, err
	}
	if blocked == nil {
		return campaign.CampaignPublishCheckDTO{Valid: true}, nil
	}
	return campaign.CampaignPublishCheckDTO{
		Valid:        false,
		FieldErrors:  blocked.FieldErrors,
		WarningSlugs: blocked.WarningSlugs,
	}, nil
}

func publishCampaign(ctx context.Context, pool *pgxpool.Pool, fx campaign.Effects, campaignID uuid.UUID, force bool) (campaign.CampaignDTO, error) {
	if pool == nil || fx == nil {
		return campaign.CampaignDTO{}, fmt.Errorf("service unavailable")
	}
	if force && !campaign.CanForceCampaignPublish(ctx) {
		return campaign.CampaignDTO{}, campaign.ErrValidationf("force publish requires admin role")
	}
	row, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return campaign.CampaignDTO{}, err
	}
	if err := campaign.AssertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return campaign.CampaignDTO{}, err
	}
	// Publish is idempotent: ACTIVE re-runs gate only; PAUSED resumes when gate passes.
	switch row.Status {
	case db.CampaignStatusTypeACTIVE:
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, row, force); err != nil {
			return campaign.CampaignDTO{}, err
		}
		return getCampaign(ctx, pool, fx, campaignID)
	case db.CampaignStatusTypePAUSED:
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, row, force); err != nil {
			return campaign.CampaignDTO{}, err
		}
		if err := fx.ResumeCampaignForPublish(ctx, campaignID, force); err != nil {
			return campaign.CampaignDTO{}, err
		}
		return getCampaign(ctx, pool, fx, campaignID)
	default:
		return campaign.CampaignDTO{}, campaign.ErrValidationf("campaign cannot be published from current status")
	}
}

func updateCampaignSchedule(
	ctx context.Context,
	pool *pgxpool.Pool,
	fx campaign.Effects,
	campaignID uuid.UUID,
	startAt, endAt *time.Time,
	daypartHours []int16,
) error {
	if err := campaign.ValidateDaypartHours(daypartHours); err != nil {
		return err
	}
	if err := campaign.ValidateSchedule(startAt, endAt); err != nil {
		return err
	}
	if pool == nil {
		return campaign.ErrServiceUnavailable()
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}
		// ApplyCampaignSchedulePatch may auto-pause/resume and enqueue UPDATE_CAMPAIGN_SCHEDULE outbox.
		return fx.ApplyCampaignSchedulePatch(ctx, q, campaignID, locked, startAt, endAt, daypartHours)
	})
}

const clickHouseStaleThreshold = 5 * time.Minute

const clickhouseStatsTimeout = 10 * time.Second

type clickhouseLagCache struct {
	mu      sync.Mutex
	lag     time.Duration
	updated time.Time
}

const clickhouseLagCacheTTL = 30 * time.Second

// Process-wide CH ingestion lag cache; stale=true when lag exceeds clickHouseStaleThreshold (5 min).
var globalClickHouseLagCache clickhouseLagCache

func getCampaignStats(
	ctx context.Context,
	pool *pgxpool.Pool,
	clickhouseQuery *database.ClickHouseQuery,
	campaignID uuid.UUID,
	from, to time.Time,
	granularity string,
) (campaign.CampaignStatsDTO, error) {
	if granularity != "hour" && granularity != "day" {
		return campaign.CampaignStatsDTO{}, fmt.Errorf("%w: %s", campaign.ErrUnsupportedGranularity, granularity)
	}
	if !to.After(from) {
		return campaign.CampaignStatsDTO{}, fmt.Errorf("%w: to must be after from", campaign.ErrInvalidTimeRange)
	}
	if pool == nil {
		return campaign.CampaignStatsDTO{}, campaign.ErrServiceUnavailable()
	}

	q := db.New(pool)
	camp, err := q.GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return campaign.CampaignStatsDTO{}, mapCampaignStoreError(err)
	}

	stats, err := q.SumCampaignStatsInRange(ctx, db.SumCampaignStatsInRangeParams{
		CampaignID: domain.ToUUID(campaignID),
		FromDate:   pgtype.Date{Time: from.UTC(), Valid: true},
		ToDate:     pgtype.Date{Time: to.UTC(), Valid: true},
	})
	if err != nil {
		return campaign.CampaignStatsDTO{}, err
	}

	// PG campaign_stats rollup is strong-consistency default; CH buckets overlay below when wired.
	report := campaign.CampaignStatsDTO{
		CampaignID:   campaignID.String(),
		CurrentSpend: formatCampaignMicro(camp.CurrentSpend),
		Metrics: campaign.CampaignMetricsDTO{
			Impressions: stats.Impressions,
			Clicks:      stats.Clicks,
			Conversions: stats.Conversions,
		},
		Hourly:      []campaign.CampaignHourlyBucketDTO{},
		Daily:       []campaign.CampaignDailyBucketDTO{},
		Granularity: granularity,
		From:        from.UTC().Format(time.RFC3339),
		To:          to.UTC().Format(time.RFC3339),
		Stale:       true,
		Source:      "pg",
		Consistency: "strong",
	}

	if clickhouseQuery == nil {
		if granularity == "hour" {
			report.Hourly = synthesizeHourlyBuckets(
				campaignID,
				stats.Impressions,
				stats.Clicks,
				stats.Conversions,
				to,
			)
		}
		return report, nil
	}

	clickhouseCtx, cancel := context.WithTimeout(ctx, clickhouseStatsTimeout)
	defer cancel()

	var lag time.Duration
	if granularity == "hour" {
		hourly, lagVal, err := queryClickHouseHourly(clickhouseCtx, clickhouseQuery, campaignID, from, to)
		if err != nil {
			return campaign.CampaignStatsDTO{}, err
		}
		resolved := hourlyBucketsForReport(
			campaignID,
			stats.Impressions,
			stats.Clicks,
			stats.Conversions,
			to,
			hourly,
		)
		report.Hourly = resolved
		if hourlyBucketsHaveActivity(hourly) {
			lag = lagVal
			report.Consistency = "eventual"
			report.Source = "ch"
			report.Stale = lag > clickHouseStaleThreshold
		} else if hourlyBucketsHaveActivity(resolved) {
			report.Consistency = "strong"
			report.Source = "pg"
			report.Stale = true
		} else {
			lag = lagVal
			report.Consistency = "eventual"
			report.Source = "ch"
			report.Stale = lag > clickHouseStaleThreshold
		}
	} else {
		daily, lagVal, err := queryClickHouseDaily(clickhouseCtx, clickhouseQuery, campaignID, from, to)
		if err != nil {
			return campaign.CampaignStatsDTO{}, err
		}
		report.Daily = daily
		lag = lagVal
	}
	if granularity != "hour" {
		report.Consistency = "eventual"
		report.Source = "ch"
		report.Stale = lag > clickHouseStaleThreshold
	}
	return report, nil
}

func queryClickHouseHourly(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignID uuid.UUID, from, to time.Time) ([]campaign.CampaignHourlyBucketDTO, time.Duration, error) {
	if clickhouseQuery == nil {
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

	rows, err := clickhouseQuery.Query(ctx, query,
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("clickhouse hourly query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	buckets := make([]campaign.CampaignHourlyBucketDTO, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.hour, &r.impressions, &r.clicks, &r.conversions); err != nil {
			return nil, 0, fmt.Errorf("clickhouse hourly scan: %w", err)
		}
		buckets = append(buckets, campaign.CampaignHourlyBucketDTO{
			Hour:        r.hour.UTC().Format(time.RFC3339),
			Impressions: int64(r.impressions),
			Clicks:      int64(r.clicks),
			Conversions: int64(r.conversions),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	lag, err := clickHouseIngestionLag(ctx, clickhouseQuery)
	if err != nil {
		return nil, 0, err
	}
	return buckets, lag, nil
}

func queryClickHouseDaily(ctx context.Context, clickhouseQuery *database.ClickHouseQuery, campaignID uuid.UUID, from, to time.Time) ([]campaign.CampaignDailyBucketDTO, time.Duration, error) {
	if clickhouseQuery == nil {
		return nil, 0, nil
	}
	type row struct {
		day         time.Time
		impressions uint64
		clicks      uint64
		conversions uint64
	}

	query := `
SELECT
 day,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions
FROM (
 SELECT day, impression_count AS impressions, toUInt64(0) AS clicks, toUInt64(0) AS conversions
 FROM mv_campaign_daily_impressions
 WHERE campaign_id = ? AND day >= toDate(?) AND day < toDate(?)
 UNION ALL
 SELECT day, toUInt64(0), click_count, toUInt64(0)
 FROM mv_campaign_daily_clicks
 WHERE campaign_id = ? AND day >= toDate(?) AND day < toDate(?)
 UNION ALL
 SELECT day, toUInt64(0), toUInt64(0), conversion_count
 FROM mv_campaign_daily_conversions
 WHERE campaign_id = ? AND day >= toDate(?) AND day < toDate(?)
) GROUP BY day ORDER BY day`

	rows, err := clickhouseQuery.Query(ctx, query,
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
		campaignID, from.UTC(), to.UTC(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("clickhouse daily query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	buckets := make([]campaign.CampaignDailyBucketDTO, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.day, &r.impressions, &r.clicks, &r.conversions); err != nil {
			return nil, 0, fmt.Errorf("clickhouse daily scan: %w", err)
		}
		buckets = append(buckets, campaign.CampaignDailyBucketDTO{
			Day:         r.day.UTC().Format("2006-01-02"),
			Impressions: int64(r.impressions),
			Clicks:      int64(r.clicks),
			Conversions: int64(r.conversions),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	lag, err := clickHouseIngestionLag(ctx, clickhouseQuery)
	if err != nil {
		return nil, 0, err
	}
	return buckets, lag, nil
}

func ClickHouseIngestionLag(ctx context.Context, clickhouseQuery *database.ClickHouseQuery) (time.Duration, error) {
	return clickHouseIngestionLag(ctx, clickhouseQuery)
}

func clickHouseIngestionLag(ctx context.Context, clickhouseQuery *database.ClickHouseQuery) (time.Duration, error) {
	if clickhouseQuery == nil {
		return 0, nil
	}

	// Double-checked lock: one max(created_at) probe per clickhouseLagCacheTTL across handlers.
	globalClickHouseLagCache.mu.Lock()
	if time.Since(globalClickHouseLagCache.updated) < clickhouseLagCacheTTL {
		lag := globalClickHouseLagCache.lag
		globalClickHouseLagCache.mu.Unlock()
		return lag, nil
	}
	globalClickHouseLagCache.mu.Unlock()

	var latest time.Time
	err := clickhouseQuery.QueryRow(ctx, `
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

func listStatusHistory(
	ctx context.Context,
	pool *pgxpool.Pool,
	campaignID uuid.UUID,
	limit, offset int32,
) ([]campaign.StatusHistoryDTO, int64, error) {
	if pool == nil {
		return nil, 0, campaign.ErrServiceUnavailable()
	}
	q := db.New(pool)
	cid := domain.ToUUID(campaignID)
	listParams := db.ListStatusHistoryParams{
		CampaignID: cid,
		Limit:      limit,
		Offset:     offset,
	}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountStatusHistory(ctx, cid) },
		func() ([]db.CampaignStatusHistory, error) { return q.ListStatusHistory(ctx, listParams) },
		campaign.StatusHistoryToDTO,
	)
}

type SupplyChainHost interface {
	MapCampaignNotFound(err error) error
	AuditSupplyChainUpdate(ctx context.Context, q db.Querier, campaignID uuid.UUID, oldNodesJSON, newNodesJSON []byte)
}

func GetCampaignSupplyChain(ctx context.Context, pool *pgxpool.Pool, host SupplyChainHost, campaignID uuid.UUID) (supply.CampaignChainDTO, error) {
	if pool == nil || host == nil {
		return supply.CampaignChainDTO{}, campaign.ErrServiceUnavailable()
	}
	row, err := db.New(pool).GetCampaignFull(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return supply.CampaignChainDTO{}, host.MapCampaignNotFound(err)
	}
	nodes, err := parseSupplyChainNodes(row.SupplyChainNodes)
	if err != nil {
		return supply.CampaignChainDTO{}, err
	}
	return supply.CampaignChainDTO{CampaignID: campaignID.String(), Nodes: nodes}, nil
}

func UpdateCampaignSupplyChain(ctx context.Context, pool *pgxpool.Pool, host SupplyChainHost, campaignID uuid.UUID, nodes []supply.ChainNode) (supply.CampaignChainDTO, error) {
	if err := supply.ValidateChainNodes(nodes); err != nil {
		return supply.CampaignChainDTO{}, err
	}
	if pool == nil || host == nil {
		return supply.CampaignChainDTO{}, campaign.ErrServiceUnavailable()
	}
	nodesJSON, err := coldpath.MarshalJSON(nodes)
	if err != nil {
		return supply.CampaignChainDTO{}, err
	}
	var out supply.CampaignChainDTO
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
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
		host.AuditSupplyChainUpdate(ctx, q, campaignID, oldNodesJSON, nodesJSON)
		parsed, err := parseSupplyChainNodes(updated.SupplyChainNodes)
		if err != nil {
			return err
		}
		out = supply.CampaignChainDTO{CampaignID: campaignID.String(), Nodes: parsed}
		return nil
	})
	return out, err
}

func parseSupplyChainNodes(raw []byte) ([]supply.ChainNode, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []supply.ChainNode{}, nil
	}
	var nodes []supply.ChainNode
	if err := coldpath.UnmarshalJSON(raw, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (r *Runtime) CreateCampaignTemplate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	budgetLimit int64,
	pacing db.PacingModeType,
	dailyBudget int64,
	timezone string,
	freqLimit, freqWindow int32,
	targetCountries []string,
	brandID *uuid.UUID,
	daypartHours []int16,
) (uuid.UUID, error) {
	if r == nil {
		return uuid.Nil, campaign.ErrServiceUnavailable()
	}
	return createCampaignTemplate(ctx, r.PoolOrNil(), customerID, name, budgetLimit, pacing, dailyBudget, timezone, freqLimit, freqWindow, targetCountries, brandID, daypartHours)
}

func (r *Runtime) ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]campaign.CampaignTemplateDTO, int64, error) {
	return listCampaignTemplates(ctx, r.PoolOrNil(), customerID, limit, offset)
}

func (r *Runtime) CreateCampaignFromTemplate(ctx context.Context, templateID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error) {
	if r == nil || r.effects == nil {
		return uuid.Nil, campaign.ErrServiceUnavailable()
	}
	return createCampaignFromTemplate(ctx, r.PoolOrNil(), r, templateID, customerID, name, budgetLimit, idempotencyKey)
}

func (r *Runtime) SaveCampaignAsTemplate(ctx context.Context, campaignID uuid.UUID, templateName string) (uuid.UUID, error) {
	if r == nil || r.effects == nil {
		return uuid.Nil, campaign.ErrServiceUnavailable()
	}
	return saveCampaignAsTemplate(ctx, r.PoolOrNil(), r.effects, r, campaignID, templateName)
}

func createCampaignTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	customerID uuid.UUID,
	name string,
	budgetLimit int64,
	pacing db.PacingModeType,
	dailyBudget int64,
	timezone string,
	freqLimit, freqWindow int32,
	targetCountries []string,
	brandID *uuid.UUID,
	daypartHours []int16,
) (uuid.UUID, error) {
	if err := campaign.ValidateDaypartHours(daypartHours); err != nil {
		return uuid.Nil, err
	}
	if pool == nil {
		return uuid.Nil, campaign.ErrServiceUnavailable()
	}
	templateID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	var brandParam pgtype.UUID
	if brandID != nil {
		brandParam = domain.ToUUID(*brandID)
	}

	_, err = db.New(pool).CreateCampaignTemplate(ctx, db.CreateCampaignTemplateParams{
		ID:              domain.ToUUID(templateID),
		CustomerID:      domain.ToUUID(customerID),
		Name:            name,
		BudgetLimit:     budgetLimit,
		PacingMode:      pacing,
		DailyBudget:     dailyBudget,
		Timezone:        timezone,
		FreqLimit:       freqLimit,
		FreqWindow:      freqWindow,
		TargetCountries: campaign.CountriesOrEmpty(targetCountries),
		BrandID:         brandParam,
		DaypartHours:    campaign.DaypartOrEmpty(daypartHours),
	})
	return templateID, err
}

func listCampaignTemplates(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, limit, offset int32) ([]campaign.CampaignTemplateDTO, int64, error) {
	if pool == nil {
		return nil, 0, campaign.ErrServiceUnavailable()
	}
	q := db.New(pool)
	cid := domain.ToUUID(customerID)
	listParams := db.ListCampaignTemplatesParams{
		CustomerID: cid,
		Limit:      limit,
		Offset:     offset,
	}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountCampaignTemplates(ctx, cid) },
		func() ([]db.CampaignTemplate, error) { return q.ListCampaignTemplates(ctx, listParams) },
		campaignTemplateToDTO,
	)
}

func campaignTemplateToDTO(t db.CampaignTemplate) campaign.CampaignTemplateDTO {
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
	return campaign.CampaignTemplateDTO{
		ID:              uuid.UUID(t.ID.Bytes).String(),
		CustomerID:      uuid.UUID(t.CustomerID.Bytes).String(),
		Name:            t.Name,
		BudgetLimit:     formatCampaignMicro(t.BudgetLimit),
		PacingMode:      string(t.PacingMode),
		DailyBudget:     formatCampaignMicro(t.DailyBudget),
		Timezone:        t.Timezone,
		FreqLimit:       t.FreqLimit,
		FreqWindow:      t.FreqWindow,
		TargetCountries: countries,
		BrandID:         brandID,
		DaypartHours:    hours,
		CreatedAt:       t.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       t.UpdatedAt.Time.Format(time.RFC3339),
	}
}

type templateCampaignCreator interface {
	CreateCampaign(ctx context.Context, spec campaign.CreateCampaignSpec) (uuid.UUID, error)
	CreateCampaignTemplate(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		budgetLimit int64,
		pacing db.PacingModeType,
		dailyBudget int64,
		timezone string,
		freqLimit, freqWindow int32,
		targetCountries []string,
		brandID *uuid.UUID,
		daypartHours []int16,
	) (uuid.UUID, error)
}

func createCampaignFromTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	creator templateCampaignCreator,
	templateID, customerID uuid.UUID,
	name string,
	budgetLimit *int64,
	idempotencyKey string,
) (uuid.UUID, error) {
	if pool == nil || creator == nil {
		return uuid.Nil, campaign.ErrServiceUnavailable()
	}
	tmpl, err := db.New(pool).GetCampaignTemplate(ctx, domain.ToUUID(templateID))
	if err != nil {
		return uuid.Nil, campaign.MapCampaignNotFound(err, campaign.ErrTemplateNotFound)
	}
	if uuid.UUID(tmpl.CustomerID.Bytes) != customerID {
		return uuid.Nil, campaign.ErrTemplateBelongsToAnotherCustomer
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

	return creator.CreateCampaign(ctx, campaign.CreateCampaignSpec{
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

func saveCampaignAsTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	fx campaign.Effects,
	creator templateCampaignCreator,
	campaignID uuid.UUID,
	templateName string,
) (uuid.UUID, error) {
	if fx == nil || creator == nil {
		return uuid.Nil, campaign.ErrServiceUnavailable()
	}
	camp, err := fx.GetCampaignRow(ctx, campaignID)
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
	return creator.CreateCampaignTemplate(ctx,
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
