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

type OperationQuery<O extends keyof operations> = NonNullable<operations[O]['parameters']['query']>;

type OperationJsonBody<O extends keyof operations> = operations[O] extends {
  responses: {
    200: {
      content: {
        'application/json': infer Body;
      };
    };
  };
}
  ? Body
  : operations[O] extends {
        responses: {
          201: {
            content: {
              'application/json': infer Body;
            };
          };
        };
      }
    ? Body
    : operations[O] extends {
          responses: {
            202: {
              content: {
                'application/json': infer Body;
              };
            };
          };
        }
      ? Body
      : never;

type OperationJsonRequestBody<O extends keyof operations> = operations[O] extends {
  requestBody?: {
    content: {
      'application/json': infer Body;
    };
  };
}
  ? Body
  : never;

export type CustomerListResponse = components['schemas']['CustomerListResponse'] & {
  freshness_label?: string;
};

export type CustomerListQuery = OperationQuery<'customersList'>;

import type { CampaignListApiSortField } from '@/domains/campaigns/list/campaign_list_sort';

export type CampaignListQuery = Omit<OperationQuery<'campaignsList'>, 'sort'> & {
  sort?: CampaignListApiSortField;
  from?: string;
  to?: string;
};

export type CampaignListMetricsQuery = OperationQuery<'campaignsListMetrics'>;

export type CampaignListMetricsRow = components['schemas']['CampaignListMetricsRow'];
export type CampaignListMetricsBatchResponse =
  components['schemas']['CampaignListMetricsBatchResponse'];

export type CampaignStatusTotals = components['schemas']['CampaignStatusTotals'];

export type InvoiceListQuery = OperationQuery<'billingListInvoices'>;

export type BillingStatement = components['schemas']['BillingStatement'];
export type BillingForecast = components['schemas']['BillingForecast'];
export type Wallet = components['schemas']['Wallet'];
export type PaymentHistoryListResponse = components['schemas']['PaymentHistoryListResponse'];
export type PaymentSummary = components['schemas']['PaymentSummary'];
export type PaymentHistoryRow = components['schemas']['PaymentHistoryRow'];
export type BillingInvoiceLine = components['schemas']['BillingInvoiceLine'];
export type BillingInvariant = components['schemas']['BillingInvariant'];
export type BillingInvariantQuery = OperationQuery<'billingInvariant'>;
export type PreviewInvoiceRequest = components['schemas']['PreviewInvoiceRequest'];
export type InvoicePreview = components['schemas']['InvoicePreview'];
export type InvoiceDelivery = components['schemas']['InvoiceDelivery'];
export type InvoiceDeliveryListResponse = components['schemas']['InvoiceDeliveryListResponse'];
export type BillingLedgerLine = components['schemas']['BillingLedgerLine'];
export type InvoiceLedgerLinesResponse = components['schemas']['InvoiceLedgerLinesResponse'];
export type InvoiceLedgerLinesQuery = OperationQuery<'billingInvoiceLedgerLines'>;
export type BillingExportJobSpec = components['schemas']['BillingExportJobSpec'];
export type BillingExportJob = components['schemas']['BillingExportJob'];
export type BillingExportJobCreatedResponse =
  components['schemas']['BillingExportJobCreatedResponse'];

export type CustomerPaymentsListQuery = OperationQuery<'billingCustomerPayments'>;

