package smartalerts

import (
	"fmt"
	"strings"
)

const (
	ActionNotify             = "notify"
	ActionPauseCampaign      = "pause_campaign"
	ActionBlacklistPlacement = "blacklist_placement"
)

func normalizeAlertAction(raw string) string {
	switch strings.TrimSpace(raw) {
	case ActionPauseCampaign, ActionBlacklistPlacement:
		return strings.TrimSpace(raw)
	default:
		return ActionNotify
	}
}

func validateAlertAction(template, action string) error {
	action = normalizeAlertAction(action)
	if action == ActionNotify {
		return nil
	}
	template = strings.TrimSpace(template)
	if template == "" {
		return fmt.Errorf("action %q requires template margin_breach", action)
	}
	normalized, err := normalizeAlertTemplate(template)
	if err != nil {
		return err
	}
	if normalized != TemplateMarginBreach {
		return fmt.Errorf("action %q only supported for margin_breach template", action)
	}
	return nil
}
