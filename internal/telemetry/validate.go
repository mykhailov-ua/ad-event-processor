package telemetry

import (
	"encoding/json"
	"fmt"
	"strings"
)

var forbiddenPayloadKeys = []string{
	"campaign_id",
	"customer_id",
	"domain",
	"url",
	"referrer",
	"click_id",
	"user_id",
	"ip",
	"ip_address",
	"hostname",
	"email",
	"creative_id",
	"payout",
	"placement",
}

func ValidatePayloadJSON(raw []byte) error {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return fmt.Errorf("telemetry: invalid json: %w", err)
	}
	return validateValue("", root)
}

func validateValue(path string, v any) error {
	switch node := v.(type) {
	case map[string]any:
		for key, child := range node {
			if err := checkForbiddenKey(path, key); err != nil {
				return err
			}
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if err := validateValue(childPath, child); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range node {
			if err := validateValue(fmt.Sprintf("%s[%d]", path, i), child); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkForbiddenKey(path, key string) error {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, bad := range forbiddenPayloadKeys {
		if lower == bad || strings.Contains(lower, bad) {
			loc := key
			if path != "" {
				loc = path + "." + key
			}
			return fmt.Errorf("telemetry: forbidden field %q", loc)
		}
	}
	return nil
}
