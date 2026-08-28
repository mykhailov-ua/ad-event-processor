package controlplane

import (
	"net/http"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/brand"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/doctor"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/governance"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/licensingadmin"
	"ad-event-processor/internal/marginguard"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/privacyadmin"
	"ad-event-processor/internal/reconciliation"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/rtbadmin"
	"ad-event-processor/internal/settingsadmin"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/internal/supply"
)

type SettingsStore = settingsadmin.Store
type PrivacyStore = privacyadmin.Store

type BrandDTO = brand.DTO
type BrandCreativeDTO = brand.CreativeDTO
type CreateBrandRequest = brand.CreateRequest
type UpsertBrandCreativeRequest = brand.UpsertCreativeRequest
type UpdateBrandCreativeRequest = brand.UpdateCreativeRequest

type BrandHTTPHandlers = brand.HTTPHandlers

type MarginGuardHTTPHandlers = marginguard.HTTPHandlers
type MarginGuardService = marginguard.Service
type MarginGuardActivityRow = marginguard.ActivityRow

type MarginGuardPolicy = ledger.Policy

type PlatformHTTPHandlers = platformadmin.HTTPHandlers
type PlatformConfigService = platformadmin.Service
type PlatformAuthClient = platformadmin.AuthClient

var (
	ErrPlatformConfigBootstrapped    = platformadmin.ErrConfigBootstrapped
	ErrPlatformConfigNotBootstrapped = platformadmin.ErrConfigNotBootstrapped
	ErrInstallTokenInvalid           = platformadmin.ErrInstallTokenInvalid
)

type (
	SlotMapDTO                = shardadmin.SlotMapDTO
	SlotMapVersionDTO         = shardadmin.SlotMapVersionDTO
	SlotMigrationDTO          = shardadmin.SlotMigrationDTO
	SlotMigrationOrchestrator = shardadmin.SlotMigrationOrchestrator
	ShardOrchestrator         = shardadmin.ShardOrchestrator
	ShardMetrics              = shardadmin.ShardMetrics
	ShardMetricsProvider      = shardadmin.ShardMetricsProvider
	RealShardMetricsProvider  = shardadmin.RealShardMetricsProvider
	ShardAutoscaleConfig      = shardadmin.ShardAutoscaleConfig
)

var (
	ErrSlotMigrationNotReady       = shardadmin.ErrSlotMigrationNotReady
	ErrSlotMigrationNoDraft        = shardadmin.ErrSlotMigrationNoDraft
	ErrSlotMigrationKeysMissing    = shardadmin.ErrSlotMigrationKeysMissing
	ErrSlotMigrationLagNotCaughtUp = shardadmin.ErrSlotMigrationLagNotCaughtUp
)

type (
	OperationLeaseWorker      = shardadmin.OperationLeaseWorker
	OperationLeaseBookRequest = shardadmin.OperationLeaseBookRequest
	OperationLeaseBookResult  = shardadmin.OperationLeaseBookResult
	OperationLeaseExecuteFunc = shardadmin.OperationLeaseExecuteFunc
	LeaseState                = shardadmin.LeaseState
	LeaseDedupScope           = shardadmin.LeaseDedupScope
	LeaseRenewHook            = shardadmin.LeaseRenewHook
	OpKeyPoolGate             = shardadmin.OpKeyPoolGate
	LeaseFencingRegistry      = shardadmin.LeaseFencingRegistry
	LeaseFencingStore         = shardadmin.LeaseFencingStore
	ProxyBatchBookInput       = shardadmin.ProxyBatchBookInput
)

var ProxyBatchBookRequest = shardadmin.ProxyBatchBookRequest

const (
	LeaseStateBooked    = shardadmin.LeaseStateBooked
	LeaseStateExecuting = shardadmin.LeaseStateExecuting
	LeaseStateCompleted = shardadmin.LeaseStateCompleted
	LeaseStateExpired   = shardadmin.LeaseStateExpired
)

var (
	ErrLeaseRenewExhausted = shardadmin.ErrLeaseRenewExhausted
	ErrLeaseQuorumNotMet   = shardadmin.ErrLeaseQuorumNotMet
	ErrOpKeyPoolShed       = shardadmin.ErrOpKeyPoolShed
	ErrStaleFencingEpoch   = shardadmin.ErrStaleFencingEpoch
)

