import type { components, operations } from '../types/generated/openapi.js';
import type {
  BillingSummary,
  Campaign,
  CampaignListResponse,
  PatchCampaignRequest,
  Customer,
  Invoice,
  InvoiceListResponse,
  SessionResponse,
  TaxProfile,
} from '../types/index.js';

export type {
  BillingSummary,
  Campaign,
  CampaignListResponse,
  PatchCampaignRequest,
  Customer,
  Invoice,
  InvoiceListResponse,
  SessionResponse,
  TaxProfile,
};

export type CustomerListResponse = components['schemas']['CustomerListResponse'] & {
  freshness_label?: string;
};

export type CustomerListQuery = NonNullable<
  operations['customersList']['parameters']['query']
>;

export type CampaignListQuery = NonNullable<
  operations['campaignsList']['parameters']['query']
>;

export type InvoiceListQuery = NonNullable<
  operations['billingListInvoices']['parameters']['query']
>;

export type BillingStatement = components['schemas']['BillingStatement'];
export type BillingForecast = components['schemas']['BillingForecast'];
export type Wallet = components['schemas']['Wallet'];
export type PaymentHistoryListResponse = components['schemas']['PaymentHistoryListResponse'];
export type PaymentSummary = components['schemas']['PaymentSummary'];
export type PaymentHistoryRow = components['schemas']['PaymentHistoryRow'];
export type BillingInvoiceLine = components['schemas']['BillingInvoiceLine'];
export type BillingInvariant = components['schemas']['BillingInvariant'];
export type BillingInvariantQuery = NonNullable<
  operations['billingInvariant']['parameters']['query']
>;
export type PreviewInvoiceRequest = components['schemas']['PreviewInvoiceRequest'];
export type InvoicePreview = components['schemas']['InvoicePreview'];
export type InvoiceDelivery = components['schemas']['InvoiceDelivery'];
export type InvoiceDeliveryListResponse = components['schemas']['InvoiceDeliveryListResponse'];
export type BillingLedgerLine = components['schemas']['BillingLedgerLine'];
export type InvoiceLedgerLinesResponse = components['schemas']['InvoiceLedgerLinesResponse'];
export type InvoiceLedgerLinesQuery = NonNullable<
  operations['billingInvoiceLedgerLines']['parameters']['query']
>;
export type BillingExportJobSpec = components['schemas']['BillingExportJobSpec'];
export type BillingExportJob = components['schemas']['BillingExportJob'];
export type BillingExportJobCreatedResponse =
  components['schemas']['BillingExportJobCreatedResponse'];

export type CustomerPaymentsListQuery = NonNullable<
  operations['billingCustomerPayments']['parameters']['query']
>;

export type DoctorSummary = components['schemas']['DoctorSummary'];
export type StackHealthSnapshot = components['schemas']['StackHealthSnapshot'];
export type DashboardSummary = components['schemas']['DashboardSummary'];
export type AuditLog = components['schemas']['AuditLog'];
export type AuditListQuery = NonNullable<operations['auditList']['parameters']['query']>;
export type AuditExportQuery = NonNullable<operations['auditExport']['parameters']['query']>;
export type CustomerBalance = components['schemas']['CustomerBalance'];
export type BalanceLedgerEntry = components['schemas']['BalanceLedgerEntry'];
export type CustomerLedgerListResponse = components['schemas']['CustomerLedgerListResponse'];
export type CustomerLedgerListQuery = NonNullable<
  operations['billingCustomerLedger']['parameters']['query']
>;
export type ReportCatalogResponse = components['schemas']['ReportCatalogResponse'];
export type ReportCatalogRow = components['schemas']['ReportCatalogRow'];
export type ReportMapEnvelope = components['schemas']['ReportMapEnvelope'];
export type ReportMapRow = components['schemas']['ReportMapRow'];
export type ClickLogEvent = components['schemas']['ClickLogEvent'];
export type ClickLogPostback = components['schemas']['ClickLogPostback'];
export type ClickLogReportResponse = components['schemas']['ClickLogReportResponse'];
export type DataFreshness = components['schemas']['DataFreshness'];
export type FraudEvidencePack = components['schemas']['FraudEvidencePack'];
export type ReportJobSpec = components['schemas']['ReportJobSpec'];
export type TelegramReportExportRequest = components['schemas']['TelegramReportExportRequest'];
export type ReportJobStatus = components['schemas']['ReportJobStatus'];
export type DLQInboxEntry = components['schemas']['DLQInboxEntry'];
export type DLQInboxListResponse = components['schemas']['DLQInboxListResponse'];
export type FraudIntegration = {
  campaign_id: string;
  name?: string;
  provider?: string;
  configured?: boolean;
  health_status?: string;
  last_success_at?: string;
  dlq_count?: number;
  last_error?: string;
};

