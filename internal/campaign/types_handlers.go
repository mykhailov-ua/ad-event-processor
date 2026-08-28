package campaign

import "time"

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
