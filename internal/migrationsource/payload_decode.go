package migrationsource

import (
	"fmt"
	"strings"
)

func decodeJSONArrayOrCampaignsWrapper(payload []byte, decodeArray func([]byte) error, decodeWrapper func([]byte) error) error {
	payload = bytesTrimSpace(payload)
	if len(payload) == 0 {
		return fmt.Errorf("empty payload")
	}
	if payload[0] == '[' {
		return decodeArray(payload)
	}
	return decodeWrapper(payload)
}

func keitaroAdminTrackingURL(domain, alias, parameters, override string) (string, error) {
	if u := strings.TrimSpace(override); u != "" {
		return u, nil
	}
	base := strings.TrimRight(strings.TrimSpace(domain), "/")
	alias = strings.TrimSpace(alias)
	if base == "" || alias == "" {
		return "", fmt.Errorf("missing domain and alias (or tracking_url override)")
	}
	u := base + "/" + alias
	params := strings.TrimSpace(parameters)
	if params == "" {
		return u, nil
	}
	if strings.HasPrefix(params, "?") {
		return u + params, nil
	}
	return u + "?" + params, nil
}
