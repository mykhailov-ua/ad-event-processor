package migrationsource

// SourceKind identifies an external migration payload format.
type SourceKind string

const (
	SourceKindKeitaroJSON SourceKind = "keitaro_json"
	SourceKindBinomJSON   SourceKind = "binom_json"
	SourceKindNativeV1    SourceKind = "native_v1"
)

// MaxPayloadBytes is the upper bound for migrate preview/import request bodies.
const MaxPayloadBytes = 1 << 20

// Warning describes a non-fatal mapping issue during preview.
type Warning struct {
	Slug        string `json:"slug"`
	Message     string `json:"message"`
	CampaignRef string `json:"campaign_ref,omitempty"`
}

// MappedCampaign is a preview row for one imported campaign.
type MappedCampaign struct {
	Ref                   string            `json:"ref"`
	Name                  string            `json:"name"`
	TrafficSourceName     string            `json:"traffic_source_name,omitempty"`
	BundledSlug           string            `json:"bundled_slug,omitempty"`
	UITemplateID          string            `json:"ui_template_id,omitempty"`
	IntegrationSchemaName string            `json:"integration_schema_name,omitempty"`
	ClickQueryParams      map[string]string `json:"click_query_params,omitempty"`
	TargetURL             string            `json:"target_url,omitempty"`
	BudgetLimitMicro      int64             `json:"budget_limit_micro,omitempty"`
	IngressCostParam      string            `json:"ingress_cost_param,omitempty"`
	PostbackURLTemplate   string            `json:"postback_url_template,omitempty"`
}

// PreviewResult is the parse-only output for migrate preview.
type PreviewResult struct {
	SourceKind      SourceKind       `json:"source_kind"`
	MappedCampaigns []MappedCampaign `json:"mapped_campaigns"`
	Warnings        []Warning        `json:"warnings,omitempty"`
	SecretsStripped int              `json:"secrets_stripped"`
}

// NormalizedCampaign is the internal intermediate after adapter parse.
type NormalizedCampaign struct {
	Ref               string
	Name              string
	TrafficSourceName string
	TrackingURL       string
	LanderURL         string
	PostbackURL       string
	BudgetUSD         float64
}

// NormalizedBundle aggregates parsed campaigns before macro mapping.
type NormalizedBundle struct {
	SourceKind SourceKind
	Campaigns  []NormalizedCampaign
}
