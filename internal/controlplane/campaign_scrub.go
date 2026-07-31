package controlplane

import (
	"context"

	"espx/internal/controlplane/authz"
)

type CampaignResponse = CampaignDTO

func (c CampaignDTO) Scrub(level authz.MaskLevel) CampaignDTO {
	if level == authz.MaskFull {
		return c
	}
	out := c
	out.TargetURL = ""
	out.CreativePayload = nil
	out.ReferrerFilter = ""
	return out
}

func (c BrandCreativeDTO) Scrub(level authz.MaskLevel) BrandCreativeDTO {
	if level == authz.MaskFull {
		return c
	}
	out := c
	out.LandingURL = ""
	return out
}

func scrubCampaignDTO(ctx context.Context, dto CampaignDTO) CampaignDTO {
	if snap, ok := authz.SnapshotFromContext(ctx); ok {
		return dto.Scrub(snap.Mask)
	}
	return dto
}
