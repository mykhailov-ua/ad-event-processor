package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type IntegrationHealthStatus = campaign.IntegrationHealthStatus

const (
	IntegrationHealthOK   = campaign.IntegrationHealthOK
	IntegrationHealthWarn = campaign.IntegrationHealthWarn
	IntegrationHealthFail = campaign.IntegrationHealthFail
)

func (s *Service) GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (IntegrationHealthDTO, error) {
	if s == nil || s.pool == nil {
		return IntegrationHealthDTO{}, fmt.Errorf("service unavailable")
	}
	row, err := s.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return IntegrationHealthDTO{}, err
	}
	if err := assertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return IntegrationHealthDTO{}, err
	}

	input := campaign.IntegrationHealthInput{
		CampaignID:             campaignID,
		IntegrationSchemaBound: row.IntegrationSchemaID.Valid,
		TrafficTemplateID:      formatOptionalText(row.TrafficTemplateID),
		TargetURL:              strings.TrimSpace(row.TargetUrl),
		ClickQueryParams:       clickQueryParamsFromRaw(row.ClickQueryParams),
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
		input.CostSyncNetwork = campaign.CostSyncNetworkForTrafficTemplate(input.TrafficTemplateID)
	}
	if input.CostSyncNetwork != "" {
		q := db.New(s.pool)
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

	q := db.New(s.pool)
	if pb, err := q.GetPostbackConfig(ctx, domain.ToUUID(campaignID)); err == nil {
		if strings.TrimSpace(pb.UrlTemplate) != "" {
			input.PostbackConfigured = true
		}
	} else if !isPgNoRows(err) {
		return IntegrationHealthDTO{}, err
	}

	return campaign.BuildCampaignIntegrationHealth(input), nil
}

func isPgNoRows(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "no rows"))
}