export type ReportRunQuery = {
  customer_id?: string;
  from?: string;
  to?: string;
  campaign_id?: string;
  click_id?: string;
  limit?: number;
  offset?: number;
  cursor?: string;
};

export type ClickLogReportQuery = {
  customer_id: string;
  from?: string;
  to?: string;
  campaign_id?: string;
  click_id?: string;
  cursor?: string;
};

export type DlqInboxListQuery = {
  limit?: number;
  cursor?: string;
  source?: string;
};

export type MLManualLabel = components['schemas']['MLManualLabel'];
export type FraudManualLabelRequest = components['schemas']['FraudManualLabelRequest'];
export type FraudManualLabelBulkRequest = components['schemas']['FraudManualLabelBulkRequest'];
export type FraudManualLabelBulkResponse = components['schemas']['FraudManualLabelBulkResponse'];
export type FraudPolicyPreset = components['schemas']['FraudPolicyPreset'];
export type PatchFraudPolicyPresetRequest = components['schemas']['PatchFraudPolicyPresetRequest'];
export type FraudOverrideRequest = {
  campaign_id?: string;
  ip?: string;
  ip_hash?: string;
};
export type RoleDashboard = components['schemas']['RoleDashboard'];
export type DashboardRole =
  | 'buyer'
  | 'adops'
  | 'cfo'
  | 'accountant'
  | 'fraud'
  | 'operator';

export type DashboardQuery = {
  customer_id: string;
  campaign_id?: string;
  from?: string;
  to?: string;
};

export type FraudLabelsQuery = {
  customer_id: string;
  limit?: number;
  offset?: number;
};

export type FraudLabelsListResponse = components['schemas']['FraudLabelsListResponse'];

export type FraudDecisionQuery = {
  customer_id: string;
  ip_hash: string;
  campaign_id?: string;
  hours?: number;
};

export type TeamOverviewQuery = {
  customer_id?: string;
};

export type TeamBudgetApprovalsQuery = {
  customer_id: string;
  limit?: number;
  offset?: number;
};

export type TeamBudgetApprovalsListResponse = components['schemas']['TeamBudgetApprovalsListResponse'];

export type TeamMembersQuery = {
  customer_id: string;
  limit?: number;
  offset?: number;
};

export type TeamMembersListResponse = components['schemas']['TeamMembersListResponse'];

export type OpsBlacklistListQuery = {
  limit?: number;
  offset?: number;
};

export type PlatformSettingsView = components['schemas']['PlatformSettingsView'];
export type PlatformSettingsPatch = components['schemas']['PlatformSettingsPatch'];
export type PlatformBootstrapRequest = components['schemas']['PlatformBootstrapRequest'];
export type PlatformApplyRequest = components['schemas']['PlatformApplyRequest'];
export type PlatformApplyResponse = components['schemas']['PlatformApplyResponse'];
export type InviteTeamMemberRequest = components['schemas']['InviteTeamMemberRequest'];
export type UpdateTeamMemberRequest = components['schemas']['UpdateTeamMemberRequest'];
export type TeamOverview = components['schemas']['TeamOverview'];
export type TeamMember = components['schemas']['TeamMember'];
export type TeamBudgetApproval = components['schemas']['TeamBudgetApproval'];
export type FraudDecision = components['schemas']['FraudDecision'];
export type OpsBlacklistEntry = components['schemas']['OpsBlacklistEntry'];
export type OpsBlacklistListResponse = components['schemas']['OpsBlacklistListResponse'];
export type OpsBlacklistWriteRequest = components['schemas']['OpsBlacklistWriteRequest'];
export type OpsBlacklistDeleteRequest = components['schemas']['OpsBlacklistDeleteRequest'];

export type IncidentSnapshot = components['schemas']['IncidentSnapshot'];
export type OutboxListResponse = components['schemas']['OutboxListResponse'];
export type OutboxEvent = components['schemas']['OutboxEvent'];
export type ShardHealthStatus = components['schemas']['ShardHealthStatus'];
export type OpsShardsResponse = {
  emergency_breaker?: string;
  shards?: ShardHealthStatus[];
};
export type OpsShardCatchupResponse = {
  status?: string;
};
export type DashboardMetrics = components['schemas']['DashboardMetrics'];
export type DashboardMetricsQuery = NonNullable<
  operations['opsDashboardMetrics']['parameters']['query']
