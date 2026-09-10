package licenseissue

import (
	"fmt"
	"strings"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensing/entitlements"
)

type CatalogSKU struct {
	Code            string  `json:"code"`
	DisplayName     string  `json:"display_name"`
	PriceUSDMonthly float64 `json:"price_usd_monthly"`
	ValidDays       int     `json:"valid_days"`
	ButtonLabel     string  `json:"button_label"`
	CallbackData    string  `json:"callback_data"`
	UserRequestable bool    `json:"user_requestable"`
	RequiresHWID    bool    `json:"requires_hwid"`
}

type PlanButtonsResponse struct {
	Rows [][]TelegramButton `json:"rows"`
}

type TelegramButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

func catalogFromSKU(sku entitlements.SKUDefinition) CatalogSKU {
	label := buttonLabel(sku)
	return CatalogSKU{
		Code:            sku.Code,
		DisplayName:     sku.DisplayName,
		PriceUSDMonthly: sku.PriceUSDMonthly,
		ValidDays:       sku.ValidDays,
		ButtonLabel:     label,
		CallbackData:    "plan:" + sku.Code,
		UserRequestable: strings.EqualFold(sku.Code, licensing.SKUCodePilot),
		RequiresHWID:    true,
	}
}

func buttonLabel(sku entitlements.SKUDefinition) string {
	name := strings.TrimSpace(sku.DisplayName)
	if name == "" {
		name = sku.Code
	}
	if strings.EqualFold(sku.Code, licensing.SKUCodePilot) || sku.PriceUSDMonthly <= 0 {
		return fmt.Sprintf("%s (free %dd)", name, sku.ValidDays)
	}
	return fmt.Sprintf("%s $%.0f/mo", name, sku.PriceUSDMonthly)
}

func (s *Service) PlanButtons() (PlanButtonsResponse, error) {
	items, err := s.Catalog()
	if err != nil {
		return PlanButtonsResponse{}, err
	}
	rows := make([][]TelegramButton, 0, len(items))
	for _, item := range items {
		if !item.UserRequestable {
			continue
		}
		rows = append(rows, []TelegramButton{{
			Text:         item.ButtonLabel,
			CallbackData: item.CallbackData,
		}})
	}
	for _, item := range items {
		if item.UserRequestable {
			continue
		}
		rows = append(rows, []TelegramButton{{
			Text:         item.ButtonLabel,
			CallbackData: item.CallbackData,
		}})
	}
	return PlanButtonsResponse{Rows: rows}, nil
}
