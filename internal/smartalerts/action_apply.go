package smartalerts

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type marginActivityTarget struct {
	campaignID  uuid.UUID
	placementID string
}

func (w *Worker) applyRuleAction(
	ctx context.Context,
	rule smartAlertRuleRow,
	windowStart, windowEnd time.Time,
	eventID uuid.UUID,
) error {
	action := normalizeAlertAction(rule.Action)
	if action == ActionNotify {
		return nil
	}
	template, ok := parseTemplateFromMetric(rule.Metric)
	if !ok || template != TemplateMarginBreach {
		return fmt.Errorf("action %q requires margin_breach template", action)
	}
	targets, err := w.listMarginActivityTargets(ctx, rule, windowStart, windowEnd)
	if err != nil {
		return err
	}
	reason := fmt.Sprintf("smart_alert:%s:%s", rule.ID, eventID)
	switch action {
	case ActionPauseCampaign:
		return w.pauseCampaignsFromMarginActivity(ctx, targets, reason)
	case ActionBlacklistPlacement:
		return w.blacklistPlacementsFromMarginActivity(ctx, targets, reason)
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
}

func (w *Worker) listMarginActivityTargets(
	ctx context.Context,
	rule smartAlertRuleRow,
	windowStart, windowEnd time.Time,
) ([]marginActivityTarget, error) {
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	var campParam pgtype.UUID
	if rule.HasCampaign {
		campParam = domain.ToUUID(rule.CampaignID)
	}
	rows, err := w.host.Pool().Query(ctx, `
SELECT DISTINCT mga.campaign_id, mga.placement_id
FROM margin_guard_activity mga
JOIN campaigns c ON c.id = mga.campaign_id
WHERE c.customer_id = $1
 AND c.deleted_at IS NULL
 AND mga.action = 'pause'
 AND mga.created_at >= $2
 AND mga.created_at < $3
 AND ($4::uuid IS NULL OR c.id = $4)`,
		domain.ToUUID(rule.CustomerID), windowStart, windowEnd, campParam,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []marginActivityTarget
	for rows.Next() {
		var target marginActivityTarget
		if err := rows.Scan(&target.campaignID, &target.placementID); err != nil {
			return nil, err
		}
		out = append(out, target)
	}
	return out, rows.Err()
}

func (w *Worker) pauseCampaignsFromMarginActivity(ctx context.Context, targets []marginActivityTarget, reason string) error {
	if w == nil || w.host == nil {
		return fmt.Errorf("service unavailable")
	}
	seen := make(map[uuid.UUID]struct{}, len(targets))
	for _, target := range targets {
		if _, ok := seen[target.campaignID]; ok {
			continue
		}
		seen[target.campaignID] = struct{}{}
		if err := w.host.PauseCampaign(ctx, target.campaignID, reason); err != nil {
			return fmt.Errorf("pause campaign %s: %w", target.campaignID, err)
		}
	}
	return nil
}

func (w *Worker) blacklistPlacementsFromMarginActivity(ctx context.Context, targets []marginActivityTarget, reason string) error {
	if w == nil || w.host == nil {
		return fmt.Errorf("service unavailable")
	}
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		placementID := stringsTrim(target.placementID)
		if placementID == "" {
			continue
		}
		key := target.campaignID.String() + ":" + placementID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := w.host.BlacklistPlacement(ctx, target.campaignID, placementID); err != nil {
			return fmt.Errorf("blacklist placement %s for campaign %s: %w", placementID, target.campaignID, err)
		}
	}
	return nil
}
