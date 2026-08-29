package campaign

import (
	"context"
	"encoding/json"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
)

type CloneCampaignSpec struct {
	SourceID       uuid.UUID
	NamePrefix     string
	NameSuffix     string
	IdempotencyKey string
	Options        CloneCampaignOptions
}

type CloneCampaignResult struct {
	ID       string `json:"id"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}

type CreateCampaignSpec struct {
	CustomerID       uuid.UUID
	BrandID          *uuid.UUID
	Name             string
	BudgetLimitMicro int64
	PacingMode       string
	DailyBudgetMicro int64
	Timezone         string
	FreqLimit        int32
	FreqWindow       int32
	TargetCountries  []string
	StartAt          *time.Time
	EndAt            *time.Time
	DaypartHours     []int16
	TemplateID       *uuid.UUID
	IdempotencyKey   string
}

type CampaignFraudConfigDTO struct {
	CampaignID               string                       `json:"campaign_id"`
	FraudThresholdPass       uint8                        `json:"fraud_threshold_pass"`
	FraudThresholdSuspect    uint8                        `json:"fraud_threshold_suspect"`
	FraudThresholdIVT        uint8                        `json:"fraud_threshold_ivt"`
	FraudThresholdBlock      uint8                        `json:"fraud_threshold_block"`
	SilentRejectEnabled      bool                         `json:"silent_reject_enabled"`
	BehaviorFlags            uint32                       `json:"behavior_flags"`
	CanvasRetestEnabled      bool                         `json:"canvas_retest_enabled"`
	CgnatIPPolicyEnabled     bool                         `json:"cgnat_ip_policy_enabled"`
	AcceptLangGeoEnabled     bool                         `json:"accept_lang_geo_enabled"`
	JSONSerializationEnabled bool                         `json:"json_serialization_enabled"`
	ConversionRejectRules    domain.ConversionRejectRules `json:"conversion_reject_rules"`
}

type PatchCampaignFraudRequest struct {
	Preset                   *string                       `json:"preset,omitempty"`
	FraudThresholdPass       *uint8                        `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect    *uint8                        `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT        *uint8                        `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock      *uint8                        `json:"fraud_threshold_block,omitempty"`
	SilentRejectEnabled      *bool                         `json:"silent_reject_enabled,omitempty"`
	BehaviorFlags            *uint32                       `json:"behavior_flags,omitempty"`
	CanvasRetestEnabled      *bool                         `json:"canvas_retest_enabled,omitempty"`
	CgnatIPPolicyEnabled     *bool                         `json:"cgnat_ip_policy_enabled,omitempty"`
	AcceptLangGeoEnabled     *bool                         `json:"accept_lang_geo_enabled,omitempty"`
	JSONSerializationEnabled *bool                         `json:"json_serialization_enabled,omitempty"`
	ConversionRejectRules    *domain.ConversionRejectRules `json:"conversion_reject_rules,omitempty"`
}

