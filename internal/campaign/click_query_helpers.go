package campaign

import (
	"fmt"
	"strings"
)

const (
	maxClickQueryParamKeys     = 40
	maxClickQueryParamValueLen = 512
	maxTrafficTemplateIDLen    = 64
)

var allowedClickQueryKeys = func() map[string]bool {
	keys := map[string]bool{
		"ad_campaign_id": true,
		"fbclid":         true,
		"gclid":          true,
		"ttclid":         true,
	}
	for i := 1; i <= 30; i++ {
		keys[fmt.Sprintf("sub%d", i)] = true
	}
	return keys
}()

func validateTrafficTemplateID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	if len(id) > maxTrafficTemplateIDLen {
		return fmt.Errorf("traffic_template_id too long")
	}
	return nil
}

func validateClickQueryParams(params map[string]string) error {
	if len(params) > maxClickQueryParamKeys {
		return fmt.Errorf("click_query_params: too many keys")
	}
	for key, value := range params {
		if !allowedClickQueryKeys[key] {
			return fmt.Errorf("click_query_params: invalid key %q", key)
		}
		if len(value) > maxClickQueryParamValueLen {
			return fmt.Errorf("click_query_params: value too long for %q", key)
		}
	}
	return nil
}

func normalizeClickQueryParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return nil
	}
	out := make(map[string]string, len(params))
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ValidateTrafficTemplateID(id string) error {
	return validateTrafficTemplateID(id)
}

func ValidateClickQueryParams(params map[string]string) error {
	return validateClickQueryParams(params)
}
