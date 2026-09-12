package campaign

import (
	"fmt"
	"strings"
)

// BulkPatchFields allowlisted fields for POST /api/v1/campaigns/bulk-patch.
// Flow, fraud, and revision-sensitive fields stay on single-campaign PATCH.
type BulkPatchFields struct {
	TargetURL          *string              `json:"target_url,omitempty"`
	BudgetLimitMicro   *int64               `json:"budget_limit_micro,omitempty"`
	BudgetLimit        *string              `json:"budget_limit,omitempty"`
	PacingMode         *string              `json:"pacing_mode,omitempty"`
	DailyBudgetMicro   *int64               `json:"daily_budget_micro,omitempty"`
	Timezone           *string              `json:"timezone,omitempty"`
	TargetCountries    []string             `json:"target_countries,omitempty"`
	Status             *string              `json:"status,omitempty"`
	ReferrerFilter     *string              `json:"referrer_filter,omitempty"`
	FallbackClickURL   *string              `json:"fallback_click_url,omitempty"`
	BudgetFailoverMode *string              `json:"budget_failover_mode,omitempty"`
	FlowPathWeights    []BulkFlowPathWeight `json:"flow_path_weights,omitempty"`
}

type BulkPatchCampaignRequest struct {
	CampaignIDs []string        `json:"campaign_ids"`
	Patch       BulkPatchFields `json:"patch"`
}

type BulkPatchCampaignResultRow struct {
	ID        string `json:"id"`
	OK        bool   `json:"ok"`
	ErrorCode string `json:"error_code,omitempty"`
}

type BulkPatchCampaignResponse struct {
	Results []BulkPatchCampaignResultRow `json:"results"`
}

func BulkPatchHasScalarFields(f BulkPatchFields) bool {
	scalar := f
	scalar.FlowPathWeights = nil
	return !BulkPatchFieldsEmpty(scalar)
}

func BulkPatchFieldsEmpty(f BulkPatchFields) bool {
	return f.TargetURL == nil &&
		f.BudgetLimitMicro == nil &&
		f.BudgetLimit == nil &&
		f.PacingMode == nil &&
		f.DailyBudgetMicro == nil &&
		f.Timezone == nil &&
		len(f.TargetCountries) == 0 &&
		f.Status == nil &&
		f.ReferrerFilter == nil &&
		f.FallbackClickURL == nil &&
		f.BudgetFailoverMode == nil &&
		len(f.FlowPathWeights) == 0
}

func ValidateBulkPatchFields(f BulkPatchFields) error {
	if BulkPatchFieldsEmpty(f) {
		return ErrValidationf("patch must include at least one field")
	}
	if f.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*f.Status))
		switch status {
		case "active", "paused", "draft", "archived":
		default:
			return ErrValidationf("invalid status")
		}
	}
	if f.BudgetFailoverMode != nil {
		mode := strings.ToLower(strings.TrimSpace(*f.BudgetFailoverMode))
		switch mode {
		case "reject", "fallback_url", "flow_next":
		default:
			return ErrValidationf("invalid budget_failover_mode")
		}
	}
	for i, row := range f.FlowPathWeights {
		if row.PathIndex < 0 {
			return ErrValidationf(fmt.Sprintf("flow_path_weights[%d]: path_index must be non-negative", i))
		}
		if row.Weight <= 0 {
			return ErrValidationf(fmt.Sprintf("flow_path_weights[%d]: weight must be positive", i))
		}
	}
	return nil
}

func BulkPatchFieldsToPatchRequest(f BulkPatchFields) PatchCampaignRequest {
	var countries []string
	if len(f.TargetCountries) > 0 {
		countries = append([]string(nil), f.TargetCountries...)
	}
	return PatchCampaignRequest{
		TargetURL:          f.TargetURL,
		BudgetLimitMicro:   f.BudgetLimitMicro,
		BudgetLimit:        f.BudgetLimit,
		PacingMode:         f.PacingMode,
		DailyBudgetMicro:   f.DailyBudgetMicro,
		Timezone:           f.Timezone,
		TargetCountries:    countries,
		Status:             f.Status,
		ReferrerFilter:     f.ReferrerFilter,
		FallbackClickURL:   f.FallbackClickURL,
		BudgetFailoverMode: f.BudgetFailoverMode,
	}
}