export type DoctorSummary = components['schemas']['DoctorSummary'];
export type StackHealthSnapshot = components['schemas']['StackHealthSnapshot'];
export type DashboardSummary = components['schemas']['DashboardSummary'];
export type AuditLog = components['schemas']['AuditLog'];
export type AuditListQuery = OperationQuery<'auditList'>;
export type AuditExportQuery = OperationQuery<'auditExport'>;
export type CustomerBalance = components['schemas']['CustomerBalance'];
export type BalanceLedgerEntry = components['schemas']['BalanceLedgerEntry'];
export type CustomerLedgerListResponse = components['schemas']['CustomerLedgerListResponse'];
export type CustomerLedgerListQuery = OperationQuery<'billingCustomerLedger'>;
export type ReportCatalogResponse = components['schemas']['ReportCatalogResponse'];
export type ReportCatalogRow = components['schemas']['ReportCatalogRow'];
export type ReportMapEnvelope = components['schemas']['ReportMapEnvelope'];
export type ReportMapRow = components['schemas']['ReportMapRow'];
export type ClickLogEvent = components['schemas']['ClickLogEvent'];
export type ClickLogPostback = components['schemas']['ClickLogPostback'];
export type ClickLogReportResponse = components['schemas']['ClickLogReportResponse'];
export type DataFreshness = components['schemas']['DataFreshness'];
export type FraudEvidencePack = components['schemas']['FraudEvidencePack'];
export type WireSignalBreakdownRow = components['schemas']['WireSignalBreakdownRow'];
export type WireSignalBreakdownReportResponse =
  components['schemas']['WireSignalBreakdownReportResponse'];
export type FraudBreakdownRow = components['schemas']['FraudBreakdownRow'];
export type FraudBreakdownReportResponse = components['schemas']['FraudBreakdownReportResponse'];
export type CustomerFraudByTypeRow = components['schemas']['CustomerFraudByTypeRow'];
export type CustomerFraudByTypeReportResponse =
  components['schemas']['CustomerFraudByTypeReportResponse'];

export type FraudReasonsReportKey = 'fraud-breakdown' | 'wire-signal-breakdown';

export type FraudReasonRow = FraudBreakdownRow | WireSignalBreakdownRow;

export function fraudReasonPlacementId(row: FraudReasonRow): string | undefined {
  return 'placement_id' in row ? row.placement_id : undefined;
}

export function fraudReasonSignalsDegraded(row: FraudReasonRow): boolean {
  return 'signals_degraded' in row ? Boolean(row.signals_degraded) : false;
}
export type ReportJobSpec = components['schemas']['ReportJobSpec'];
export type TelegramReportExportRequest = components['schemas']['TelegramReportExportRequest'];
export type ReportJobStatus = components['schemas']['ReportJobStatus'];
export type DLQInboxEntry = components['schemas']['DLQInboxEntry'];
export type DLQInboxListResponse = components['schemas']['DLQInboxListResponse'];
export type DLQEntry = components['schemas']['DLQEntry'];
export type DLQListResponse = components['schemas']['DLQListResponse'];
export type ConsentRecord = components['schemas']['ConsentRecord'];
export type FraudIntegration = components['schemas']['FraudIntegration'];

export type ReportRunQuery = Omit<OperationQuery<'reportConversionTypePayout'>, 'customer_id'> & {
  customer_id?: string;
  click_id?: string;
};

export type FraudCatalogReportKey =
  | 'silent-reject-impression-funnel'
  | 'signal-effectiveness'
  | 'customer-fraud-by-dimension'
  | 'ivt-by-source'
  | 'layer-desync-summary'
  | 'layer-desync-drilldown'
  | 'rtt-split-tunnel'
  | 'filter-rejects';

export type SilentRejectImpressionFunnelRow =
  components['schemas']['SilentRejectImpressionFunnelRow'];
export type SilentRejectImpressionFunnelReportResponse =
  components['schemas']['SilentRejectImpressionFunnelReportResponse'];
export type SignalEffectivenessRow = components['schemas']['SignalEffectivenessRow'];
export type SignalEffectivenessReportResponse =
  components['schemas']['SignalEffectivenessReportResponse'];
export type CustomerFraudByDimensionRow = components['schemas']['CustomerFraudByDimensionRow'];
export type CustomerFraudByDimensionReportResponse =
  components['schemas']['CustomerFraudByDimensionReportResponse'];
export type IVTBySourceRow = components['schemas']['IVTBySourceRow'];
export type IVTBySourceReportResponse = components['schemas']['IVTBySourceReportResponse'];
export type LayerDesyncSummaryRow = components['schemas']['LayerDesyncSummaryRow'];
export type LayerDesyncSummaryReportResponse =
  components['schemas']['LayerDesyncSummaryReportResponse'];