var (
	RelayDeliveryOpID       = shardadmin.RelayDeliveryOpID
	ProxyBatchOpID          = shardadmin.ProxyBatchOpID
	ProxyBatchOpIDFromBytes = shardadmin.ProxyBatchOpIDFromBytes
	EncodeLeaseDedupScope   = shardadmin.EncodeLeaseDedupScope
	DecodeLeaseDedupScope   = shardadmin.DecodeLeaseDedupScope
	QuorumRequired          = shardadmin.QuorumRequired
	ScopeWithAttempt        = shardadmin.ScopeWithAttempt
	DedupScopeForLease      = shardadmin.DedupScopeForLease
	AuthoritativeLeaseView  = shardadmin.AuthoritativeLeaseView
	NewLeaseFencingRegistry = shardadmin.NewLeaseFencingRegistry
	NewLeaseFencingStore    = shardadmin.NewLeaseFencingStore
	ValidLeaseState         = shardadmin.ValidLeaseState
	LeaseOpID               = shardadmin.LeaseOpID
	LeaseFactorU            = shardadmin.LeaseFactorU
	LeaseDeadline           = shardadmin.LeaseDeadline
)

type (
	LicensingHTTPHandlers      = licensingadmin.HTTPHandlers
	LicenseStatusResponse      = licensingadmin.LicenseStatusResponse
	ApplyLicenseRequest        = licensingadmin.ApplyLicenseRequest
	LicenseService             = licensingadmin.LicenseService
	LicenseDiagnosticsProvider = licensingadmin.LicenseDiagnosticsProvider
	LicenseFeatureRequiredBody = licensingadmin.LicenseFeatureRequiredBody
	LicenseRevokeQueueWorker   = licensingadmin.RevokeQueueWorker
	EulaHTTPHandlers           = licensingadmin.EulaHTTPHandlers
	EulaStatusDTO              = licensingadmin.EulaStatusDTO
	AcceptEulaRequest          = licensingadmin.AcceptEulaRequest
	EulaService                = licensingadmin.EulaService
)

var ErrEulaVersionMismatch = licensingadmin.ErrEulaVersionMismatch

type BrokerPendingDeltaReader = reconciliation.BrokerPendingDeltaReader

type GlobalSpendReconciler = reconciliation.GlobalSpendReconciler

type GlobalSpendReconcilerConfig = reconciliation.GlobalSpendReconcilerConfig

var NewGlobalSpendReconciler = reconciliation.NewGlobalSpendReconciler

type QuotaRepairPayload = governance.QuotaRepairPayload

type ReconciliationAdjustPayload = reconciliation.ReconciliationAdjustPayload

type ReconService = reconciliation.ReconService

var NewReconService = reconciliation.NewReconService

type (
	FlowHTTPHandlers           = flow.HTTPHandlers
	FlowService                = flow.Service
	LanderDTO                  = flow.LanderDTO
	OfferDTO                   = flow.OfferDTO
	CreateLanderRequest        = flow.CreateLanderRequest
	CreateOfferRequest         = flow.CreateOfferRequest
	CreateFlowRequest          = flow.CreateFlowRequest
	UpdateFlowRequest          = flow.UpdateFlowRequest
	FlowDTO                    = flow.DTO
	HostedEditorFileDTO        = flow.HostedEditorFileDTO
	HostedEditorStateDTO       = flow.HostedEditorStateDTO
	HostedEditorFileBodyDTO    = flow.HostedEditorFileBodyDTO
	HostedEditorSaveResultDTO  = flow.HostedEditorSaveResultDTO
	HostedEditorPublishRequest = flow.HostedEditorPublishRequest
)

func validateFlowPathShape(paths []FlowPathDTO) error {
	return flow.ValidatePathShape(paths)
}

type SellerDTO = supply.SellerDTO
type AdsTxtEntryDTO = supply.AdsTxtEntryDTO
type SellerWriteRequest = supply.SellerWriteRequest
type AdsTxtWriteRequest = supply.AdsTxtWriteRequest
type SupplyExportPathDTO = supply.ExportPathDTO
type SupplyValidationDTO = supply.ValidationDTO
type SupplyChainNode = supply.ChainNode
type CampaignSupplyChainDTO = supply.CampaignChainDTO
type SellerCreateSpec = supply.SellerCreateSpec
type SellerUpdateSpec = supply.SellerUpdateSpec
type AdsTxtEntryCreateSpec = supply.AdsTxtEntryCreateSpec
type AdsTxtEntryUpdateSpec = supply.AdsTxtEntryUpdateSpec
type SupplyFilesPayload = supply.FilesPayload

