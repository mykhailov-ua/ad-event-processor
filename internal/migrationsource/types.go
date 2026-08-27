package migrationsource

type SourceKind string

const (
	SourceKindKeitaroJSON     SourceKind = "keitaro_json"
	SourceKindKeitaroAdminAPI SourceKind = "keitaro_admin_api"
	SourceKindBinomJSON       SourceKind = "binom_json"
	SourceKindBinomReportAPI  SourceKind = "binom_report_api"
	SourceKindNativeV1        SourceKind = "native_v1"
)

const MaxPayloadBytes = 1 << 20

type Warning struct {
	Slug        string `json:"slug"`
	Message     string `json:"message"`
	CampaignRef string `json:"campaign_ref,omitempty"`
}

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

type PreviewResult struct {
	SourceKind      SourceKind       `json:"source_kind"`
	MappedCampaigns []MappedCampaign `json:"mapped_campaigns"`
	Warnings        []Warning        `json:"warnings,omitempty"`
	SecretsStripped int              `json:"secrets_stripped"`
}

type NormalizedCampaign struct {
	Ref               string
	Name              string
	TrafficSourceName string
	TrackingURL       string
	LanderURL         string
	PostbackURL       string
	BudgetUSD         float64
}

type NormalizedBundle struct {
	SourceKind SourceKind
	Campaigns  []NormalizedCampaign
}