export type FilterRejectRow = components['schemas']['FilterRejectRow'];
export type FilterRejectReportResponse = components['schemas']['FilterRejectReportResponse'];
export type LayerDesyncDrilldownReportResponse =
  components['schemas']['LayerDesyncDrilldownReportResponse'];
export type RTTSplitTunnelReportResponse = components['schemas']['RTTSplitTunnelReportResponse'];
export type CampaignToggleCohortReportResponse =
  components['schemas']['CampaignToggleCohortReportResponse'];
export type CampaignToggleCohortKPIRow = components['schemas']['CampaignToggleCohortKPIRow'];

export type FraudCatalogReportRow = Record<string, unknown>;

export type FraudCatalogReportResponse = {
  rows: FraudCatalogReportRow[];
  series?: FraudCatalogReportRow[];
  freshness?: DataFreshness;
  next_cursor?: string;
  truncated?: boolean;
};

export type FraudCatalogDimension = 'placement' | 'sub1' | 'sub2' | 'country' | 'campaign';

export type FraudCatalogReportQuery = {
  customer_id?: string;
  from?: string;
  to?: string;
  campaign_id?: string;
  limit?: number;
  offset?: number;
  dimension?: FraudCatalogDimension;
  compare?: boolean;
  slice?: boolean;
  layer_desync_count?: number;
};

export type CampaignToggleCohortQuery = OperationQuery<'reportCampaignToggleCohort'>;

export type ClickLogReportQuery = OperationQuery<'reportClickLog'>;

export type DlqInboxListQuery = OperationQuery<'opsListDlqInbox'> & {
  source?: string;
};

export type DlqListQuery = OperationQuery<'opsListDlq'>;

export type MLManualLabel = components['schemas']['MLManualLabel'];
export type FraudManualLabelRequest = components['schemas']['FraudManualLabelRequest'];
export type FraudManualLabelBulkRequest = components['schemas']['FraudManualLabelBulkRequest'];
export type FraudManualLabelBulkResponse = components['schemas']['FraudManualLabelBulkResponse'];
export type FraudPolicyPreset = components['schemas']['FraudPolicyPreset'];
export type PatchFraudPolicyPresetRequest = components['schemas']['PatchFraudPolicyPresetRequest'];

export type ModeratorCorpusTuple = {
  id: string;
  ja3: string;
  ja4?: string;
  tcp_sig?: string;
  webgl_renderer?: string;
  layer_desync_count?: number;
  note?: string;
  source?: string;
  created_at?: string;
  created_at_display?: string;
  updated_at?: string;
  updated_at_display?: string;
};

export type ModeratorCorpusListResponse = {
  items: ModeratorCorpusTuple[];
  total: number;
  limit: number;
  offset: number;
  last_refresh?: string;
};

export type ModeratorCorpusUpsertRequest = {
  ja3: string;
  ja4?: string;
  tcp_sig?: string;
  webgl_renderer?: string;
  layer_desync_count?: number;
  note?: string;
  source?: string;
};

export type ModeratorCorpusImportRequest = {
  csv: string;
};

export type ModeratorCorpusImportResponse = {
  upserted: number;
};

export type ModeratorCorpusPreviewResponse = {
  match_count_7d: number;
};
export type FraudOverrideRequest = components['schemas']['FraudOverrideRequest'];
export type RoleDashboard = components['schemas']['RoleDashboard'];
export type DashboardRole = 'buyer' | 'adops' | 'cfo' | 'accountant' | 'fraud' | 'operator';

export type DashboardQuery = OperationQuery<'dashboardBuyer'>;

export type CampaignDashboardQuery = OperationQuery<'dashboardCampaign'>;

export type FraudLabelsQuery = OperationQuery<'listFraudLabels'>;

export type FraudLabelsListResponse = components['schemas']['FraudLabelsListResponse'];