type patchCampaignFraudRequestRaw struct {
	Preset                   *string                       `json:"preset,omitempty"`
	FraudThresholdPass       *uint8                        `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect    *uint8                        `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT        *uint8                        `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock      *uint8                        `json:"fraud_threshold_block,omitempty"`
	SilentRejectEnabled      *bool                         `json:"silent_reject_enabled,omitempty"`
	SilentRejectPatchLegacy  *bool                         `json:"ghost_ivt_enabled,omitempty"`
	BehaviorFlags            *uint32                       `json:"behavior_flags,omitempty"`
	CanvasRetestEnabled      *bool                         `json:"canvas_retest_enabled,omitempty"`
	CgnatIPPolicyEnabled     *bool                         `json:"cgnat_ip_policy_enabled,omitempty"`
	AcceptLangGeoEnabled     *bool                         `json:"accept_lang_geo_enabled,omitempty"`
	JSONSerializationEnabled *bool                         `json:"json_serialization_enabled,omitempty"`
	ConversionRejectRules    *domain.ConversionRejectRules `json:"conversion_reject_rules,omitempty"`
}

func decodePatchCampaignFraudRequest(body []byte) (PatchCampaignFraudRequest, error) {
	var raw patchCampaignFraudRequestRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return PatchCampaignFraudRequest{}, err
	}
	req := PatchCampaignFraudRequest{
		Preset:                raw.Preset,
		FraudThresholdPass:    raw.FraudThresholdPass,
		FraudThresholdSuspect: raw.FraudThresholdSuspect,
		FraudThresholdIVT:     raw.FraudThresholdIVT,
		FraudThresholdBlock:   raw.FraudThresholdBlock,
		BehaviorFlags:         raw.BehaviorFlags,
	}
	if raw.CanvasRetestEnabled != nil {
		req.CanvasRetestEnabled = raw.CanvasRetestEnabled
	}
	if raw.CgnatIPPolicyEnabled != nil {
		req.CgnatIPPolicyEnabled = raw.CgnatIPPolicyEnabled
	}
	if raw.AcceptLangGeoEnabled != nil {
		req.AcceptLangGeoEnabled = raw.AcceptLangGeoEnabled
	}
	if raw.JSONSerializationEnabled != nil {
		req.JSONSerializationEnabled = raw.JSONSerializationEnabled
	}
	if raw.ConversionRejectRules != nil {
		req.ConversionRejectRules = raw.ConversionRejectRules
	}
	if raw.SilentRejectEnabled != nil {
		req.SilentRejectEnabled = raw.SilentRejectEnabled
	} else if raw.SilentRejectPatchLegacy != nil {
		req.SilentRejectEnabled = raw.SilentRejectPatchLegacy
	}
	return req, nil
}

type CampaignFraudPreviewDTO struct {
	CampaignID    string                    `json:"campaign_id"`
	AffectedIPs7d int64                     `json:"affected_ips_7d"`
	SampleSize    int64                     `json:"sample_size"`
	ByTier        FraudPreviewTierCountsDTO `json:"by_tier"`
	Disclaimer    string                    `json:"disclaimer"`
}

type FraudPreviewTierCountsDTO struct {
	Suspect int64 `json:"suspect"`
	IVT     int64 `json:"ivt"`
	Block   int64 `json:"block"`
}

type PreviewCampaignFraudRequest struct {
	Preset                *string `json:"preset,omitempty"`
	FraudThresholdPass    *uint8  `json:"fraud_threshold_pass,omitempty"`
	FraudThresholdSuspect *uint8  `json:"fraud_threshold_suspect,omitempty"`
	FraudThresholdIVT     *uint8  `json:"fraud_threshold_ivt,omitempty"`
	FraudThresholdBlock   *uint8  `json:"fraud_threshold_block,omitempty"`
}

type CampaignFraudService interface {
	GetCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID) (CampaignFraudConfigDTO, error)
	UpdateCampaignFraudConfig(ctx context.Context, campaignID uuid.UUID, req PatchCampaignFraudRequest) (CampaignFraudConfigDTO, error)
	PreviewCampaignFraudImpact(ctx context.Context, campaignID uuid.UUID, req PreviewCampaignFraudRequest) (CampaignFraudPreviewDTO, error)
}

type ConversionMappingDTO struct {
	InboundStatus string `json:"inbound_status"`
	GoalName      string `json:"goal_name"`
	PayoutMicro   int64  `json:"payout_micro"`
}

type ConversionMappingListResponse struct {
	Mappings []ConversionMappingDTO `json:"mappings"`
}

type ReplaceConversionMappingsRequest struct {
	Mappings []ConversionMappingDTO `json:"mappings"`
}

type CampaignPublishCheckDTO struct {
	Valid        bool              `json:"valid"`
	FieldErrors  map[string]string `json:"field_errors,omitempty"`
	WarningSlugs []string          `json:"warning_slugs,omitempty"`
}

type CampaignSmokeRedirectHop struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
}

type CampaignSmokeResultDTO struct {
	Passed        bool                       `json:"passed"`
	RedirectChain []CampaignSmokeRedirectHop `json:"redirect_chain"`
	FailureReason string                     `json:"failure_reason,omitempty"`
	CheckedAt     time.Time                  `json:"checked_at"`
	FinalHost     string                     `json:"final_host,omitempty"`
}

type CampaignWizardTrafficSourceStep struct {
	Name              string            `json:"name"`
	TrafficTemplateID string            `json:"traffic_template_id,omitempty"`
	ClickQueryParams  map[string]string `json:"click_query_params,omitempty"`
}

type CampaignWizardIntegrationTemplateStep struct {
	IntegrationSchema string `json:"integration_schema"`
	AffiliateNetwork  string `json:"affiliate_network,omitempty"`
	TrackingDomain    string `json:"tracking_domain,omitempty"`
}

type CampaignWizardFlowSkeletonStep struct {
	FlowName string                 `json:"flow_name"`
	Lander   CampaignWizardAssetRef `json:"lander"`
	Offer    CampaignWizardAssetRef `json:"offer"`
}

type CampaignWizardAssetRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CampaignWizardBudgetStep struct {
	BudgetLimitMicro int64    `json:"budget_limit_micro"`
	Timezone         string   `json:"timezone,omitempty"`
	TargetCountries  []string `json:"target_countries,omitempty"`
}

type CampaignWizardStepsDTO struct {
	TrafficSource       *CampaignWizardTrafficSourceStep       `json:"traffic_source,omitempty"`
	IntegrationTemplate *CampaignWizardIntegrationTemplateStep `json:"integration_template,omitempty"`
	FlowSkeleton        *CampaignWizardFlowSkeletonStep        `json:"flow_skeleton,omitempty"`
	Budget              *CampaignWizardBudgetStep              `json:"budget,omitempty"`
}

type CampaignWizardPreviewDTO struct {
	CampaignName      string `json:"campaign_name"`
	TrafficTemplateID string `json:"traffic_template_id,omitempty"`
	IntegrationSchema string `json:"integration_schema,omitempty"`
	FlowName          string `json:"flow_name,omitempty"`
	BudgetLimitMicro  int64  `json:"budget_limit_micro"`
	TargetURL         string `json:"target_url,omitempty"`
}

type CampaignWizardReviewDTO struct {
	Preview      CampaignWizardPreviewDTO `json:"preview"`
	WarningSlugs []string                 `json:"warning_slugs,omitempty"`
}

type CampaignWizardSessionDTO struct {
	SessionID      string                   `json:"session_id"`
	CustomerID     string                   `json:"customer_id"`
	CurrentStep    string                   `json:"current_step"`
	CompletedSteps []string                 `json:"completed_steps"`
	Steps          CampaignWizardStepsDTO   `json:"steps"`
	ReadyToCommit  bool                     `json:"ready_to_commit"`
	Review         *CampaignWizardReviewDTO `json:"review,omitempty"`
	ExpiresAt      time.Time                `json:"expires_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

type CampaignWizardCommitResult struct {
	Campaign     ImportCampaignResult     `json:"campaign"`
	Published    bool                     `json:"published,omitempty"`
	PublishCheck *CampaignPublishCheckDTO `json:"publish_check,omitempty"`
}

type CampaignWizardStored struct {
	TrafficSource       CampaignWizardTrafficSourceStep       `json:"traffic_source,omitempty"`
	IntegrationTemplate CampaignWizardIntegrationTemplateStep `json:"integration_template,omitempty"`
	FlowSkeleton        CampaignWizardFlowSkeletonStep        `json:"flow_skeleton,omitempty"`
	Budget              CampaignWizardBudgetStep              `json:"budget,omitempty"`
}

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

type IntegrationHealthStatus string

const (
	IntegrationHealthOK   IntegrationHealthStatus = "ok"
	IntegrationHealthWarn IntegrationHealthStatus = "warn"
	IntegrationHealthFail IntegrationHealthStatus = "fail"
)

type IntegrationHealthRow struct {
	Slug         string `json:"slug"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	FixRoute     string `json:"fix_route,omitempty"`
	ActionID     string `json:"action_id,omitempty"`
	FixHintLabel string `json:"fix_hint_label,omitempty"`
}

type IntegrationHealthDTO struct {
	CampaignID string                 `json:"campaign_id"`
	Summary    string                 `json:"summary"`
	Rows       []IntegrationHealthRow `json:"rows"`
}

type ListResponse[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

type CampaignMarginDTO struct {
	CampaignID           string `json:"campaign_id"`
	WindowStart          string `json:"window_start"`
	WindowHours          int    `json:"window_hours"`
	AdvertiserSpendMicro int64  `json:"advertiser_spend_micro"`
	RtbCostMicro         int64  `json:"rtb_cost_micro"`
	OperatorMarginMicro  int64  `json:"operator_margin_micro"`
	PublisherPayoutMicro int64  `json:"publisher_payout_micro"`
	CostOverRevenueLimit int64  `json:"cost_over_revenue_limit"`
	ThresholdBps         int    `json:"threshold_bps"`
	MarginBreach         bool   `json:"margin_breach"`
}

type ImportMigrationSpec struct {
	CustomerID       uuid.UUID
	IdempotencyKey   string
	SourceKind       migrationsource.SourceKind
	Payload          []byte
	NamePrefix       string
	BudgetLimitMicro *int64
}

type ImportMigrationFailure struct {
	Ref     string `json:"ref"`
	Name    string `json:"name,omitempty"`
	Message string `json:"message"`
}

type ImportMigrationResult struct {
	ImportBatchID string                    `json:"import_batch_id"`
	Imported      []ImportCampaignResult    `json:"imported"`
	Warnings      []migrationsource.Warning `json:"warnings,omitempty"`
	Failed        []ImportMigrationFailure  `json:"failed,omitempty"`
}

type PullMigrationPreviewSpec struct {
	SourceKind migrationsource.SourceKind
	BaseURL    string
	APIToken   string
	PullPath   string
}

type PullMigrationImportSpec struct {
	PullMigrationPreviewSpec
	CustomerID       uuid.UUID
	IdempotencyKey   string
	NamePrefix       string
	BudgetLimitMicro *int64
}

type MigratePreviewRequest struct {
	SourceKind string          `json:"source_kind"`
	Payload    json.RawMessage `json:"payload"`
}

type MigrateImportRequest struct {
	CustomerID       string          `json:"customer_id"`
	SourceKind       string          `json:"source_kind"`
	Payload          json.RawMessage `json:"payload"`
	NamePrefix       string          `json:"name_prefix,omitempty"`
	BudgetLimitMicro *int64          `json:"budget_limit_micro,omitempty"`
}

type MigratePullRequest struct {
	SourceKind       string `json:"source_kind"`
	BaseURL          string `json:"base_url"`
	APIToken         string `json:"api_token"`
	PullPath         string `json:"pull_path,omitempty"`
	CustomerID       string `json:"customer_id,omitempty"`
	NamePrefix       string `json:"name_prefix,omitempty"`
	BudgetLimitMicro *int64 `json:"budget_limit_micro,omitempty"`
}

type ImportValidateJobRequest struct {
	CustomerID string          `json:"customer_id"`
	SourceKind string          `json:"source_kind"`
	Payload    json.RawMessage `json:"payload"`
}

type StatusHistoryDTO struct {
	ID         int64  `json:"id"`
	CampaignID string `json:"campaign_id"`
	OldStatus  string `json:"old_status,omitempty"`
	NewStatus  string `json:"new_status"`
	Reason     string `json:"reason,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func StatusHistoryToDTO(r db.CampaignStatusHistory) StatusHistoryDTO {
	return statusHistoryToDTO(r)
}

func statusHistoryToDTO(r db.CampaignStatusHistory) StatusHistoryDTO {
	var oldStatus string
	if r.OldStatus.Valid {
		oldStatus = string(r.OldStatus.CampaignStatusType)
	}
	return StatusHistoryDTO{
		ID:         r.ID,
		CampaignID: uuid.UUID(r.CampaignID.Bytes).String(),
		OldStatus:  oldStatus,
		NewStatus:  string(r.NewStatus),
		Reason:     r.Reason.String,
		CreatedAt:  r.CreatedAt.Time.Format(time.RFC3339),
	}
}

type CampaignTemplateDTO struct {
	ID              string   `json:"id"`
	CustomerID      string   `json:"customer_id"`
	Name            string   `json:"name"`
	BudgetLimit     string   `json:"budget_limit"`
	PacingMode      string   `json:"pacing_mode"`
	DailyBudget     string   `json:"daily_budget"`
	Timezone        string   `json:"timezone"`
	FreqLimit       int32    `json:"freq_limit"`
	FreqWindow      int32    `json:"freq_window"`
	TargetCountries []string `json:"target_countries"`
	BrandID         string   `json:"brand_id,omitempty"`
	DaypartHours    []int16  `json:"daypart_hours"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type CampaignTemplateListResponse struct {
	Items []CampaignTemplateDTO `json:"items"`
	Total int64                 `json:"total"`
}

type CloneCampaignOptions struct {
	IncludeFlow            bool `json:"include_flow"`
	IncludePostbacks       bool `json:"include_postbacks"`
	IncludeFraud           bool `json:"include_fraud"`
	IncludePlacementBlocks bool `json:"include_placement_blocks"`
	ResetSpend             bool `json:"reset_spend"`
}

type IntegrationSchemaDTO struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Version   int             `json:"version"`
	Kind      string          `json:"kind"`
	Schema    json.RawMessage `json:"schema"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type AffiliateStatusPresetEntryDTO struct {
	InboundStatus string `json:"inbound_status"`
	GoalName      string `json:"goal_name"`
}

type AffiliateStatusPresetDTO struct {
	Name     string                          `json:"name"`
	Statuses []AffiliateStatusPresetEntryDTO `json:"statuses"`
}

type ApplyCampaignTemplatesRequest struct {
	TrafficSource    string `json:"traffic_source"`
	AffiliateNetwork string `json:"affiliate_network"`
	TrackingDomain   string `json:"tracking_domain"`
}

type ImportTemplatesRequest struct {
	Names []string `json:"names,omitempty"`
}

type ApplyCampaignTemplatesResult struct {
	CampaignID        string            `json:"campaign_id"`
	TrafficSource     map[string]string `json:"traffic_source,omitempty"`
	AffiliatePostback map[string]string `json:"affiliate_postback,omitempty"`
	AffiliateStatus   map[string]string `json:"affiliate_status,omitempty"`
}

type CampaignFlowPathValidator func(ctx context.Context, paths []flow.PathDTO) error

type CampaignConflictResponseDTO struct {
	Error          string      `json:"error"`
	ServerRevision string      `json:"server_revision"`
	ConflictFields []string    `json:"conflict_fields,omitempty"`
	MergeHintLabel string      `json:"merge_hint_label,omitempty"`
	Current        CampaignDTO `json:"current"`
}
