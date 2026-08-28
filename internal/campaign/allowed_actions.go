package campaign

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
)

func computeCampaignAllowedActions(ctx context.Context, status string) ([]string, map[string]string) {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return nil, nil
	}
	actions := make([]string, 0, 8)
	denied := make(map[string]string)

	add := func(action string) {
		actions = append(actions, action)
	}

	if snap.Has(authz.PermCampaignsWrite) || snap.Has(authz.PermCampaignsWriteMask) {
		if status != "ARCHIVED" {
			add("edit_general")
		}
	}
	if snap.Has(authz.PermCampaignsPause) {
		if status == "ACTIVE" {
			add("pause")
		}
		if status == "PAUSED" {
			add("resume")
		}
	}
	if snap.Has(authz.PermCampaignsWrite) {
		add("clone")
		add("edit_fraud")
		add("edit_budget")
		add("export")
	} else if snap.Has(authz.PermCampaignsWriteMask) {
		denied["edit_fraud"] = "requires_campaigns_write"
		denied["edit_budget"] = "requires_campaigns_write"
		denied["clone"] = "requires_campaigns_write"
	} else {
		denied["edit_general"] = "requires_campaigns_write"
		denied["edit_fraud"] = "requires_campaigns_write"
		denied["edit_budget"] = "requires_campaigns_write"
		denied["clone"] = "requires_campaigns_write"
		denied["export"] = "requires_campaigns_read"
	}
	if !snap.Has(authz.PermCampaignsRead) && !snap.Has(authz.PermCampaignsReadMasked) {
		return nil, denied
	}
	if snap.Has(authz.PermCampaignsRead) || snap.Has(authz.PermCampaignsReadMasked) {
		if !containsString(actions, "export") && snap.Has(authz.PermCampaignsRead) {
			add("export")
		}
	}
	return actions, denied
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func AttachCampaignPresentation(ctx context.Context, dto *CampaignDTO) {
	attachCampaignPresentation(ctx, dto)
}

func attachCampaignPresentation(ctx context.Context, dto *CampaignDTO) {
	if dto == nil {
		return
	}
	dto.StatusLabel = campaignStatusLabel(dto.Status)
	dto.StatusTone = campaignStatusTone(dto.Status)
	attachCampaignMoneyDisplay(dto)
	actions, denied := computeCampaignAllowedActions(ctx, dto.Status)
	dto.AllowedActions = actions
	if len(denied) > 0 {
		dto.DeniedReasons = denied
	}
}