export type FraudDecisionQuery = OperationQuery<'getFraudDecision'>;

export type TeamOverviewQuery = operations['teamOverview']['parameters']['query'];

export type TeamBudgetApprovalsQuery = OperationQuery<'teamBudgetApprovalsList'>;

export type TeamBudgetApprovalsListResponse =
  components['schemas']['TeamBudgetApprovalsListResponse'];

export type TeamMembersQuery = OperationQuery<'teamMembersList'>;

export type TeamMembersListResponse = components['schemas']['TeamMembersListResponse'];

export type OpsBlacklistListQuery = OperationQuery<'opsListBlacklist'>;

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
export type OpsShardsResponse = OperationJsonBody<'opsListShards'>;
export type OpsShardCatchupResponse = OperationJsonBody<'opsShard0Catchup'>;
export type DashboardMetrics = components['schemas']['DashboardMetrics'];
export type DashboardMetricsQuery = OperationQuery<'opsDashboardMetrics'>;
export type ReconRun = components['schemas']['ReconRun'];
export type ReconListQuery = OperationQuery<'reconListRuns'>;

export type OpsOutboxListQuery = OperationQuery<'opsListOutbox'>;

export type SelfServeCampaignTemplate = components['schemas']['SelfServeCampaignTemplate'];
export type SelfServeTemplateListResponse = components['schemas']['SelfServeTemplateListResponse'];
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

export type PlacementBlockSuggestion = components['schemas']['PlacementBlockSuggestion'];
export type PlacementBlockSuggestionsResponse =
  components['schemas']['PlacementBlockSuggestionsResponse'];

export type CampaignIntegrationPanel = components['schemas']['CampaignIntegrationPanel'];
export type IntegrationHealthRow = components['schemas']['IntegrationHealthRow'];
export type CampaignIntegrationHealth = components['schemas']['CampaignIntegrationHealth'];
export type ApplyCampaignTemplatesRequest = components['schemas']['ApplyCampaignTemplatesRequest'];
export type ApplyCampaignTemplatesResult = components['schemas']['ApplyCampaignTemplatesResult'];
export type DryRunCampaignTemplatesResult = {
  campaign_id: string;
  target_url?: string;
  panel_postback_url?: string;
  postback_url_template?: string;
  postback_dry_run: PostbackDryRunResult;
};

export type CampaignFlowPathError = components['schemas']['CampaignFlowPathError'];
export type CampaignFlowValidateRequest = components['schemas']['CampaignFlowValidateRequest'];
export type CampaignFlowValidateResponse = components['schemas']['CampaignFlowValidateResponse'];

export type MacroPreviewRequest = components['schemas']['MacroPreviewRequest'];
export type MacroPreviewResponse = components['schemas']['MacroPreviewResponse'];
export type CloneCampaignOptions = components['schemas']['CloneCampaignOptions'];
export type CloneCampaignRequest = components['schemas']['CloneCampaignRequest'];
export type CloneCampaignPreview = components['schemas']['CloneCampaignPreview'];
export type CloneCampaignResult = components['schemas']['CloneCampaignResult'];
export type CampaignDiffRow = components['schemas']['CampaignDiffRow'];
export type CampaignDiffResponse = components['schemas']['CampaignDiffResponse'];
export type CampaignBulkActionRequest = components['schemas']['CampaignBulkActionRequest'];
export type CampaignBulkAction = CampaignBulkActionRequest['action'];
export type CampaignBulkActionResultRow = components['schemas']['CampaignBulkActionResultRow'];
export type CampaignBulkActionResponse = components['schemas']['CampaignBulkActionResponse'];
export type CampaignGeoSummary = components['schemas']['CampaignGeoSummary'];
export type CampaignFraudEditorSummary = components['schemas']['CampaignFraudEditorSummary'];

export type OpenRtbBidRequest = components['schemas']['OpenRtbBidRequest'];