var (
	ErrSellerNotFound      = supply.ErrSellerNotFound
	ErrAdsTxtEntryNotFound = supply.ErrAdsTxtEntryNotFound
	ErrInvalidSellerType   = supply.ErrInvalidSellerType
	ErrInvalidRelationship = supply.ErrInvalidRelationship
	ErrSupplyChainTooLong  = supply.ErrChainTooLong
	ErrSellersJSONInvalid  = supply.ErrSellersJSONInvalid
)

type SupplyHTTPHandlers = supply.HTTPHandlers

type (
	RtbHTTPHandlers           = rtbadmin.HTTPHandlers
	RtbFloorsHTTPHandlers     = rtbadmin.FloorsHTTPHandlers
	RtbDealDTO                = rtbadmin.DealDTO
	RtbDealCreateSpec         = rtbadmin.DealCreateSpec
	RtbDealUpdateSpec         = rtbadmin.DealUpdateSpec
	RtbService                = rtbadmin.DealService
	RtbShadowDiffSnapshotDTO  = rtbadmin.ShadowDiffSnapshotDTO
	RtbLiveGateDTO            = rtbadmin.LiveGateDTO
	RtbRuntimeConfigReader    = rtbadmin.RuntimeConfigReader
	RtbRuntimeHintsDTO        = rtbadmin.RuntimeHintsDTO
	RtbEndpointsDTO           = rtbadmin.EndpointsDTO
	RtbIntegrationResponseDTO = rtbadmin.IntegrationResponseDTO
	RtbReconcileExportDTO     = rtbadmin.ReconcileExportDTO
	RtbReconcileCHFunc        = rtbadmin.ReconcileCHFunc
	RtbFloorSuggestionDTO     = rtbadmin.FloorSuggestionDTO
	RtbFloorsApplyRequest     = rtbadmin.FloorsApplyRequest
	RtbFloorsApplyResult      = rtbadmin.FloorsApplyResult
	RtbFloorOptimizer         = rtbadmin.FloorOptimizer
	RtbBidShadeRequest        = rtbadmin.BidShadeRequest
	RtbBidShadeResponse       = rtbadmin.BidShadeResponse
	BidFloorRecommendationDTO = rtbadmin.BidFloorRecommendationDTO
)

func ValidateRtbModeSetting(mode string) (string, error) {
	return rtbadmin.ValidateRtbModeSetting(mode)
}

var (
	ErrRtbDealNotFound     = rtbadmin.ErrRtbDealNotFound
	ErrInvalidDealPacing   = rtbadmin.ErrInvalidDealPacing
	ErrDuplicateDealID     = rtbadmin.ErrDuplicateDealID
	ErrDealCustomerMissing = rtbadmin.ErrDealCustomerMissing
	ErrInvalidDealSeats    = rtbadmin.ErrInvalidDealSeats
)

