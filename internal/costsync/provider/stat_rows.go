package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"ad-event-processor/pkg/money"
)

type networkStatRow struct {
	campaignID string
	spendMicro int64
}

func parseNetworkStatRows(body []byte, campaignKeys []string) ([]networkStatRow, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return nil, err
	}

	for _, key := range []string{"items", "result", "data", "rows", "statistics"} {
		raw, ok := top[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var items []map[string]any
		if err := json.Unmarshal(raw, &items); err != nil {
			continue
		}
		return mapNetworkStatItems(items, campaignKeys)
	}

	var items []map[string]any
	if err := json.Unmarshal(body, &items); err == nil && len(items) > 0 {
		return mapNetworkStatItems(items, campaignKeys)
	}
	return nil, nil
}

func mapNetworkStatItems(items []map[string]any, campaignKeys []string) ([]networkStatRow, error) {
	out := make([]networkStatRow, 0, len(items))
	for _, item := range items {
		if row, ok := networkStatRowFromMap(item, campaignKeys); ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func networkStatRowFromMap(item map[string]any, campaignKeys []string) (networkStatRow, bool) {
	campKey := networkCampaignKey(item, campaignKeys)
	spendMicro, err := networkSpendMicro(item)
	if err != nil || spendMicro == 0 || campKey == "" {
		return networkStatRow{}, false
	}
	return networkStatRow{campaignID: campKey, spendMicro: spendMicro}, true
}

func networkCampaignKey(item map[string]any, keys []string) string {
	for _, key := range keys {
		if v, ok := item[key]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case float64:
				return fmt.Sprintf("%.0f", t)
			}
		}
	}
	return ""
}

func networkSpendMicro(item map[string]any) (int64, error) {
	for _, key := range []string{"spent", "spend", "costs", "cost", "money", "revenue", "amount"} {
		v, ok := item[key]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			return money.ParseDecimal(strings.TrimSpace(t))
		case float64:
			return money.JSONAmountToMicro(t)
		}
	}
	return 0, nil
}