export type CampaignFraudConfig = components['schemas']['CampaignFraudConfig'];
export type ConversionRejectRules = components['schemas']['ConversionRejectRules'];
export type PatchCampaignFraudRequest = components['schemas']['PatchCampaignFraudRequest'];
export type PreviewCampaignFraudRequest = components['schemas']['PreviewCampaignFraudRequest'];
export type CampaignFraudPreview = components['schemas']['CampaignFraudPreview'];

export type CampaignStats = components['schemas']['CampaignStats'];
export type CampaignStatsQuery = OperationQuery<'campaignsGetStats'>;
export type CampaignMargin = components['schemas']['CampaignMargin'];
export type CampaignEventListResponse = components['schemas']['CampaignEventListResponse'];
export type CampaignEventListQuery = OperationQuery<'campaignsListEvents'>;
export type ConversionMappingListResponse = components['schemas']['ConversionMappingListResponse'];
export type ConversionMapping = components['schemas']['ConversionMapping'];
export type ReplaceConversionMappingsRequest =
  components['schemas']['ReplaceConversionMappingsRequest'];
export type BlockCampaignPlacementRequest = components['schemas']['BlockCampaignPlacementRequest'];
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
export type CostSyncCredentialsQuery = OperationQuery<'costSyncListCredentials'>;
export type CostSyncHistoryQuery = OperationQuery<'costSyncListHistory'>;

export type PostbackConfig = components['schemas']['PostbackConfig'];
export type UpdatePostbackConfigRequest = components['schemas']['UpdatePostbackConfigRequest'];
export type PostbackDryRunResult = components['schemas']['PostbackDryRunResult'];
export type PostbackDlqEntry = components['schemas']['PostbackDlqEntry'];
export type PostbackCampaignStatus = components['schemas']['PostbackCampaignStatus'];
export type StatusOKResponse = components['schemas']['StatusOKResponse'];

export type IntegrationSchema = components['schemas']['IntegrationSchema'];
export type CreateIntegrationSchemaRequest =
  components['schemas']['CreateIntegrationSchemaRequest'];
export type ApplyIntegrationSchemaRequest = components['schemas']['ApplyIntegrationSchemaRequest'];
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
export type PlatformCampaignLinksQuery = OperationQuery<'platformCampaignsListLinks'>;
export type PlatformCampaignSyncRunRequest =
  components['schemas']['PlatformCampaignSyncRunRequest'];

export type AffiliateStatusPreset = components['schemas']['AffiliateStatusPreset'];

export type Flow = components['schemas']['Flow'];
export type FlowPath = components['schemas']['FlowPath'] & {
  filters?: {
    countries?: string[];
    devices?: string[];
    os?: string[];
    languages?: string[];
  };
};

export type FlowValidateResponse = {
  valid: boolean;
  path_errors?: CampaignFlowPathError[];
  suggested_fix_action?: string;
};
export type CreateFlowRequest = components['schemas']['CreateFlowRequest'];
export type UpdateFlowRequest = components['schemas']['UpdateFlowRequest'];
export type Lander = components['schemas']['Lander'];
export type CreateLanderRequest = components['schemas']['CreateLanderRequest'];
export type UpdateLanderRequest = components['schemas']['UpdateLanderRequest'];
export type HostedEditorState = components['schemas']['HostedEditorState'];
export type HostedEditorFile = components['schemas']['HostedEditorFile'];
export type HostedEditorFileBody = components['schemas']['HostedEditorFileBody'];
export type Offer = components['schemas']['Offer'];
export type CreateOfferRequest = components['schemas']['CreateOfferRequest'];
export type UpdateOfferRequest = components['schemas']['UpdateOfferRequest'];
export type Brand = components['schemas']['Brand'];
export type CreateBrandRequest = components['schemas']['CreateBrandRequest'];
export type UpdateBrandRequest = components['schemas']['UpdateBrandRequest'];
export type BrandCreative = components['schemas']['BrandCreative'];
export type UpdateBrandCreativeRequest = components['schemas']['UpdateBrandCreativeRequest'];
export type BrandsListQuery = OperationQuery<'brandsList'>;
export type DomainHealth = components['schemas']['DomainHealth'] & {
  acme_state?: string;
  cloudflare_proxied?: boolean;
  wildcard_zone?: string;
  pool_id?: string;
  pool_status?: string;
};