type (
	DashboardsHTTPHandlers         = dashboardadmin.HTTPHandlers
	PeriodDTO                      = dashboardadmin.PeriodDTO
	MetricsBlockDTO                = dashboardadmin.MetricsBlockDTO
	ActionDTO                      = dashboardadmin.ActionDTO
	SourceRowDTO                   = dashboardadmin.SourceRowDTO
	BuyerAttentionDTO              = dashboardadmin.BuyerAttentionDTO
	BuyerCampaignPortfolioRowDTO   = dashboardadmin.BuyerCampaignPortfolioRowDTO
	BuyerPortfolioDTO              = dashboardadmin.BuyerPortfolioDTO
	BuyerPortfolioReader           = dashboardadmin.BuyerPortfolioReader
	CampaignDashboardReader        = dashboardadmin.CampaignDashboardReader
	RoleDashboardReader            = dashboardadmin.RoleDashboardReader
	BuyerCampaignRowDTO            = dashboardadmin.BuyerCampaignRowDTO
	RecommendationCardDTO          = dashboardadmin.RecommendationCardDTO
	AlertCardDTO                   = dashboardadmin.AlertCardDTO
	BuyerDashboardDTO              = dashboardadmin.BuyerDashboardDTO
	AccountantCloseDTO             = dashboardadmin.AccountantCloseDTO
	CFOSummaryDTO                  = dashboardadmin.CFOSummaryDTO
	CFODashboardDTO                = dashboardadmin.CFODashboardDTO
	AdOpsHealthDTO                 = dashboardadmin.AdOpsHealthDTO
	AdOpsDashboardDTO              = dashboardadmin.AdOpsDashboardDTO
	ExportJobStatusDTO             = dashboardadmin.ExportJobStatusDTO
	AccountantDashboardDTO         = dashboardadmin.AccountantDashboardDTO
	FraudOverviewDTO               = dashboardadmin.FraudOverviewDTO
	FraudDashboardDTO              = dashboardadmin.FraudDashboardDTO
	FraudGeoHintDTO                = dashboardadmin.FraudGeoHintDTO
	EdgeMetricsPanelDTO            = dashboardadmin.EdgeMetricsPanelDTO
	OperatorDashboardDTO           = dashboardadmin.OperatorDashboardDTO
	XDPPanelDTO                    = dashboardadmin.XDPPanelDTO
	CampaignDashboardDTO           = dashboardadmin.CampaignDashboardDTO
	SupplyValidationReport         = dashboardadmin.SupplyValidationReport
	PublisherBind                  = dashboardadmin.PublisherBind
	PublisherDashboardDTO          = dashboardadmin.PublisherDashboardDTO
	PublisherStatementDTO          = dashboardadmin.PublisherStatementDTO
	PublisherHTTPHandlers          = dashboardadmin.PublisherHTTPHandlers
	PublisherReader                = dashboardadmin.PublisherReader
	PublisherStatementListResponse = dashboardadmin.PublisherStatementListResponse
)

var ErrPublisherScopeRequired = dashboardadmin.ErrPublisherScopeRequired

type (
	BillingHTTPHandlers            = billingadmin.HTTPHandlers
	CryptoBillingWebhookHandlers   = billingadmin.CryptoWebhookHandlers
	CompositeReadService           = billingadmin.CompositeReadService
	UsageExportSpec                = billingadmin.UsageExportSpec
	UsageExportCursor              = billingadmin.UsageExportCursor
	AdminInvoiceFilters            = billingadmin.AdminInvoiceFilters
	ForecastDTO                    = billingadmin.ForecastDTO
	StatementDTO                   = billingadmin.StatementDTO
	WalletDTO                      = billingadmin.WalletDTO
	LedgerLineDTO                  = billingadmin.LedgerLineDTO
	InvariantDTO                   = billingadmin.InvariantDTO
	SummaryDTO                     = billingadmin.SummaryDTO
	DeliveryDTO                    = billingadmin.DeliveryDTO
	TaxProfileDTO                  = billingadmin.TaxProfileDTO
	DisputeRowDTO                  = billingadmin.DisputeRowDTO
	DisputeListResult              = billingadmin.DisputeListResult
	InvoiceRetryer                 = billingadmin.InvoiceRetryer
	InProcessInvoiceService        = billingadmin.InProcessInvoiceService
	VoidAuditor                    = billingadmin.VoidAuditor
	CustomerBalanceReader          = billingadmin.CustomerBalanceReader
	UsageDailyExporter             = billingadmin.UsageDailyExporter
	DisputeLister                  = billingadmin.DisputeLister
	PatchCustomerCostCenterRequest = platformadmin.PatchCustomerCostCenterRequest
	SelfServeInvoiceListResponse   = billingadmin.SelfServeInvoiceListResponse
	LedgerListResponse             = billingadmin.LedgerListResponse
)

var (
	NewCompositeReadService = billingadmin.NewCompositeReadService
	ParseUsageExportCursor  = billingadmin.ParseUsageExportCursor
	ParseStatementPeriod    = billingadmin.ParseStatementPeriod
	errExportLimit          = billingadmin.ErrExportLimit
)

const defaultExportChunkMaxBytes = billingadmin.DefaultExportChunkMaxBytes

func WriteBillingError(w http.ResponseWriter, err error) {
	billingadmin.WriteBillingError(w, err)
}

