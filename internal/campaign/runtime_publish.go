package campaign

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func evaluateCampaignPublish(ctx context.Context, fx Effects, campaignID uuid.UUID) (CampaignPublishCheckDTO, error) {
	if fx == nil {
		return CampaignPublishCheckDTO{}, fmt.Errorf("service unavailable")
	}
	row, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return CampaignPublishCheckDTO{}, err
	}
	if err := assertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return CampaignPublishCheckDTO{}, err
	}
	blocked, err := collectCampaignPublishBlocked(ctx, fx, campaignID, row)
	if err != nil {
		return CampaignPublishCheckDTO{}, err
	}
	if blocked == nil {
		return CampaignPublishCheckDTO{Valid: true}, nil
	}
	return CampaignPublishCheckDTO{
		Valid:        false,
		FieldErrors:  blocked.FieldErrors,
		WarningSlugs: blocked.WarningSlugs,
	}, nil
}

func publishCampaign(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID, force bool) (CampaignDTO, error) {
	if pool == nil || fx == nil {
		return CampaignDTO{}, fmt.Errorf("service unavailable")
	}
	if force && !CanForceCampaignPublish(ctx) {
		return CampaignDTO{}, errValidation("force publish requires admin role")
	}
	row, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return CampaignDTO{}, err
	}
	if err := assertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return CampaignDTO{}, err
	}
	switch row.Status {
	case db.CampaignStatusTypeACTIVE:
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, row, force); err != nil {
			return CampaignDTO{}, err
		}
		return getCampaign(ctx, pool, fx, campaignID)
	case db.CampaignStatusTypePAUSED:
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, row, force); err != nil {
			return CampaignDTO{}, err
		}
		if err := fx.ResumeCampaignForPublish(ctx, campaignID, force); err != nil {
			return CampaignDTO{}, err
		}
		return getCampaign(ctx, pool, fx, campaignID)
	default:
		return CampaignDTO{}, errValidation("campaign cannot be published from current status")
	}
}

func collectCampaignPublishBlocked(ctx context.Context, fx Effects, campaignID uuid.UUID, row db.Campaign) (*CampaignPublishBlockedError, error) {
	input, err := buildPublishGateInput(ctx, fx, campaignID, row)
	if err != nil {
		return nil, err
	}
	return EvaluatePublishBlocked(ctx, input), nil
}

func BuildPublishGateInput(ctx context.Context, fx Effects, campaignID uuid.UUID, row db.Campaign) (PublishGateEvalInput, error) {
	return buildPublishGateInput(ctx, fx, campaignID, row)
}

func CollectPublishBlocked(ctx context.Context, fx Effects, campaignID uuid.UUID, row db.Campaign) (*CampaignPublishBlockedError, error) {
	return collectCampaignPublishBlocked(ctx, fx, campaignID, row)
}

func buildPublishGateInput(ctx context.Context, fx Effects, campaignID uuid.UUID, row db.Campaign) (PublishGateEvalInput, error) {
	input := PublishGateEvalInput{
		CampaignID:        campaignID,
		Name:              row.Name,
		BudgetLimit:       row.BudgetLimit,
		CurrentSpend:      row.CurrentSpend,
		TargetCountries:   append([]string(nil), row.TargetCountries...),
		TargetURL:         strings.TrimSpace(row.TargetUrl),
		ClickQueryParams:  ClickQueryParamsFromRaw(row.ClickQueryParams),
		ClickDelivery:     row.ClickDelivery,
		ProxyUpstreamURL:  row.ProxyUpstreamUrl,
		AllowHTTPInsecure: fx.ProxyAllowHTTPInsecure(),
	}

	flowID, err := fx.CampaignFlowID(ctx, campaignID)
	if err != nil || flowID == "" {
		input.FlowMissing = true
	} else {
		parsedFlowID, parseErr := uuid.Parse(flowID)
		if parseErr != nil {
			input.FlowPathError = "invalid flow id"
		} else {
			flow, flowErr := fx.GetFlow(ctx, parsedFlowID)
			if flowErr != nil {
				input.FlowPathError = "flow not found"
			} else {
				paths, pathsErr := ParseFlowPaths(flow.Paths)
				if pathsErr != nil {
					input.FlowPathError = "invalid stored flow paths"
				} else {
					if validateErr := fx.ValidateFlowPaths(ctx, paths); validateErr != nil {
						input.FlowPathError = validateErr.Error()
					}
					flowResp := BuildCampaignFlowValidateResponse(paths)
					if !flowResp.Valid {
						input.FlowPathError = FormatFlowPathErrors(flowResp.PathErrors)
					}
				}
			}
		}
	}

	health, healthErr := fx.GetCampaignIntegrationHealth(ctx, campaignID)
	if healthErr == nil {
		input.IntegrationHealth = health
		input.IntegrationHealthOK = true
	}

	return input, nil
}