export type DomainBulkRequest = {
  hostnames?: string[];
  csv?: string;
  cloudflare_zone_id?: string;
  pool_id?: string;
};

export type DomainBulkJobRow = {
  hostname: string;
  ok: boolean;
  error?: string;
};

export type DomainBulkJobStatus = {
  job_id: string;
  kind: 'park_probe' | 'ssl';
  status: 'pending' | 'running' | 'completed' | 'failed';
  total: number;
  completed: number;
  failed: number;
  results?: DomainBulkJobRow[];
  error?: string;
  created_at: string;
  updated_at: string;
};

export type BurnDomainRequest = {
  delete_cloudflare?: boolean;
};

export type BurnDomainResponse = {
  hostname: string;
  pool_status?: string;
  cloudflare_deleted?: boolean;
};
export type AddDomainRequest = components['schemas']['AddDomainRequest'];
export type ParkDomainRequest = components['schemas']['ParkDomainRequest'];
export type ParkDomainResponse = components['schemas']['ParkDomainResponse'];
export type WildcardSSLRequest = {
  cloudflare_zone_id: string;
  zone_name: string;
  pool_id?: string;
  include_apex?: boolean;
};
export type WildcardSSLResponse = {
  id: string;
  pool_id: string;
  wildcard_hostname: string;
  acme_state: string;
  ssl_not_after?: string;
  cloudflare_proxied: boolean;
  message?: string;
};
export type CloudflareZone = {
  id: string;
  name: string;
};
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
export type AutomationPresetParameter = components['schemas']['AutomationPresetParameter'];
export type AutomationRule = components['schemas']['AutomationRule'];
export type AutomationDryRunResult = components['schemas']['AutomationDryRunResult'];
export type UpsertAutomationRuleRequest = components['schemas']['UpsertAutomationRuleRequest'];
export type AutomationListRulesQuery = OperationQuery<'automationListRules'>;

export type TrafficOptimizerPreset = components['schemas']['TrafficOptimizerPreset'];
export type TrafficOptimizerRule = components['schemas']['TrafficOptimizerRule'];
export type TrafficOptimizerDryRunResult = components['schemas']['TrafficOptimizerDryRunResult'];
export type UpsertTrafficOptimizerRuleRequest =
  components['schemas']['UpsertTrafficOptimizerRuleRequest'];
export type TrafficOptimizerListRulesQuery = OperationQuery<'trafficOptimizerListRules'>;

export type SmartAlertRule = components['schemas']['SmartAlertRule'];
export type SmartAlertEvent = components['schemas']['SmartAlertEvent'];
export type UpsertSmartAlertRuleRequest = components['schemas']['UpsertSmartAlertRuleRequest'];
export type SmartAlertsListRulesQuery = OperationQuery<'smartAlertsListRules'>;
export type SmartAlertsListHistoryQuery = OperationQuery<'smartAlertsListHistory'>;

export type MarginGuardPolicy = components['schemas']['MarginGuardPolicy'];
export type MarginGuardActivity = components['schemas']['MarginGuardActivity'];
export type MarginGuardOverrideRequest = components['schemas']['MarginGuardOverrideRequest'];
export type MarginGuardListPoliciesQuery = OperationQuery<'marginGuardListPolicies'>;
export type MarginGuardListActivityQuery = OperationQuery<'marginGuardListActivity'>;

export type PublisherDashboard = components['schemas']['PublisherDashboard'];
export type PublisherStatement = components['schemas']['PublisherStatement'];
export type PublisherStatementListResponse =
  components['schemas']['PublisherStatementListResponse'];
export type PublisherStatementsQuery = OperationQuery<'publisherStatements'>;

