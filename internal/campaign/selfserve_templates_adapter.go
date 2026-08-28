package campaign

import (
	"context"

	"github.com/google/uuid"
)

type SelfServeTemplateHost interface {
	ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]CampaignTemplateDTO, int64, error)
	CreateCampaignFromTemplate(ctx context.Context, templateID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error)
}

type selfServeTemplatesAdapter struct {
	host SelfServeTemplateHost
}

func NewSelfServeTemplatesAdapter(host SelfServeTemplateHost) SelfServeTemplates {
	if host == nil {
		return nil
	}
	return selfServeTemplatesAdapter{host: host}
}

func (a selfServeTemplatesAdapter) ListCampaignTemplates(
	ctx context.Context,
	customerID uuid.UUID,
	limit, offset int32,
) ([]CampaignTemplateDTO, int64, error) {
	return a.host.ListCampaignTemplates(ctx, customerID, limit, offset)
}

func (a selfServeTemplatesAdapter) CreateCampaignFromTemplate(
	ctx context.Context,
	templateID, customerID uuid.UUID,
	name string,
	budgetLimit *int64,
	idempotencyKey string,
) (uuid.UUID, error) {
	return a.host.CreateCampaignFromTemplate(ctx, templateID, customerID, name, budgetLimit, idempotencyKey)
}