>;
export type ReconRun = components['schemas']['ReconRun'];
export type ReconListQuery = NonNullable<operations['reconListRuns']['parameters']['query']>;

export type OpsOutboxListQuery = {
  limit?: number;
  cursor?: string;
};

export type SelfServeCampaignTemplate = components['schemas']['SelfServeCampaignTemplate'];
export type SelfServeTemplateListResponse =
  components['schemas']['SelfServeTemplateListResponse'];
export type SelfServeCreateCampaignRequest =
  components['schemas']['SelfServeCreateCampaignRequest'];

export type CampaignValidateResponse = components['schemas']['CampaignValidateResponse'];
export type CampaignPublishCheck = components['schemas']['CampaignPublishCheck'];
export type CampaignPublishBlockedError = components['schemas']['CampaignPublishBlockedError'];
export type IngressCostConfig = components['schemas']['IngressCostConfig'];

export type MigrationPreviewResult = components['schemas']['MigrationPreviewResult'];
export type MigratePreviewRequest = components['schemas']['MigratePreviewRequest'];
export type MigrateImportRequest = components['schemas']['MigrateImportRequest'];
export type MigratePullRequest = components['schemas']['MigratePullRequest'];
export type MigrationSourcesResponse = components['schemas']['MigrationSourcesResponse'];
export type ImportValidateJobRequest = components['schemas']['ImportValidateJobRequest'];
export type ImportMigrationResult = components['schemas']['ImportMigrationResult'];
export type ImportCampaignRequest = components['schemas']['ImportCampaignRequest'];
export type ImportCampaignResult = components['schemas']['ImportCampaignResult'];
export type AssignCampaignOwnerRequest = components['schemas']['AssignCampaignOwnerRequest'];
export type CampaignExportBundle = components['schemas']['CampaignExportBundle'];
export type CampaignEditorShell = components['schemas']['CampaignEditorShell'];

export type PlacementBlockSuggestion = {
  placement_id: string;
  impressions?: number;
  ivt_rate?: number;
  ivt_rate_label?: string;
  reason_label?: string;
  suggested_action?: string;
};

export type PlacementBlockSuggestionsResponse = {
  items: PlacementBlockSuggestion[];
  campaign_id?: string;
  from?: string;
  to?: string;
};

export type CampaignIntegrationPanel = components['schemas']['CampaignIntegrationPanel'];
export type CampaignIntegrationHealth = components['schemas']['CampaignIntegrationHealth'];
export type ApplyCampaignTemplatesRequest = components['schemas']['ApplyCampaignTemplatesRequest'];
export type ApplyCampaignTemplatesResult = components['schemas']['ApplyCampaignTemplatesResult'];

export type CampaignFraudConfig = components['schemas']['CampaignFraudConfig'];
export type PatchCampaignFraudRequest = components['schemas']['PatchCampaignFraudRequest'];
export type PreviewCampaignFraudRequest = components['schemas']['PreviewCampaignFraudRequest'];
export type CampaignFraudPreview = components['schemas']['CampaignFraudPreview'];

export type CampaignStats = components['schemas']['CampaignStats'];
export type CampaignStatsQuery = NonNullable<
  operations['campaignsGetStats']['parameters']['query']
>;
export type CampaignMargin = components['schemas']['CampaignMargin'];
export type CampaignEventListResponse = components['schemas']['CampaignEventListResponse'];
export type CampaignEventListQuery = NonNullable<
  operations['campaignsListEvents']['parameters']['query']
>;
export type ConversionMappingListResponse =
  components['schemas']['ConversionMappingListResponse'];
export type ConversionMapping = components['schemas']['ConversionMapping'];
export type ReplaceConversionMappingsRequest =
  components['schemas']['ReplaceConversionMappingsRequest'];
export type BlockCampaignPlacementRequest =
  components['schemas']['BlockCampaignPlacementRequest'];
export type CampaignSmokeResult = components['schemas']['CampaignSmokeResult'];

export type CampaignWizardSession = components['schemas']['CampaignWizardSession'];
export type CampaignWizardSessionRequest = components['schemas']['CampaignWizardSessionRequest'];
export type CampaignWizardCommitResult = components['schemas']['CampaignWizardCommitResult'];
export type CampaignOnboardingTemplate = components['schemas']['CampaignOnboardingTemplate'];

