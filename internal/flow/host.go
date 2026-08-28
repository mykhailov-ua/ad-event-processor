package flow

import (
	"context"
)

type Host interface {
	LanderPublicBase(ctx context.Context) string
	ValidateFlowPaths(ctx context.Context, paths []PathDTO) error
	PublishFlowReload(ctx context.Context) error
}

type CampaignFlowHost interface {
	PublishCampaignUpdate(ctx context.Context, campaignID string)
}
