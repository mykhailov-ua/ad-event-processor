package management

import (
	"context"

	"espx/internal/management/authz"
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

func scrubCampaignDTOs(ctx context.Context, dtos []CampaignDTO) []CampaignDTO {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok || snap.Mask == authz.MaskFull {
		return dtos
	}
	out := make([]CampaignDTO, len(dtos))
	for i, d := range dtos {
		out[i] = d.Scrub(snap.Mask)
	}
	return out
}
