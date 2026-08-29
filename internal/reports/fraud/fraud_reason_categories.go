package fraud

import "strings"

type fraudCategoryMeta struct {
	category string
	label    string
}

const (
	fraudCategoryInvalidDevice   = "invalid_device_signals"
	fraudCategoryAutomated       = "automated_traffic"
	fraudCategoryGeoLanguage     = "geo_language_mismatch"
	fraudCategoryProxyDatacenter = "proxy_datacenter"
	fraudCategoryPolicy          = "policy_reject"
	fraudCategoryIVTTier         = "ivt_tier"
	fraudCategoryOther           = "other"
)

var fraudReasonCategoryTable = map[string]fraudCategoryMeta{
	"tls_ja4_mismatch":           {fraudCategoryInvalidDevice, "Invalid device signals"},
	"client_hints_mismatch":      {fraudCategoryInvalidDevice, "Invalid device signals"},
	"header_order_mismatch":      {fraudCategoryInvalidDevice, "Invalid device signals"},
	"h2_settings_mismatch":       {fraudCategoryInvalidDevice, "Invalid device signals"},
	"h2_pseudo_order_mismatch":   {fraudCategoryInvalidDevice, "Invalid device signals"},
	"h2_downgrade_artifact":      {fraudCategoryInvalidDevice, "Invalid device signals"},
	"tls_alpn_mismatch":          {fraudCategoryInvalidDevice, "Invalid device signals"},
	"sec_fetch_anomaly":          {fraudCategoryInvalidDevice, "Invalid device signals"},
	"accept_encoding_mismatch":   {fraudCategoryInvalidDevice, "Invalid device signals"},
	"device_mismatch":            {fraudCategoryInvalidDevice, "Invalid device signals"},
	"os_fingerprint_mismatch":    {fraudCategoryInvalidDevice, "Invalid device signals"},
	"tls_fingerprint_blocklist":  {fraudCategoryInvalidDevice, "Invalid device signals"},
	"json_serialization_bot":     {fraudCategoryAutomated, "Automated traffic"},
	"behavior_telemetry_missing": {fraudCategoryAutomated, "Automated traffic"},
	"behavior_bezier_bot":        {fraudCategoryAutomated, "Automated traffic"},
	"accept_lang_geo_mismatch":   {fraudCategoryGeoLanguage, "Geo or language mismatch"},
	"residential_proxy":          {fraudCategoryProxyDatacenter, "Proxy or datacenter traffic"},
	"datacenter_ip":              {fraudCategoryProxyDatacenter, "Proxy or datacenter traffic"},
	"tcp_tunnel_mss":             {fraudCategoryProxyDatacenter, "Proxy or datacenter traffic"},
	"tcp_mss_anomaly":            {fraudCategoryProxyDatacenter, "Proxy or datacenter traffic"},
	"tcp_syn_os_mismatch":        {fraudCategoryProxyDatacenter, "Proxy or datacenter traffic"},
	"low_ttc":                    {fraudCategoryIVTTier, "Invalid traffic tier"},
	"missing_imp_ts":             {fraudCategoryIVTTier, "Invalid traffic tier"},
	"l3_blocklist":               {fraudCategoryIVTTier, "Invalid traffic tier"},
	"ipv4_rotation":              {fraudCategoryIVTTier, "Invalid traffic tier"},
	"attestation_missing":        {fraudCategoryPolicy, "Campaign policy"},
	"moderator_ip":               {fraudCategoryPolicy, "Campaign policy"},
}

var wireSignalReasonCodes = map[string]struct{}{
	"sec_fetch_anomaly":          {},
	"client_hints_mismatch":      {},
	"accept_encoding_mismatch":   {},
	"accept_lang_geo_mismatch":   {},
	"tls_ja4_mismatch":           {},
	"tls_alpn_mismatch":          {},
	"tcp_syn_os_mismatch":        {},
	"tcp_tunnel_mss":             {},
	"tcp_mss_anomaly":            {},
	"h2_settings_mismatch":       {},
	"h2_pseudo_order_mismatch":   {},
	"h2_downgrade_artifact":      {},
	"header_order_mismatch":      {},
	"json_serialization_bot":     {},
	"behavior_telemetry_missing": {},
	"behavior_bezier_bot":        {},
	"residential_proxy":          {},
}

func FraudReasonToCategory(reason string) (category string, label string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fraudCategoryOther, "Other"
	}
	if meta, ok := fraudReasonCategoryTable[reason]; ok {
		return meta.category, meta.label
	}
	return fraudCategoryOther, "Other"
}

func FraudReasonCategoriesFromField(reasonField string) []string {
	if reasonField == "" {
		return nil
	}
	parts := strings.Split(reasonField, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		category, _ := FraudReasonToCategory(part)
		if _, dup := seen[category]; dup {
			continue
		}
		seen[category] = struct{}{}
		out = append(out, category)
	}
	return out
}

func isWireSignalReason(reason string) bool {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return false
	}
	for _, part := range strings.Split(reason, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := wireSignalReasonCodes[part]; ok {
			return true
		}
	}
	return false
}

func FraudCategoryLabel(category string) string {
	switch category {
	case fraudCategoryInvalidDevice:
		return "Invalid device signals"
	case fraudCategoryAutomated:
		return "Automated traffic"
	case fraudCategoryGeoLanguage:
		return "Geo or language mismatch"
	case fraudCategoryProxyDatacenter:
		return "Proxy or datacenter traffic"
	case fraudCategoryPolicy:
		return "Campaign policy"
	case fraudCategoryIVTTier:
		return "Invalid traffic tier"
	default:
		return "Other"
	}
}
