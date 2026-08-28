package campaign

import (
	"context"
	"errors"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetCampaignIntegrationHealth(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID) (IntegrationHealthDTO, error) {
	if pool == nil || fx == nil {
		return IntegrationHealthDTO{}, errServiceUnavailable()
	}
	row, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return IntegrationHealthDTO{}, err
	}
	if err := assertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return IntegrationHealthDTO{}, err
	}

	input := IntegrationHealthInput{
		CampaignID:             campaignID,
		IntegrationSchemaBound: row.IntegrationSchemaID.Valid,
		TrafficTemplateID:      FormatOptionalText(row.TrafficTemplateID),
		TargetURL:              strings.TrimSpace(row.TargetUrl),
		ClickQueryParams:       ClickQueryParamsFromRaw(row.ClickQueryParams),
	}
	if len(row.IngressCostConfig) > 0 {
		parsed := domain.ParseIngressCostConfigJSON(row.IngressCostConfig)
		if parsed.Enabled() {
			input.IngressCostConfigured = true
			switch parsed.Param {
			case domain.IngressCostParamCost:
				input.IngressCostParam = "cost"
			case domain.IngressCostParamCPC:
				input.IngressCostParam = "cpc"
			case domain.IngressCostParamBid:
				input.IngressCostParam = "bid"
			}
		}
	}
	if input.TrafficTemplateID != "" {
		input.CostSyncNetwork = CostSyncNetworkForTrafficTemplate(input.TrafficTemplateID)
	}
	if input.CostSyncNetwork != "" {
		q := db.New(pool)
		cred, err := q.GetCostSyncCredential(ctx, db.GetCostSyncCredentialParams{
			CustomerID: row.CustomerID,
			Network:    input.CostSyncNetwork,
		})
		if err == nil && strings.TrimSpace(cred.Network) != "" {
			input.CostSyncCredentialPresent = true
		} else if err != nil && !isPgNoRows(err) {
			return IntegrationHealthDTO{}, err
		}
	}

	q := db.New(pool)
	if pb, err := q.GetPostbackConfig(ctx, domain.ToUUID(campaignID)); err == nil {
		if strings.TrimSpace(pb.UrlTemplate) != "" {
			input.PostbackConfigured = true
		}
	} else if !isPgNoRows(err) {
		return IntegrationHealthDTO{}, err
	}

	return BuildCampaignIntegrationHealth(input), nil
}

func AuditCampaignRevisionConflict(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID, expectedRevision string) {
	if pool == nil || fx == nil {
		return
	}
	row, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return
	}
	expectedRevision = strings.TrimSpace(expectedRevision)
	_ = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		var uid uuid.UUID
		if u, ok := authz.GetUser(ctx); ok {
			uid = u.UserID
		}
		fx.AuditLog(ctx, q, uid, "CAMPAIGN_REVISION_CONFLICT", "campaign", &campaignID, auditRevisionConflictChange{
			ExpectedRevision: expectedRevision,
			ServerRevision:   campaignRevision(row.UpdatedAt.Time.Format(time.RFC3339)),
		}, nil)
		return nil
	})
}

type auditRevisionConflictChange struct {
	ExpectedRevision string `json:"expected_revision"`
	ServerRevision   string `json:"server_revision"`
}

func isPgNoRows(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "no rows"))
}
