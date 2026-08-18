package controlplane

import (
	"context"

	"github.com/google/uuid"
)

type selfServeTemplatesAdapter struct {
	svc *Service
}

func (a selfServeTemplatesAdapter) ListCampaignTemplates(
	ctx context.Context,
	customerID uuid.UUID,
	limit, offset int32,
) ([]CampaignTemplateDTO, int64, error) {
	items, total, err := a.svc.ListCampaignTemplates(ctx, customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]CampaignTemplateDTO, len(items))
	for i, row := range items {
		out[i] = mapCampaignTemplateAdminDTO(row)
	}
	return out, total, nil
}

func (a selfServeTemplatesAdapter) CreateCampaignFromTemplate(
	ctx context.Context,
	templateID, customerID uuid.UUID,
	name string,
	budgetLimit *int64,
	idempotencyKey string,
) (uuid.UUID, error) {
	return a.svc.CreateCampaignFromTemplate(ctx, templateID, customerID, name, budgetLimit, idempotencyKey)
}

func mapCampaignTemplateAdminDTO(row CampaignTemplateDTO) CampaignTemplateDTO {
	return CampaignTemplateDTO{
		ID:              row.ID,
		CustomerID:      row.CustomerID,
		Name:            row.Name,
		BudgetLimit:     row.BudgetLimit,
		PacingMode:      row.PacingMode,
		DailyBudget:     row.DailyBudget,
		Timezone:        row.Timezone,
		FreqLimit:       row.FreqLimit,
		FreqWindow:      row.FreqWindow,
		TargetCountries: row.TargetCountries,
		BrandID:         row.BrandID,
		DaypartHours:    row.DaypartHours,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
