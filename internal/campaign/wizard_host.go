package campaign

import (
	"context"

	"github.com/google/uuid"
)

type WizardHost interface {
	ApplyOnboardingTemplate(key string) (CampaignWizardStored, error)
	ImportBundledTemplate(ctx context.Context, schemaName string) error
	ImportCampaign(ctx context.Context, spec ImportCampaignSpec) (ImportCampaignResult, error)
	ApplyAffiliateNetworkTemplate(ctx context.Context, campaignID uuid.UUID, network, trackingDomain string) error
	PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (CampaignDTO, error)
	TrackingDomain(ctx context.Context, override string) string
	InboundTargetURL(ctx context.Context, schemaName, trackingDomain string) (string, error)
}
