package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listCampaigns(
	ctx context.Context,
	pool *pgxpool.Pool,
	effects Effects,
	customerID uuid.UUID,
	status string,
	limit, offset int32,
) ([]CampaignDTO, int64, error) {
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
		OwnerUserID: campaignOwnerUserFilter(ctx),
	}
	listParams := db.ListCampaignsParams{
		Limit:       limit,
		Offset:      offset,
		CustomerID:  cid,
		Status:      st,
		OwnerUserID: campaignOwnerUserFilter(ctx),
	}

	items, total, err := coldpath.PaginatedList(
		func() (int64, error) { return q.CountCampaigns(ctx, countParams) },
		func() ([]db.Campaign, error) { return q.ListCampaigns(ctx, listParams) },
		func(c db.Campaign) CampaignDTO { return scrubCampaignDTO(ctx, c) },
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
	effects Effects,
	campaignID uuid.UUID,
) (CampaignDTO, error) {
	if pool == nil {
		return CampaignDTO{}, fmt.Errorf("service unavailable")
	}
	q := db.New(pool)
	c, err := q.GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return CampaignDTO{}, mapCampaignStoreError(err)
	}
	if err := assertMediaBuyerCampaignAccess(ctx, c); err != nil {
		return CampaignDTO{}, err
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
		return ErrCampaignNotFound
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

func ScrubCampaignFields(c CampaignDTO, level authz.MaskLevel) CampaignDTO {
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

func scrubCampaignDTO(ctx context.Context, c db.Campaign) CampaignDTO {
	countries := c.TargetCountries
	if countries == nil {
		countries = []string{}
	}
	dto := CampaignDTO{
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
		DaypartHours:               DaypartOrEmpty(c.DaypartHours),
		OwnerUserID:                formatCampaignOptionalUUID(c.OwnerUserID),
		TrafficTemplateID:          FormatOptionalText(c.TrafficTemplateID),
		ClickQueryParams:           ClickQueryParamsFromRaw(c.ClickQueryParams),
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
		out := ScrubCampaignFields(dto, snap.Mask)
		attachCampaignPresentation(ctx, &out)
		return out
	}
	attachCampaignPresentation(ctx, &dto)
	return dto
}
