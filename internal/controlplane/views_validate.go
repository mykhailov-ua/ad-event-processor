package controlplane

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxSavedViewSpecBytes = 8 * 1024

var allowedSavedViewSpecKeys = map[string]struct{}{
	"from":             {},
	"to":               {},
	"compare":          {},
	"campaign_id":      {},
	"limit":            {},
	"columns":          {},
	"from_offset_days": {},
	"to_offset_days":   {},
}

func allowedSavedViewReportKeys() map[string]struct{} {
	keys := make(map[string]struct{}, len(liveReportExportKeys())+4)
	for _, key := range liveReportExportKeys() {
		keys[key] = struct{}{}
	}
	return keys
}

func validateSavedViewInput(name, reportKey string, spec json.RawMessage) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errValidation("name is required")
	}
	if len(name) > 128 {
		return errValidation("name must be at most 128 characters")
	}
	reportKey = strings.TrimSpace(reportKey)
	if reportKey == "" {
		return errValidation("report_key is required")
	}
	if _, ok := allowedSavedViewReportKeys()[reportKey]; !ok {
		return errValidation(fmt.Sprintf("unsupported report_key %q", reportKey))
	}
	return validateSavedViewSpec(spec)
}

func validateSavedViewSpec(spec json.RawMessage) error {
	if len(spec) == 0 {
		return nil
	}
	if len(spec) > maxSavedViewSpecBytes {
		return errValidation(fmt.Sprintf("spec exceeds %d bytes", maxSavedViewSpecBytes))
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(spec, &raw); err != nil {
		return errValidation("spec must be a JSON object")
	}
	for key, value := range raw {
		if _, ok := allowedSavedViewSpecKeys[key]; !ok {
			return errValidation(fmt.Sprintf("unsupported spec key %q", key))
		}
		if err := validateSavedViewSpecValue(key, value); err != nil {
			return err
		}
	}
	if fromRaw, ok := raw["from"]; ok {
		if toRaw, hasTo := raw["to"]; hasTo {
			var fromStr, toStr string
			if err := json.Unmarshal(fromRaw, &fromStr); err != nil {
				return errValidation("invalid spec.from")
			}
			if err := json.Unmarshal(toRaw, &toStr); err != nil {
				return errValidation("invalid spec.to")
			}
			if _, _, err := parseReportRangeFromStrings(fromStr, toStr); err != nil {
				return errValidation(err.Error())
			}
		}
	}
	return nil
}

func validateSavedViewSpecValue(key string, value json.RawMessage) error {
	switch key {
	case "from", "to":
		var ts string
		if err := json.Unmarshal(value, &ts); err != nil {
			return errValidation(fmt.Sprintf("invalid spec.%s", key))
		}
		ts = strings.TrimSpace(ts)
		if ts == "" {
			return errValidation(fmt.Sprintf("spec.%s is required", key))
		}
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			return errValidation(fmt.Sprintf("invalid spec.%s timestamp", key))
		}
		return nil
	case "compare":
		var asBool bool
		if err := json.Unmarshal(value, &asBool); err == nil {
			return nil
		}
		var asString string
		if err := json.Unmarshal(value, &asString); err != nil {
			return errValidation("invalid spec.compare")
		}
		if asString != "" && asString != "previous" {
			return errValidation("spec.compare must be previous when set as string")
		}
		return nil
	case "campaign_id":
		var id string
		if err := json.Unmarshal(value, &id); err != nil {
			return errValidation("invalid spec.campaign_id")
		}
		if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
			return errValidation("invalid spec.campaign_id")
		}
		return nil
	case "limit", "from_offset_days", "to_offset_days":
		var n int
		if err := json.Unmarshal(value, &n); err != nil {
			return errValidation(fmt.Sprintf("invalid spec.%s", key))
		}
		if n < 0 {
			return errValidation(fmt.Sprintf("spec.%s must be non-negative", key))
		}
		return nil
	case "columns":
		var cols []string
		if err := json.Unmarshal(value, &cols); err != nil {
			return errValidation("invalid spec.columns")
		}
		if len(cols) > 64 {
			return errValidation("spec.columns exceeds 64 entries")
		}
		for i := range cols {
			col := strings.TrimSpace(cols[i])
			if col == "" || len(col) > 64 {
				return errValidation("invalid spec.columns entry")
			}
		}
		return nil
	default:
		return nil
	}
}