type (
	FraudHTTPHandlers             = fraudadmin.HTTPHandlers
	FraudDecisionDTO              = fraudadmin.FraudDecisionDTO
	FraudIntegrationDTO           = fraudadmin.FraudIntegrationDTO
	FraudOverrideRequest          = fraudadmin.FraudOverrideRequest
	FraudPolicyPresetDTO          = fraudadmin.FraudPolicyPresetDTO
	PatchFraudPolicyPresetRequest = fraudadmin.PatchFraudPolicyPresetRequest
	FraudTierThresholdsDTO        = fraudadmin.FraudTierThresholdsDTO
	MLManualLabelDTO              = fraudadmin.MLManualLabelDTO
	FraudManualLabelRow           = fraudadmin.FraudManualLabelRow
	FraudManualLabelRequest       = fraudadmin.FraudManualLabelRequest
	MLManualLabelRequest          = fraudadmin.MLManualLabelRequest
	FraudLabelsService            = fraudadmin.LabelsService
	FraudDecisionsService         = fraudadmin.DecisionsService
	FraudIntegrationsService      = fraudadmin.IntegrationsService
	FraudOverridesService         = fraudadmin.OverridesService
	FraudPresetsService           = fraudadmin.PresetsService
	FraudScoringOverrideRequest   = fraudadmin.FraudScoringOverrideRequest
	CampaignFraudConfigUpdate     = campaign.PatchCampaignFraudRequest
)

func FraudDecisionDisclaimer() string {
	return fraudadmin.DecisionDisclaimer
}

type (
	TeamMemberDTO            = platformadmin.TeamMemberDTO
	TeamBudgetApprovalDTO    = platformadmin.TeamBudgetApprovalDTO
	UpdateTeamMemberRequest  = platformadmin.UpdateTeamMemberRequest
	InviteTeamMemberRequest  = platformadmin.InviteTeamMemberRequest
	TeamOverviewDTO          = platformadmin.TeamOverviewDTO
	TeamHTTPHandlers         = platformadmin.TeamHTTPHandlers
	TeamOverviewService      = platformadmin.TeamOverviewService
	CustomerDTO              = platformadmin.CustomerDTO
	CustomerListResponse     = platformadmin.CustomerListResponse
	CustomersHTTPHandlers    = platformadmin.CustomersHTTPHandlers
	DomainHealthDTO          = platformadmin.DomainHealthDTO
	DomainSSLSetupResult     = platformadmin.DomainSSLSetupResult
	DomainTLSAllowedResponse = platformadmin.DomainTLSAllowedResponse
	ParkDomainRequest        = platformadmin.ParkDomainRequest
	ParkDomainResponse       = platformadmin.ParkDomainResponse
	DomainHealthHTTPHandlers = platformadmin.DomainHealthHTTPHandlers
	SupportFeedbackMeta      = platformadmin.SupportFeedbackMeta
	SupportFeedbackRecord    = platformadmin.SupportFeedbackRecord
)

type (
	auditReasonChange            = platformadmin.AuditReasonChange
	auditIDChange                = platformadmin.AuditIDChange
	auditIdempotencyMeta         = platformadmin.AuditIdempotencyMeta
	auditOutboxEventMeta         = platformadmin.AuditOutboxEventMeta
	auditTxSourceMeta            = platformadmin.AuditTxSourceMeta
	auditQuotaRepairMeta         = platformadmin.AuditQuotaRepairMeta
	auditQuotaDeadShardRelease   = platformadmin.AuditQuotaDeadShardRelease
	auditCreateCustomerChange    = platformadmin.AuditCreateCustomerChange
	auditAmountChange            = platformadmin.AuditAmountChange
	auditPaymentSettlementChange = platformadmin.AuditPaymentSettlementChange
	auditPaymentRefundChange     = platformadmin.AuditPaymentRefundChange
	auditPaymentDisputeChange    = platformadmin.AuditPaymentDisputeChange
	auditOverdraftChange         = platformadmin.AuditOverdraftChange
	auditPacingChange            = platformadmin.AuditPacingChange
	auditCampaignAdminChange     = platformadmin.AuditCampaignAdminChange
	auditCampaignBrandChange     = platformadmin.AuditCampaignBrandChange
	auditCampaignBudgetChange    = platformadmin.AuditCampaignBudgetChange
	auditBrandFcapChange         = platformadmin.AuditBrandFcapChange
	auditCampaignScheduleChange  = platformadmin.AuditCampaignScheduleChange
	auditPacingLoopAdjustment    = platformadmin.AuditPacingLoopAdjustment
	auditCreateCampaignChange    = platformadmin.AuditCreateCampaignChange
	auditCohortSnapshotChange    = platformadmin.AuditCohortSnapshotChange
	auditSellerCreateChange      = platformadmin.AuditSellerCreateChange
	auditSellerUpdateChange      = platformadmin.AuditSellerUpdateChange
	auditAdsTxtDomainChange      = platformadmin.AuditAdsTxtDomainChange
	auditSupplyChainChange       = platformadmin.AuditSupplyChainChange
	auditAutoscaleBudgetTransfer = platformadmin.AuditAutoscaleBudgetTransfer
	auditRtbDealCreateChange     = platformadmin.AuditRtbDealCreateChange
	auditRtbDealUpdateChange     = platformadmin.AuditRtbDealUpdateChange
	auditRtbDealDeleteChange     = platformadmin.AuditRtbDealDeleteChange
	auditSlotMapVersionCreated   = platformadmin.AuditSlotMapVersionCreated
	auditSlotMapMarkMigrating    = platformadmin.AuditSlotMapMarkMigrating
	auditSlotMapActivated        = platformadmin.AuditSlotMapActivated
	auditSlotMapRollback         = platformadmin.AuditSlotMapRollback
)

