package campaign

import (
	"context"
	"strings"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

func CollectPublishBlocked(ctx context.Context, fx Effects, campaignID uuid.UUID, row db.Campaign) (*CampaignPublishBlockedError, error) {
	input, err := BuildPublishGateInput(ctx, fx, campaignID, row)
	if err != nil {
		return nil, err
	}
	return EvaluatePublishBlocked(ctx, input), nil
}

func BuildPublishGateInput(ctx context.Context, fx Effects, campaignID uuid.UUID, row db.Campaign) (PublishGateEvalInput, error) {
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
			flowDTO, flowErr := fx.GetFlow(ctx, parsedFlowID)
			if flowErr != nil {
				input.FlowPathError = "flow not found"
			} else {
				paths, pathsErr := ParseFlowPaths(flowDTO.Paths)
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
