package costsync

import (
	"fmt"
	"strings"
)

type ExtraField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Placeholder string `json:"placeholder,omitempty"`
	Hint        string `json:"hint,omitempty"`
}

type NetworkCredentialSchema struct {
	Network        string       `json:"network"`
	Label          string       `json:"label"`
	AccountIDLabel string       `json:"account_id_label,omitempty"`
	ExtraFields    []ExtraField `json:"extra_fields"`
}

var networkCredentialSchemas = []NetworkCredentialSchema{
	{Network: "facebook", Label: "Facebook", ExtraFields: []ExtraField{
		{Key: "ad_account_id", Label: "Ad account ID", Hint: "act_ prefix optional; overrides Account ID when set"},
	}},
	{Network: "google", Label: "Google Ads", ExtraFields: []ExtraField{
		{Key: "customer_id", Label: "Google Ads customer ID", Hint: "Numeric customer id without dashes"},
		{Key: "developer_token", Label: "Developer token", Secret: true},
	}},
	{Network: "tiktok", Label: "TikTok Ads", ExtraFields: []ExtraField{
		{Key: "advertiser_id", Label: "Advertiser ID"},
	}},
	{Network: "microsoft_ads", Label: "Microsoft Ads", ExtraFields: []ExtraField{
		{Key: "customer_id", Label: "Customer ID", Required: true},
		{Key: "developer_token", Label: "Developer token", Required: true, Secret: true},
		{Key: "customer_account_id", Label: "Customer account ID", Hint: "Optional; overrides Account ID when set"},
	}},
	{Network: "snapchat", Label: "Snapchat Ads", ExtraFields: []ExtraField{
		{Key: "ad_account_id", Label: "Ad account ID"},
	}},
	{Network: "linkedin", Label: "LinkedIn Ads", ExtraFields: []ExtraField{
		{Key: "ad_account_id", Label: "Ad account ID"},
		{Key: "linkedin_version", Label: "LinkedIn API version", Placeholder: "202405"},
	}},
	{Network: "pinterest", Label: "Pinterest Ads", ExtraFields: []ExtraField{
		{Key: "ad_account_id", Label: "Ad account ID"},
	}},
	{Network: "trafficstars", Label: "TrafficStars"},
	{Network: "richads", Label: "RichAds", ExtraFields: []ExtraField{
		{Key: "segment", Label: "Report segment", Placeholder: "campaign_id", Hint: "Default campaign_id when empty"},
	}},
	{Network: "galaksion", Label: "Galaksion", ExtraFields: []ExtraField{
		{Key: "email", Label: "Login email", Hint: "Used when API token is not set"},
		{Key: "password", Label: "Login password", Secret: true},
		{Key: "group_by", Label: "Group by", Placeholder: "campaign"},
		{Key: "order_by", Label: "Order by"},
	}},
	{Network: "propellerads", Label: "PropellerAds"},
	{Network: "mgid", Label: "MGID", ExtraFields: []ExtraField{
		{Key: "client_id", Label: "Client ID"},
	}},
	{Network: "adsterra", Label: "Adsterra"},
	{Network: "exoclick", Label: "ExoClick", ExtraFields: []ExtraField{
		{Key: "auth_type", Label: "Auth type", Placeholder: "api_token"},
		{Key: "api_token", Label: "API token", Secret: true, Hint: "When auth_type is api_token"},
	}},
	{Network: "hilltopads", Label: "HilltopAds"},
	{Network: "clickadu", Label: "Clickadu"},
	{Network: "popads", Label: "PopAds"},
	{Network: "revcontent", Label: "Revcontent", AccountIDLabel: "Client ID", ExtraFields: []ExtraField{
		{Key: "client_id", Label: "Client ID", Hint: "Alternative to Account ID field"},
		{Key: "client_secret", Label: "Client secret", Secret: true, Hint: "Alternative to API key field"},
	}},
	{Network: "taboola", Label: "Taboola", ExtraFields: []ExtraField{
		{Key: "account_id", Label: "Taboola account ID", Hint: "Overrides Account ID when set"},
	}},
	{Network: "outbrain", Label: "Outbrain"},
	{Network: "tonic_rsoc", Label: "Tonic RSOC", ExtraFields: []ExtraField{
		{Key: "secret", Label: "API secret", Secret: true, Hint: "Basic auth secret paired with API key"},
	}},
	{Network: "system1_rsoc", Label: "System1 RSOC"},
	{Network: "mondiad", Label: "Mondiad", ExtraFields: []ExtraField{
		{Key: "client_id", Label: "Client ID", Required: true, Hint: "Mondiad dashboard API clientId; Account ID field also accepted"},
	}},
	{Network: "juicyads", Label: "JuicyAds", ExtraFields: []ExtraField{
		{Key: "campaign_type", Label: "Campaign type", Placeholder: "popunders", Hint: "JuicyAds API path segment; default popunders"},
	}},
	{Network: "evadav", Label: "Evadav"},
}

func ListNetworkCredentialSchemas() []NetworkCredentialSchema {
	out := make([]NetworkCredentialSchema, len(networkCredentialSchemas))
	copy(out, networkCredentialSchemas)
	return out
}

func CredentialSchemaForNetwork(network string) (NetworkCredentialSchema, bool) {
	for _, schema := range networkCredentialSchemas {
		if schema.Network == network {
			return schema, true
		}
	}
	return NetworkCredentialSchema{}, false
}

func ValidateExtraConfig(network string, extra map[string]string) error {
	schema, ok := CredentialSchemaForNetwork(network)
	if !ok {
		return fmt.Errorf("unsupported cost sync network %q", network)
	}
	if extra == nil {
		extra = map[string]string{}
	}
	for _, field := range schema.ExtraFields {
		if !field.Required {
			continue
		}
		if strings.TrimSpace(extra[field.Key]) == "" {
			return fmt.Errorf("extra_config.%s is required for network %s", field.Key, network)
		}
	}
	return nil
}

func MergeExtraConfig(existing, incoming map[string]string, schema NetworkCredentialSchema) map[string]string {
	merged := make(map[string]string, len(existing)+len(incoming))
	for k, v := range existing {
		merged[k] = v
	}
	if incoming == nil {
		return merged
	}
	secretKeys := make(map[string]struct{})
	for _, field := range schema.ExtraFields {
		if field.Secret {
			secretKeys[field.Key] = struct{}{}
		}
	}
	for k, v := range incoming {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			merged[k] = trimmed
			continue
		}
		if _, secret := secretKeys[k]; secret {
			continue
		}
		delete(merged, k)
	}
	return merged
}

func MaskExtraConfigForResponse(network string, extra map[string]string) (visible map[string]string, set map[string]bool) {
	visible = make(map[string]string)
	set = make(map[string]bool)
	if len(extra) == 0 {
		return visible, set
	}
	schema, ok := CredentialSchemaForNetwork(network)
	secretKeys := make(map[string]struct{})
	if ok {
		for _, field := range schema.ExtraFields {
			if field.Secret {
				secretKeys[field.Key] = struct{}{}
			}
		}
	}
	for k, v := range extra {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if _, secret := secretKeys[k]; secret {
			set[k] = true
			continue
		}
		visible[k] = v
	}
	return visible, set
}
