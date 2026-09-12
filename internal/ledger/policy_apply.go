package ledger

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

func (w *Worker) applyCampaignBreach(ctx context.Context, policy *Policy, campaignID uuid.UUID, reason string) error {
	if w == nil || w.enforcement == nil {
		return fmt.Errorf("margin guard enforcement host not configured")
	}
	mode := CampaignBreachEnforcement(policy)
	switch mode {
	case EnforcementNotifyOnly:
		w.notifyMarginGuardCampaignPause(ctx, campaignID, reason)
		return nil
	case EnforcementPauseCampaign:
		if err := w.enforcement.PauseCampaign(ctx, campaignID, reason); err != nil {
			return err
		}
		if PolicyWantsPlatformPause(policy) {
			if err := w.enforcement.PlatformPauseCampaign(ctx, campaignID, policy.PlatformNetwork, reason); err != nil {
				slog.Warn("margin guard platform pause failed",
					"campaign_id", campaignID,
					"network", policy.PlatformNetwork,
					"error", err,
				)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported campaign enforcement %q", mode)
	}
}

func (w *Worker) applyPlacementBreach(ctx context.Context, policy *Policy, decision *Decision) error {
	if w == nil || w.enforcement == nil || decision == nil {
		return fmt.Errorf("margin guard enforcement host not configured")
	}
	mode := PlacementBreachEnforcement(policy)
	switch mode {
	case EnforcementNotifyOnly:
		return nil
	case EnforcementPauseCampaign:
		if err := w.enforcement.PauseCampaign(ctx, decision.CampaignID, decision.Reason); err != nil {
			return err
		}
		if PolicyWantsPlatformPause(policy) {
			if err := w.enforcement.PlatformPauseCampaign(ctx, decision.CampaignID, policy.PlatformNetwork, decision.Reason); err != nil {
				slog.Warn("margin guard platform pause failed",
					"campaign_id", decision.CampaignID,
					"network", policy.PlatformNetwork,
					"error", err,
				)
			}
		}
		return nil
	case EnforcementBlacklistPlacement:
		return w.enforcement.BlacklistPlacement(ctx, decision.CampaignID, decision.PlacementID)
	default:
		return fmt.Errorf("unsupported placement enforcement %q", mode)
	}
}

func (w *Worker) notifyMarginGuardCampaignPause(ctx context.Context, campaignID uuid.UUID, reason string) {
	if w == nil || w.notifier == nil {
		return
	}
	title := "Margin Guard: Campaign breach"
	body := fmt.Sprintf("Campaign: %s\nReason: %s", campaignID, reason)
	if _, err := w.notifier.SendNotification(ctx, "TELEGRAM", "admin", title, body); err != nil {
		slog.Error("failed to send margin guard campaign notification", "error", err)
	}
}