export type CostSyncNetworkSchema = components['schemas']['CostSyncNetworkSchema'];
export type CostSyncCredential = components['schemas']['CostSyncCredential'];
export type UpsertCostSyncCredentialRequest =
  components['schemas']['UpsertCostSyncCredentialRequest'];
export type CostSyncRun = components['schemas']['CostSyncRun'];
export type RunCostSyncRequest = components['schemas']['RunCostSyncRequest'];
export type RunCostSyncAcceptedResponse = components['schemas']['RunCostSyncAcceptedResponse'];
export type CostSyncCredentialsQuery = NonNullable<
  operations['costSyncListCredentials']['parameters']['query']
>;
export type CostSyncHistoryQuery = NonNullable<
  operations['costSyncListHistory']['parameters']['query']
>;

export type PostbackConfig = components['schemas']['PostbackConfig'];
export type UpdatePostbackConfigRequest = components['schemas']['UpdatePostbackConfigRequest'];
export type PostbackDryRunResult = components['schemas']['PostbackDryRunResult'];
export type PostbackDlqEntry = components['schemas']['PostbackDlqEntry'];
export type PostbackCampaignStatus = components['schemas']['PostbackCampaignStatus'];
export type StatusOKResponse = components['schemas']['StatusOKResponse'];

export type IntegrationSchema = components['schemas']['IntegrationSchema'];
export type CreateIntegrationSchemaRequest =
  components['schemas']['CreateIntegrationSchemaRequest'];
export type ApplyIntegrationSchemaRequest =
  components['schemas']['ApplyIntegrationSchemaRequest'];
export type ApplyIntegrationSchemaResponse =
  components['schemas']['ApplyIntegrationSchemaResponse'];
export type IntegrationTemplateCatalogEntry =
  components['schemas']['IntegrationTemplateCatalogEntry'];
export type ImportIntegrationTemplatesRequest =
  components['schemas']['ImportIntegrationTemplatesRequest'];

export type PlatformCampaignLink = components['schemas']['PlatformCampaignLink'];
export type UpsertPlatformCampaignLinkRequest =
  components['schemas']['UpsertPlatformCampaignLinkRequest'];
export type PlatformCampaignMutationRequest =
  components['schemas']['PlatformCampaignMutationRequest'];
export type PlatformCampaignMutation = components['schemas']['PlatformCampaignMutation'];
export type PlatformCampaignLinksQuery = NonNullable<
  operations['platformCampaignsListLinks']['parameters']['query']
>;
export type PlatformCampaignSyncRunRequest =
  components['schemas']['PlatformCampaignSyncRunRequest'];

export type AffiliateStatusPreset = components['schemas']['AffiliateStatusPreset'];

export type Flow = components['schemas']['Flow'];
export type FlowPath = components['schemas']['FlowPath'];
export type CreateFlowRequest = components['schemas']['CreateFlowRequest'];
export type UpdateFlowRequest = components['schemas']['UpdateFlowRequest'];
export type Lander = components['schemas']['Lander'];
export type CreateLanderRequest = components['schemas']['CreateLanderRequest'];
export type HostedEditorState = components['schemas']['HostedEditorState'];
export type HostedEditorFile = components['schemas']['HostedEditorFile'];
export type Offer = components['schemas']['Offer'];
export type CreateOfferRequest = components['schemas']['CreateOfferRequest'];
export type Brand = components['schemas']['Brand'];
export type CreateBrandRequest = components['schemas']['CreateBrandRequest'];
export type BrandCreative = components['schemas']['BrandCreative'];
export type UpdateBrandCreativeRequest = components['schemas']['UpdateBrandCreativeRequest'];
export type BrandsListQuery = NonNullable<operations['brandsList']['parameters']['query']>;
export type DomainHealth = components['schemas']['DomainHealth'];
export type AddDomainRequest = components['schemas']['AddDomainRequest'];
export type ParkDomainRequest = components['schemas']['ParkDomainRequest'];
export type ParkDomainResponse = components['schemas']['ParkDomainResponse'];
export type DomainSSLSetupResult = components['schemas']['DomainSSLSetupResult'];
export type Seller = components['schemas']['Seller'];
export type SellerWriteRequest = components['schemas']['SellerWriteRequest'];
export type AdsTxtEntry = components['schemas']['AdsTxtEntry'];
export type AdsTxtWriteRequest = components['schemas']['AdsTxtWriteRequest'];
export type SupplyExportPath = components['schemas']['SupplyExportPath'];
export type SupplyValidation = components['schemas']['SupplyValidation'];

