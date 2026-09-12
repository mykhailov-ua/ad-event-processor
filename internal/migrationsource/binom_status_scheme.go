package migrationsource

import (
	"fmt"
	"strings"
)

type binomStatusScheme struct {
	Rules []binomStatusSchemeRule `json:"rules"`
}

type binomStatusSchemeRule struct {
	IfStatus        string `json:"if_status"`
	IfGoal          string `json:"if_goal"`
	ThenStatus      string `json:"then_status"`
	ThenGoal        string `json:"then_goal"`
	PayoutMode      string `json:"payout_mode"`
	PayoutMicro     int64  `json:"payout_micro"`
	DisablePostback bool   `json:"disable_postback"`
	Enabled         *bool  `json:"enabled"`
}

type MappedStatusSchemeRule struct {
	SortOrder         int32  `json:"sort_order"`
	WhenStatus        string `json:"when_status"`
	WhenGoal          string `json:"when_goal"`
	SetInternalStatus string `json:"set_internal_status"`
	SetGoalName       string `json:"set_goal_name"`
	PayoutMode        string `json:"payout_mode"`
	PayoutMicro       int64  `json:"payout_micro"`
	FireOutbound      bool   `json:"fire_outbound"`
	Enabled           bool   `json:"enabled"`
}

func mapBinomStatusScheme(scheme *binomStatusScheme, campaignRef string) ([]MappedStatusSchemeRule, []Warning) {
	if scheme == nil || len(scheme.Rules) == 0 {
		return nil, nil
	}
	out := make([]MappedStatusSchemeRule, 0, len(scheme.Rules))
	warnings := make([]Warning, 0)
	unmapped := 0
	for i, row := range scheme.Rules {
		mapped, warn, ok := mapBinomStatusSchemeRule(row, int32(i+1))
		if warn != nil {
			warn.CampaignRef = campaignRef
			warnings = append(warnings, *warn)
		}
		if !ok {
			unmapped++
			continue
		}
		out = append(out, mapped)
	}
	if unmapped > 0 {
		warnings = append(warnings, Warning{
			Slug:        "binom_status_scheme_unmapped",
			Message:     fmt.Sprintf("%d status scheme rules could not be mapped", unmapped),
			CampaignRef: campaignRef,
		})
	}
	return out, warnings
}

func mapBinomStatusSchemeRule(row binomStatusSchemeRule, sortOrder int32) (MappedStatusSchemeRule, *Warning, bool) {
	whenStatus := strings.ToLower(strings.TrimSpace(row.IfStatus))
	whenGoal := strings.ToLower(strings.TrimSpace(row.IfGoal))
	if whenStatus == "" && whenGoal == "" {
		return MappedStatusSchemeRule{}, &Warning{
			Slug:    "binom_status_scheme_unmapped",
			Message: "rule missing if_status and if_goal",
		}, false
	}
	payoutMode, payoutWarn := mapBinomPayoutMode(row.PayoutMode)
	if payoutWarn != nil {
		return MappedStatusSchemeRule{}, payoutWarn, false
	}
	enabled := true
	if row.Enabled != nil {
		enabled = *row.Enabled
	}
	return MappedStatusSchemeRule{
		SortOrder:         sortOrder,
		WhenStatus:        whenStatus,
		WhenGoal:          whenGoal,
		SetInternalStatus: strings.TrimSpace(row.ThenStatus),
		SetGoalName:       strings.TrimSpace(row.ThenGoal),
		PayoutMode:        payoutMode,
		PayoutMicro:       row.PayoutMicro,
		FireOutbound:      !row.DisablePostback,
		Enabled:           enabled,
	}, nil, true
}

func mapBinomPayoutMode(raw string) (string, *Warning) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "", "inherit":
		return "inherit", nil
	case "fixed":
		return "fixed", nil
	case "zero", "0":
		return "zero", nil
	case "pass_through", "pass-through", "passthrough":
		return "pass_through", nil
	case "accumulate", "accumulate_payout":
		return "accumulate_payout", nil
	default:
		return "", &Warning{
			Slug:    "binom_status_scheme_unmapped",
			Message: "unsupported payout_mode " + raw,
		}
	}
}
