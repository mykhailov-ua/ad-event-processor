package campaign

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CampaignExportBundle struct {
	ExportVersion               int                     `json:"export_version"`
	ExportedAt                  string                  `json:"exported_at"`
	Campaign                    CampaignExportCampaign  `json:"campaign"`
	Flow                        *CampaignExportFlow     `json:"flow,omitempty"`
	Landers                     []CampaignExportLander  `json:"landers,omitempty"`
	Offers                      []CampaignExportOffer   `json:"offers,omitempty"`
	PostbackConfig              *CampaignExportPostback `json:"postback_config,omitempty"`
	ConversionMappings          []ConversionMappingDTO  `json:"conversion_mappings,omitempty"`
	IntegrationSchemaName       string                  `json:"integration_schema_name,omitempty"`
	StatusIntegrationSchemaName string                  `json:"status_integration_schema_name,omitempty"`
}

type CampaignExportCampaign struct {
	Name                       string                `json:"name"`
	BudgetLimitMicro           int64                 `json:"budget_limit_micro"`
	PacingMode                 string                `json:"pacing_mode,omitempty"`
	DailyBudgetMicro           int64                 `json:"daily_budget_micro,omitempty"`
	Timezone                   string                `json:"timezone,omitempty"`
	FreqLimit                  int32                 `json:"freq_limit,omitempty"`
	FreqWindow                 int32                 `json:"freq_window,omitempty"`
	TargetCountries            []string              `json:"target_countries,omitempty"`
	TargetURL                  string                `json:"target_url,omitempty"`
	SafePageURL                string                `json:"safe_page_url,omitempty"`
	SafePageEnabled            bool                  `json:"safe_page_enabled,omitempty"`
	AttestationEnabled         bool                  `json:"attestation_enabled,omitempty"`
	AttestationMode            string                `json:"attestation_mode,omitempty"`
	AttestationTTLSec          int32                 `json:"attestation_ttl_sec,omitempty"`
	DmrEnabled                 bool                  `json:"dmr_enabled,omitempty"`
	CIDRBlockEnabled           bool                  `json:"cidr_block_enabled,omitempty"`
	ProxyVPNBlockEnabled       bool                  `json:"proxy_vpn_block_enabled,omitempty"`
	ModeratorIntelEnabled      bool                  `json:"moderator_intel_enabled,omitempty"`
	ReviewTrafficAction        string                `json:"review_traffic_action,omitempty"`
	TLSFingerprintBlockEnabled bool                  `json:"tls_fingerprint_block_enabled,omitempty"`
	ConnTypePolicy             string                `json:"conn_type_policy,omitempty"`
	LinkSigningEnabled         bool                  `json:"link_signing_enabled,omitempty"`
	LinkSigningTTLSec          int32                 `json:"link_signing_ttl_sec,omitempty"`
	ClickDelivery              string                `json:"click_delivery,omitempty"`
	ProxyUpstreamURL           string                `json:"proxy_upstream_url,omitempty"`
	ProxyRewriteAssets         bool                  `json:"proxy_rewrite_assets,omitempty"`
	ReferrerFilter             string                `json:"referrer_filter,omitempty"`
	StartAt                    string                `json:"start_at,omitempty"`
	EndAt                      string                `json:"end_at,omitempty"`
	DaypartHours               []int16               `json:"daypart_hours,omitempty"`
	IngressCostConfig          *IngressCostConfigDTO `json:"ingress_cost_config,omitempty"`
	TrafficTemplateID          string                `json:"traffic_template_id,omitempty"`
	ClickQueryParams           map[string]string     `json:"click_query_params,omitempty"`
	CreativePayload            json.RawMessage       `json:"creative_payload,omitempty"`
	FraudThresholdPass         int16                 `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect      int16                 `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT          int16                 `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock        int16                 `json:"fraud_threshold_block,omitempty"`
	SilentRejectEnabled        bool                  `json:"silent_reject_enabled,omitempty"`
}

type CampaignExportLander struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type CampaignExportOffer struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CampaignExportFlow struct {
	Name  string                   `json:"name"`
	Paths []CampaignExportFlowPath `json:"paths"`
}

type CampaignExportFlowPath struct {
	Weight  int32                         `json:"weight"`
	Filters *FlowPathFiltersDTO           `json:"filters,omitempty"`
	Landers []CampaignExportFlowLanderRef `json:"landers"`
	Offers  []CampaignExportFlowOfferRef  `json:"offers"`
}

type CampaignExportFlowLanderRef struct {
	Ref    string `json:"ref"`
	Weight int32  `json:"weight"`
}

type CampaignExportFlowOfferRef struct {
	Ref      string `json:"ref"`
	Weight   int32  `json:"weight"`
	CapDaily *int32 `json:"cap_daily,omitempty"`
	CapTotal *int32 `json:"cap_total,omitempty"`
}

type CampaignExportPostback struct {
	Provider      string `json:"provider"`
	URLTemplate   string `json:"url_template"`
	TargetEvent   string `json:"target_event,omitempty"`
	TestEventCode string `json:"test_event_code,omitempty"`
}

type ImportCampaignSpec struct {
	CustomerID     uuid.UUID
	NameOverride   string
	BudgetOverride *int64
	IdempotencyKey string
	Bundle         CampaignExportBundle
}

type ImportCampaignResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