export type RtbDeal = components['schemas']['RtbDeal'];
export type RtbDealCreateSpec = components['schemas']['RtbDealCreateSpec'];
export type RtbDealUpdateSpec = components['schemas']['RtbDealUpdateSpec'];
export type RtbFloorsApplyRequest = components['schemas']['RtbFloorsApplyRequest'];
export type RtbFloorsApplyResult = components['schemas']['RtbFloorsApplyResult'];
export type RtbFloorSuggestion = components['schemas']['RtbFloorSuggestion'];
export type RtbIntegrationProfile = components['schemas']['RtbIntegrationProfile'];
export type RtbShadowDiffSnapshot = components['schemas']['RtbShadowDiffSnapshot'];
export type RtbReconcileExport = components['schemas']['RtbReconcileExport'];
export type OpenRtbValidationResult = components['schemas']['OpenRtbValidationResult'];

export type AutomationPreset = components['schemas']['AutomationPreset'];
export type AutomationRule = components['schemas']['AutomationRule'];
export type AutomationDryRunResult = components['schemas']['AutomationDryRunResult'];
export type UpsertAutomationRuleRequest = components['schemas']['UpsertAutomationRuleRequest'];
export type AutomationListRulesQuery = NonNullable<
  operations['automationListRules']['parameters']['query']
>;

export type TrafficOptimizerPreset = components['schemas']['TrafficOptimizerPreset'];
export type TrafficOptimizerRule = components['schemas']['TrafficOptimizerRule'];
export type TrafficOptimizerDryRunResult = components['schemas']['TrafficOptimizerDryRunResult'];
export type UpsertTrafficOptimizerRuleRequest =
  components['schemas']['UpsertTrafficOptimizerRuleRequest'];
export type TrafficOptimizerListRulesQuery = NonNullable<
  operations['trafficOptimizerListRules']['parameters']['query']
>;

export type SmartAlertRule = components['schemas']['SmartAlertRule'];
export type SmartAlertEvent = components['schemas']['SmartAlertEvent'];
export type UpsertSmartAlertRuleRequest = components['schemas']['UpsertSmartAlertRuleRequest'];
export type SmartAlertsListRulesQuery = NonNullable<
  operations['smartAlertsListRules']['parameters']['query']
>;
export type SmartAlertsListHistoryQuery = NonNullable<
  operations['smartAlertsListHistory']['parameters']['query']
>;

export type MarginGuardPolicy = components['schemas']['MarginGuardPolicy'];
export type MarginGuardActivity = components['schemas']['MarginGuardActivity'];
export type MarginGuardOverrideRequest = components['schemas']['MarginGuardOverrideRequest'];
export type MarginGuardListPoliciesQuery = NonNullable<
  operations['marginGuardListPolicies']['parameters']['query']
>;
export type MarginGuardListActivityQuery = NonNullable<
  operations['marginGuardListActivity']['parameters']['query']
>;

export type PublisherDashboard = components['schemas']['PublisherDashboard'];
export type PublisherStatement = components['schemas']['PublisherStatement'];
export type PublisherStatementListResponse =
  components['schemas']['PublisherStatementListResponse'];
export type PublisherStatementsQuery = NonNullable<
  operations['publisherStatements']['parameters']['query']
>;

export type ReportSchedule = components['schemas']['ReportSchedule'];
export type CreateReportScheduleRequest = components['schemas']['CreateReportScheduleRequest'];
export type UpdateReportScheduleRequest = components['schemas']['UpdateReportScheduleRequest'];
export type ReportSchedulesListQuery = NonNullable<
  operations['reportSchedulesList']['parameters']['query']
>;

export type SavedView = components['schemas']['SavedView'];
export type CreateSavedViewRequest = components['schemas']['CreateSavedViewRequest'];
export type UpdateSavedViewRequest = components['schemas']['UpdateSavedViewRequest'];
export type SavedViewsListQuery = NonNullable<
  operations['listSavedViews']['parameters']['query']
>;