export type ReportSchedule = components['schemas']['ReportSchedule'];
export type CreateReportScheduleRequest = components['schemas']['CreateReportScheduleRequest'];
export type UpdateReportScheduleRequest = components['schemas']['UpdateReportScheduleRequest'];
export type ReportSchedulesListQuery = OperationQuery<'reportSchedulesList'>;

export type SavedView = components['schemas']['SavedView'];
export type CreateSavedViewRequest = components['schemas']['CreateSavedViewRequest'];
export type UpdateSavedViewRequest = components['schemas']['UpdateSavedViewRequest'];
export type SavedViewsListQuery = OperationQuery<'listSavedViews'>;

export type TelegramBot = components['schemas']['TelegramBot'];
export type TelegramPostback = components['schemas']['TelegramPostback'];
export type TelegramUpdatePostbackRequest = components['schemas']['TelegramUpdatePostbackRequest'];
export type TelegramDeeplink = components['schemas']['TelegramDeeplink'];
export type TelegramValidateRequest = components['schemas']['TelegramValidateRequest'];
export type TelegramValidateResult = components['schemas']['TelegramValidateResult'];
export type TelegramListPostbacksQuery = OperationQuery<'telegramListPostbacks'>;

export type CampaignForecast = components['schemas']['CampaignForecast'];
export type CampaignForecastRequest = components['schemas']['CampaignForecastRequest'];

export type SelfServeInvoiceListResponse = components['schemas']['SelfServeInvoiceListResponse'];
export type SelfServeInvoicesQuery = OperationQuery<'selfserveListInvoices'>;
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
export type CommandPaletteSearchQuery = OperationQuery<'commandPaletteSearch'>;
export type CommandPaletteOpenRequest = OperationJsonRequestBody<'commandPaletteOpen'>;

export type EulaStatus = components['schemas']['EulaStatus'];
export type AcceptEulaRequest = components['schemas']['AcceptEulaRequest'];
export type LicenseStatus = components['schemas']['LicenseStatus'];
export type ApplyLicenseRequest = components['schemas']['ApplyLicenseRequest'];
export type MetaResponse = components['schemas']['MetaResponse'];
export type DisputeRow = components['schemas']['DisputeRow'];
export type DisputeListResponse = components['schemas']['DisputeListResponse'];
export type DisputeListQuery = OperationQuery<'disputesList'>;
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

export type PostbackHealthRow = {
  campaign_id: string;
  provider: string;
  success_rate_24h?: number;
  p95_latency_ms?: number;
  last_error?: string;
  dlq_pending_count: number;
  health_status: 'ok' | 'warn' | 'fail';
};

export type PostbackHealthResponse = {
  rows: PostbackHealthRow[];
  alert_threshold_success_rate: number;
  runbook_path?: string;
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

export type AuthLoginRequest = components['schemas']['AuthLoginRequest'];
export type AuthUser = components['schemas']['SessionBootstrapUser'];
export type AuthLoginResponse = components['schemas']['PublicLoginResponse'];
export type AuthRefreshResponse = components['schemas']['AuthRefreshResponse'];
export type PublicActivateRequest = components['schemas']['PublicActivateRequest'];
export type PublicAcceptInviteRequest = components['schemas']['PublicAcceptInviteRequest'];
export type PublicLoginResponse = components['schemas']['PublicLoginResponse'];

export type OpsMlModelStatusResponse = OperationJsonBody<'opsMlModelStatus'>;
export type OpsMlModelEvalResponse = OperationJsonBody<'opsMlModelEval'>;
export type OpsDomainRotationResponse = OperationJsonBody<'opsDomainRotation'>;
export type OpsTlsAllowedResponse = OperationJsonBody<'opsTlsAllowedCheck'>;
export type OpsConsentProofsResponse = OperationJsonBody<'opsConsentProofs'>;
export type OpsRumResponse = OperationJsonBody<'opsRum'>;
export type TelegramReportExportResponse = OperationJsonBody<'reportTelegramExport'>;