type LedgerDTO = BalanceLedgerDTO

type (
	ExportHTTPHandlers                = billingadmin.ExportHTTPHandlers
	ExportJobRunner                   = billingadmin.JobRunner
	ExportJobSpec                     = billingadmin.JobSpec
	JobSpec                           = billingadmin.JobSpec
	JobStatusDTO                      = billingadmin.JobStatusDTO
	LedgerExportJobStatusDTO          = billingadmin.JobStatusDTO
	CostSyncHTTPHandlers              = billingadmin.CostSyncHTTPHandlers
	CostSyncCredentialDTO             = billingadmin.CostSyncCredentialDTO
	UpsertCostSyncCredentialRequest   = billingadmin.UpsertCostSyncCredentialRequest
	RunCostSyncRequest                = billingadmin.RunCostSyncRequest
	CostSyncRunDTO                    = billingadmin.CostSyncRunDTO
	PostbackConfigDTO                 = campaign.PostbackConfigDTO
	UpdatePostbackConfigRequest       = campaign.UpdatePostbackConfigRequest
	PostbackDlqDTO                    = campaign.PostbackDlqDTO
	PostbackCampaignStatusDTO         = campaign.PostbackCampaignStatusDTO
	IntegrationSchemaHTTPHandlers     = campaign.IntegrationSchemaHTTPHandlers
	IntegrationSchemaDTO              = campaign.IntegrationSchemaDTO
	CreateIntegrationSchemaRequest    = campaign.CreateIntegrationSchemaRequest
	ApplyIntegrationSchemaRequest     = campaign.ApplyIntegrationSchemaRequest
	ApplyCampaignTemplatesRequest     = campaign.ApplyCampaignTemplatesRequest
	ApplyCampaignTemplatesResult      = campaign.ApplyCampaignTemplatesResult
	ImportTemplatesRequest            = campaign.ImportTemplatesRequest
	AffiliateStatusPresetDTO          = campaign.AffiliateStatusPresetDTO
	AffiliateStatusPresetEntryDTO     = campaign.AffiliateStatusPresetEntryDTO
	AutomationHTTPHandlers            = automation.HTTPHandlers
	AutomationRuleDTO                 = automation.RuleDTO
	UpsertAutomationRuleRequest       = automation.UpsertRuleRequest
	AutomationDryRunResponse          = automation.DryRunResponse
	MetaHTTPHandlers                  = platformadmin.MetaHTTPHandlers
	MetaResponseDTO                   = platformadmin.MetaResponseDTO
	MetaLicenseDTO                    = platformadmin.MetaLicenseDTO
	MetaLicenseBuildInput             = platformadmin.MetaLicenseBuildInput
	MetaEnrichOut                     = platformadmin.MetaEnrichOut
	MetaEnricher                      = platformadmin.MetaEnricher
	PlatformCampaignHTTPHandlers      = platformadmin.PlatformCampaignHTTPHandlers
	PlatformCampaignLinkDTO           = platformadmin.PlatformCampaignLinkDTO
	UpsertPlatformCampaignLinkRequest = platformadmin.UpsertPlatformCampaignLinkRequest
	PlatformCampaignMutationRequest   = platformadmin.PlatformCampaignMutationRequest
	PlatformCampaignMutationDTO       = platformadmin.PlatformCampaignMutationDTO
	SessionHTTPHandlers               = platformadmin.SessionHTTPHandlers
	SessionNavItemDTO                 = platformadmin.SessionNavItemDTO
	SessionResponseDTO                = platformadmin.SessionResponseDTO
	SelfServeHTTPHandlers             = campaign.SelfServeHTTPHandlers
	PostbackHTTPHandlers              = campaign.PostbackHTTPHandlers
	ViewsHTTPHandlers                 = reports.ViewsHTTPHandlers
	ViewsStore                        = reports.ViewsStore
	SavedViewDTO                      = reports.SavedViewDTO
	CreateViewRequest                 = reports.CreateViewRequest
	UpdateViewRequest                 = reports.UpdateViewRequest
	DoctorHTTPHandlers                = doctor.DoctorHTTPHandlers
	DoctorResponseDTO                 = doctor.DoctorResponseDTO
	SupportHTTPHandlers               = platformadmin.SupportHTTPHandlers
	SupportFeedbackRecorder           = platformadmin.SupportFeedbackRecorder
)