export type TelegramBot = components['schemas']['TelegramBot'];
export type TelegramPostback = components['schemas']['TelegramPostback'];
export type TelegramUpdatePostbackRequest = components['schemas']['TelegramUpdatePostbackRequest'];
export type TelegramDeeplink = components['schemas']['TelegramDeeplink'];
export type TelegramValidateRequest = components['schemas']['TelegramValidateRequest'];
export type TelegramValidateResult = components['schemas']['TelegramValidateResult'];
export type TelegramListPostbacksQuery = NonNullable<
  operations['telegramListPostbacks']['parameters']['query']
>;

export type CampaignForecast = components['schemas']['CampaignForecast'];
export type CampaignForecastRequest = components['schemas']['CampaignForecastRequest'];

export type SelfServeInvoiceListResponse = components['schemas']['SelfServeInvoiceListResponse'];
export type SelfServeInvoicesQuery = NonNullable<
  operations['selfserveListInvoices']['parameters']['query']
>;
export type CreatePaymentIntentRequest = components['schemas']['CreatePaymentIntentRequest'];
export type PaymentIntentCreatedResponse = components['schemas']['PaymentIntentCreatedResponse'];
export type CreateAPIKeyRequest = components['schemas']['CreateAPIKeyRequest'];
export type APIKeyCreatedResponse = components['schemas']['APIKeyCreatedResponse'];
export type SelfServePauseCampaignRequest = components['schemas']['SelfServePauseCampaignRequest'];

export type CommandPaletteItem = components['schemas']['CommandPaletteItem'];
export type CommandPaletteSearchResponse = components['schemas']['CommandPaletteSearchResponse'];
export type CommandPaletteRoutesResponse = components['schemas']['CommandPaletteRoutesResponse'];
export type CommandPaletteRecentsResponse = components['schemas']['CommandPaletteRecentsResponse'];
export type CommandPaletteRecordRecentRequest =
  components['schemas']['CommandPaletteRecordRecentRequest'];
export type CommandPaletteSearchQuery = NonNullable<
  operations['commandPaletteSearch']['parameters']['query']
>;
export type CommandPaletteOpenRequest = {
  source?: string;
};

export type EulaStatus = components['schemas']['EulaStatus'];
export type AcceptEulaRequest = components['schemas']['AcceptEulaRequest'];
export type LicenseStatus = components['schemas']['LicenseStatus'];
export type ApplyLicenseRequest = components['schemas']['ApplyLicenseRequest'];
export type MetaResponse = components['schemas']['MetaResponse'];
export type DisputeRow = components['schemas']['DisputeRow'];
export type DisputeListResponse = components['schemas']['DisputeListResponse'];
export type DisputeListQuery = NonNullable<operations['disputesList']['parameters']['query']>;
export type SupportFeedbackMeta = components['schemas']['SupportFeedbackMeta'];
export type CreateSupportFeedbackRequest = components['schemas']['SupportFeedbackRequest'];
export type SupportFeedbackResponse = components['schemas']['SupportFeedbackResponse'];

export type AuditListResult = {
  items: AuditLog[];
  total: number;
};

export type OpsHomeSnapshot = {
  doctor: DoctorSummary;
  stackHealth: StackHealthSnapshot;
  dashboardSummary: DashboardSummary;
};

export type PostbacksSnapshot = {
  configs: PostbackConfig[];
  dlq: PostbackDlqEntry[];
  campaignStatus: PostbackCampaignStatus[];
};

export type CostSyncSnapshot = {
  networks: CostSyncNetworkSchema[];
  credentials: CostSyncCredential[];
  history: CostSyncRun[];
};

export type IntegrationSnapshot = {
  schemas: IntegrationSchema[];
  templates: IntegrationTemplateCatalogEntry[];
};

/**
 * Auth wire types mirror internal/control/http/http_auth.go.
 * Login routes are not fully described in api/openapi yet; keep aligned with handler DTOs.
 */
export type AuthLoginRequest = {
  email: string;
  password: string;
};

export type AuthUser = {
  id: string;
  email?: string;
  role: string;
  customer_id: string;
  permissions?: string[];
};

export type AuthLoginResponse = {
  user: AuthUser;
};

/** Mirrors api/openapi/components/schemas/platform.yaml until bundle regen includes public auth schemas. */
export type PublicActivateRequest = {
  license_token: string;
  email: string;
  password: string;
  team_name: string;
};

export type PublicAcceptInviteRequest = {
  token: string;
  password: string;
};

export type PublicLoginResponse = {
  user: AuthUser;
};

export type AuthRefreshResponse = {
  status: string;
};
