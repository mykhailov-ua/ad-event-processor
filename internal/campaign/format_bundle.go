package campaign

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"

	"github.com/jackc/pgx/v5/pgtype"
)

func attachCampaignMoneyDisplay(dto *CampaignDTO) {
	AttachCampaignMoneyDisplay(dto)
}

func attachCampaignTimestampDisplay(dto *CampaignDTO) {
	if dto == nil {
		return
	}
	if dto.CreatedAtDisplay == "" && dto.CreatedAt != "" {
		dto.CreatedAtDisplay = coldpath.RFC3339Display(dto.CreatedAt)
	}
	if dto.UpdatedAtDisplay == "" && dto.UpdatedAt != "" {
		dto.UpdatedAtDisplay = coldpath.RFC3339Display(dto.UpdatedAt)
	}
}

func AttachCampaignTimestampDisplay(raw string) string {
	return coldpath.RFC3339Display(raw)
}

func AttachCampaignMoneyDisplay(dto *CampaignDTO) {
	if dto == nil {
		return
	}
	if dto.BudgetLimitDisplay == "" && dto.BudgetLimit != "" {
		dto.BudgetLimitDisplay = moneyDisplayFromDecimal(dto.BudgetLimit)
	}
	if dto.DailyBudgetDisplay == "" && dto.DailyBudget != "" {
		dto.DailyBudgetDisplay = moneyDisplayFromDecimal(dto.DailyBudget)
	}
	if dto.CurrentSpendDisplay == "" && dto.CurrentSpend != "" {
		dto.CurrentSpendDisplay = moneyDisplayFromDecimal(dto.CurrentSpend)
	}
}

// Clamps to [0, 100]. Omitted when budget_limit is zero or unparsable.
func AttachCampaignBudgetUsedPct(dto *CampaignDTO) {
	if dto == nil {
		return
	}
	limitMicro, errLimit := money.ParseDecimal(dto.BudgetLimit)
	if errLimit != nil || limitMicro <= 0 {
		return
	}
	spendMicro, errSpend := money.ParseDecimal(dto.CurrentSpend)
	if errSpend != nil {
		return
	}
	pct := float64(spendMicro) / float64(limitMicro) * 100
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		return
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	dto.BudgetUsedPct = &pct
}

func moneyDisplayFromDecimal(amount string) string {
	if amount == "" {
		return ""
	}
	return "USD " + amount
}

func FormatRateDisplay(rate float64) string {
	return formatRateDisplay(rate)
}

func formatRateDisplay(rate float64) string {
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", rate*100)
}

func FormatBasisPointsDisplay(bps int) string {
	return formatBasisPointsDisplay(bps)
}

func formatBasisPointsDisplay(bps int) string {
	return formatRateDisplay(float64(bps) / 10000)
}

func campaignStatusLabel(status string) string {
	return CampaignStatusLabel(status)
}

func CampaignStatusLabel(status string) string {
	switch status {
	case "ACTIVE":
		return "Active"
	case "PAUSED":
		return "Paused"
	case "ARCHIVED":
		return "Archived"
	default:
		if status == "" {
			return "Unknown"
		}
		return status
	}
}

func campaignStatusTone(status string) string {
	return CampaignStatusTone(status)
}

func CampaignStatusTone(status string) string {
	switch status {
	case "ACTIVE":
		return "success"
	case "PAUSED":
		return "warning"
	case "ARCHIVED":
		return "muted"
	default:
		return "muted"
	}
}

func FormatOptionalText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return strings.TrimSpace(t.String)
}

func ClickQueryParamsFromRaw(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil || len(out) == 0 {
		return nil
	}
	return normalizeClickQueryParams(out)
}