var (
	NewJobRunner                    = billingadmin.NewJobRunner
	NewViewsStore                   = reports.NewViewsStore
	BuildMetaLicense                = platformadmin.BuildMetaLicense
	ValidateReportScheduleForActor  = reports.ValidateReportScheduleForActor
	ErrViewNotFound                 = reports.ErrViewNotFound
	ValidateSelfServeAPIKeyScopes   = campaign.ValidateSelfServeAPIKeyScopes
	RestrictSnapshotForAPIKeyScopes = campaign.RestrictSnapshotForAPIKeyScopes
	DenyScopedAPIKeyOperatorReport  = campaign.DenyScopedAPIKeyOperatorReport
	ApplyAutomationPreset           = automation.ApplyPreset
)

const (
	JobStatusPending   = billingadmin.JobStatusPending
	JobStatusRunning   = billingadmin.JobStatusRunning
	JobStatusCompleted = billingadmin.JobStatusCompleted
	JobStatusFailed    = billingadmin.JobStatusFailed
	JobStatusCancelled = billingadmin.JobStatusCancelled
)

type (
	CampaignCreateSpec                    = campaign.CreateCampaignSpec
	CreateCampaignInput                   = campaign.CreateCampaignSpec
	CampaignTemplateDTO                   = campaign.CampaignTemplateDTO
	CampaignTemplateListResponse          = campaign.CampaignTemplateListResponse
	CampaignDTO                           = campaign.CampaignDTO
	CampaignListResponse                  = campaign.CampaignListResponse
	CampaignEventListResponse             = campaign.CampaignEventListResponse
	CampaignEventDTO                      = campaign.CampaignEventDTO
	PatchCampaignRequest                  = campaign.PatchCampaignRequest
	CloneCampaignSpec                     = campaign.CloneCampaignSpec
	CloneCampaignResult                   = campaign.CloneCampaignResult
	CampaignPublishCheckDTO               = campaign.CampaignPublishCheckDTO
	CampaignPublishBlockedError           = campaign.CampaignPublishBlockedError
	CampaignSmokeResultDTO                = campaign.CampaignSmokeResultDTO
	ConversionMappingDTO                  = campaign.ConversionMappingDTO
	ConversionMappingListResponse         = campaign.ConversionMappingListResponse
	ReplaceConversionMappingsRequest      = campaign.ReplaceConversionMappingsRequest
	CampaignWizardSessionDTO              = campaign.CampaignWizardSessionDTO
	CampaignWizardCommitResult            = campaign.CampaignWizardCommitResult
	CampaignWizardStored                  = campaign.CampaignWizardStored
	IntegrationHealthDTO                  = campaign.IntegrationHealthDTO
	IntegrationHealthRow                  = campaign.IntegrationHealthRow
	IntegrationHealthInput                = campaign.IntegrationHealthInput
	CampaignsHTTPHandlers                 = campaign.CampaignsHTTPHandlers
	CampaignReader                        = campaign.CampaignReader
	CampaignRuntime                       = campaign.Runtime
	CampaignFlowPathValidator             = campaign.CampaignFlowPathValidator
	ConversionMappingService              = campaign.ConversionMappingService
	CampaignMarginDTO                     = campaign.CampaignMarginDTO
	CampaignExportBundle                  = campaign.CampaignExportBundle
	CampaignExportCampaign                = campaign.CampaignExportCampaign
	CampaignExportLander                  = campaign.CampaignExportLander
	CampaignExportOffer                   = campaign.CampaignExportOffer
	CampaignExportFlow                    = campaign.CampaignExportFlow
	CampaignExportFlowPath                = campaign.CampaignExportFlowPath
	CampaignExportFlowLanderRef           = campaign.CampaignExportFlowLanderRef
	CampaignExportFlowOfferRef            = campaign.CampaignExportFlowOfferRef
	CampaignExportPostback                = campaign.CampaignExportPostback
	ImportCampaignSpec                    = campaign.ImportCampaignSpec
	ImportCampaignResult                  = campaign.ImportCampaignResult
	ImportMigrationSpec                   = campaign.ImportMigrationSpec
	ImportMigrationFailure                = campaign.ImportMigrationFailure
	ImportMigrationResult                 = campaign.ImportMigrationResult
	ImportValidateJobRequest              = campaign.ImportValidateJobRequest
	PullMigrationPreviewSpec              = campaign.PullMigrationPreviewSpec
	PullMigrationImportSpec               = campaign.PullMigrationImportSpec
	CampaignFraudConfigDTO                = campaign.CampaignFraudConfigDTO
	PatchCampaignFraudRequest             = campaign.PatchCampaignFraudRequest
	CampaignFraudPreviewDTO               = campaign.CampaignFraudPreviewDTO
	FraudPreviewTierCountsDTO             = campaign.FraudPreviewTierCountsDTO
	PreviewCampaignFraudRequest           = campaign.PreviewCampaignFraudRequest
	CampaignFraudService                  = campaign.CampaignFraudService
	CustomerBalanceDTO                    = campaign.CustomerBalanceDTO
	BalanceLedgerDTO                      = campaign.BalanceLedgerDTO
	LedgerExportResult                    = campaign.LedgerExportResult
	UsageExportResult                     = campaign.UsageExportResult
	FlowPathFiltersDTO                    = campaign.FlowPathFiltersDTO
	FlowPathErrorDTO                      = campaign.FlowPathErrorDTO
	CloneCampaignOptions                  = campaign.CloneCampaignOptions
	IngressCostConfigDTO                  = campaign.IngressCostConfigDTO
	AuditLogDTO                           = campaign.AuditLogDTO
	MutationPreviewDTO                    = campaign.MutationPreviewDTO
	MutationPreview                       = campaign.MutationPreviewDTO
	BlacklistDTO                          = campaign.BlacklistDTO
	CampaignSmokeRedirectHop              = campaign.CampaignSmokeRedirectHop
	CampaignWizardReviewDTO               = campaign.CampaignWizardReviewDTO
	CampaignWizardTrafficSourceStep       = campaign.CampaignWizardTrafficSourceStep
	CampaignWizardIntegrationTemplateStep = campaign.CampaignWizardIntegrationTemplateStep
	CampaignWizardFlowSkeletonStep        = campaign.CampaignWizardFlowSkeletonStep
	CampaignWizardBudgetStep              = campaign.CampaignWizardBudgetStep
	CampaignWizardStepsDTO                = campaign.CampaignWizardStepsDTO
	CampaignWizardPreviewDTO              = campaign.CampaignWizardPreviewDTO
	CampaignWizardAssetRef                = campaign.CampaignWizardAssetRef
	FlowPathDTO                           = campaign.FlowPathDTO
	FlowPathLanderRef                     = campaign.FlowPathLanderRef
	FlowPathOfferRef                      = campaign.FlowPathOfferRef
	CampaignStatsDTO                      = campaign.CampaignStatsDTO
	StatusHistoryDTO                      = campaign.StatusHistoryDTO
	CampaignHourlyBucketDTO               = campaign.CampaignHourlyBucketDTO
	CampaignDailyBucketDTO                = campaign.CampaignDailyBucketDTO
	SpendCurvePoint                       = campaign.SpendCurvePoint
	ForecastAdvisory                      = campaign.ForecastAdvisory
	CampaignForecastInput                 = campaign.CampaignForecastInput
	CampaignForecastDTO                   = campaign.CampaignForecastDTO
	FlowValidateResponseDTO               = campaign.FlowValidateResponseDTO
	BlacklistListResponse                 = campaign.BlacklistListResponse
	CampaignMetricsDTO                    = campaign.CampaignMetricsDTO
)

var (
	ErrForecastClickHouseTimeout = campaign.ErrForecastClickHouseTimeout
	ErrForecastUnavailable       = campaign.ErrForecastUnavailable
	ErrCampaignPublishBlocked    = campaign.ErrCampaignPublishBlocked
)
