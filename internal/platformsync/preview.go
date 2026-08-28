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
		if isPausedExternalStatus(link.Network, link.ExternalStatus) {
			preview.Noop = true
			preview.VendorWrite = false
			return preview, nil
		}
		preview.StatusTo = pausedStatusForNetwork(link.Network)
	case ActionResume:
		if isActiveExternalStatus(link.Network, link.ExternalStatus) {
			preview.Noop = true
			preview.VendorWrite = false
			return preview, nil
		}
		preview.StatusTo = activeStatusForNetwork(link.Network)
	case ActionSetDailyBudget:
		if NormalizeNetwork(link.Network) == NetworkTikTok || NormalizeNetwork(link.Network) == NetworkMicrosoftAds {
			return MutationPreview{}, fmt.Errorf("daily budget mutation not supported for network %q", link.Network)
		}
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

func isPausedExternalStatus(network, status string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch NormalizeNetwork(network) {
	case NetworkGoogle:
		return normalized == "PAUSED" || normalized == "CAMPAIGN_STATUS_PAUSED"
	case NetworkMicrosoftAds:
		return normalized == "PAUSED" || strings.EqualFold(status, "Paused")
	default:
		return normalized == "PAUSED" || normalized == "DISABLE"
	}
}

func isActiveExternalStatus(network, status string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch NormalizeNetwork(network) {
	case NetworkGoogle:
		return normalized == "ENABLED" || normalized == "CAMPAIGN_STATUS_ENABLED"
	case NetworkMicrosoftAds:
		return normalized == "ACTIVE" || strings.EqualFold(status, "Active")
	default:
		return normalized == "ACTIVE" || normalized == "ENABLE"
	}
}

func pausedStatusForNetwork(network string) string {
	switch NormalizeNetwork(network) {
	case NetworkGoogle:
		return "PAUSED"
	case NetworkTikTok:
		return "DISABLE"
	case NetworkMicrosoftAds:
		return "Paused"
	default:
		return "PAUSED"
	}
}

func activeStatusForNetwork(network string) string {
	switch NormalizeNetwork(network) {
	case NetworkGoogle:
		return "ENABLED"
	case NetworkTikTok:
		return "ENABLE"
	case NetworkMicrosoftAds:
		return "Active"
	default:
		return "ACTIVE"
	}
}
