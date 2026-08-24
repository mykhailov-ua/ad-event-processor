package platformsync

import (
	"fmt"
	"strings"

	db "ad-event-processor/internal/domain/db"
)

func PreviewMutation(link db.PlatformCampaignLink, action string, req MutationRequest) (MutationPreview, error) {
	action = strings.TrimSpace(action)
	if !NetworkSupported(link.Network) {
		return MutationPreview{}, fmt.Errorf("unsupported network %q", link.Network)
	}

	preview := MutationPreview{
		DryRun:      true,
		Action:      action,
		Network:     link.Network,
		CampaignID:  uuidString(link.CampaignID),
		StatusFrom:  link.ExternalStatus,
		BudgetFrom:  link.ExternalDailyBudgetMicro.Int64,
		VendorWrite: true,
	}

	switch action {
	case ActionPause:
		if strings.EqualFold(link.ExternalStatus, "PAUSED") || strings.EqualFold(link.ExternalStatus, "CAMPAIGN_STATUS_PAUSED") {
			preview.Noop = true
			preview.VendorWrite = false
			return preview, nil
		}
		preview.StatusTo = pausedStatusForNetwork(link.Network)
	case ActionResume:
		if strings.EqualFold(link.ExternalStatus, "ACTIVE") || strings.EqualFold(link.ExternalStatus, "ENABLED") || strings.EqualFold(link.ExternalStatus, "CAMPAIGN_STATUS_ENABLED") {
			preview.Noop = true
			preview.VendorWrite = false
			return preview, nil
		}
		preview.StatusTo = activeStatusForNetwork(link.Network)
	case ActionSetDailyBudget:
		if req.DailyBudgetMicro <= 0 {
			return MutationPreview{}, fmt.Errorf("daily_budget_micro required")
		}
		if link.ExternalDailyBudgetMicro.Valid && link.ExternalDailyBudgetMicro.Int64 == req.DailyBudgetMicro {
			preview.Noop = true
			preview.VendorWrite = false
		}
		preview.BudgetTo = req.DailyBudgetMicro
	default:
		return MutationPreview{}, fmt.Errorf("unsupported action %q", action)
	}
	return preview, nil
}

func pausedStatusForNetwork(network string) string {
	if NormalizeNetwork(network) == NetworkGoogle {
		return "PAUSED"
	}
	return "PAUSED"
}

func activeStatusForNetwork(network string) string {
	if NormalizeNetwork(network) == NetworkGoogle {
		return "ENABLED"
	}
	return "ACTIVE"
}
