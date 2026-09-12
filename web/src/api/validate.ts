// Runtime shape guards for high-traffic list/mutation responses.
// Mismatch throws ApiError(502, INVALID_RESPONSE); callers surface via ErrorBlock.
import { ApiError } from '@/api/api_error';
import type {
  AuditLog,
  AutomationRule,
  Campaign,
  CampaignBulkActionResponse,
  CampaignBulkActionResultRow,
  CampaignFlowValidateResponse,
  CampaignListMetricsBatchResponse,
  CampaignListMetricsTotalsResponse,
  CampaignListResponse,
  CampaignPublishBlockedError,
  CampaignValidateResponse,
  CampaignWizardCommitResult,
  CampaignWizardSession,
  Flow,
  FlowPath,
  FlowPathFilters,
  FraudBreakdownReportResponse,
  FraudCatalogReportKey,
  FraudCatalogReportResponseMap,
  FraudBreakdownRow,
  Lander,
  LanderHostingCounts,
  LanderListResponse,
  Offer,
  PostbackDryRunResult,
  HostedEditorFileBody,
  HostedEditorSaveResult,
  HostedEditorState,
  MarginGuardActivity,
  MarginGuardPolicy,
  TrafficOptimizerDryRunResult,
  TrafficOptimizerPreset,
  TrafficOptimizerRule,
  WireSignalBreakdownReportResponse,
  WireSignalBreakdownRow,
} from '@/api/types';

export function isRecord(value: unknown): value is Record<string, unknown> {
  return value != null && typeof value === 'object' && !Array.isArray(value);
}

function invalidResponse(message: string): never {
  throw new ApiError(502, 'INVALID_RESPONSE', message);
}

export function parseAuditLogRow(value: unknown): AuditLog {
  if (!isRecord(value)) {
    invalidResponse('Audit log entry must be an object');
  }
  if (typeof value.id !== 'number' || typeof value.action !== 'string') {
    invalidResponse('Audit log entry missing id or action');
  }
  return value as AuditLog;
}

export function parseAuditLogList(value: unknown): AuditLog[] {
  if (!Array.isArray(value)) {
    invalidResponse('Expected JSON array response');
  }
  return value.map(parseAuditLogRow);
}

export function parseCampaign(value: unknown): Campaign {
  if (
    !isRecord(value) ||
    typeof value.id !== 'string' ||
    typeof value.name !== 'string' ||
    typeof value.status !== 'string' ||
    typeof value.customer_id !== 'string'
  ) {
    invalidResponse('Campaign missing required fields');
  }
  return value as Campaign;
}

export function parseCampaignListMetricsTotalsResponse(
  value: unknown
): CampaignListMetricsTotalsResponse {
  if (
    !isRecord(value) ||
    typeof value.campaign_count !== 'number' ||
    typeof value.flow_count !== 'number' ||
    typeof value.margin_breach_count !== 'number' ||
    typeof value.from !== 'string' ||
    typeof value.to !== 'string' ||
    typeof value.stale !== 'boolean' ||
    !isRecord(value.totals)
  ) {
    invalidResponse('Campaign metrics totals response missing required fields');
  }
  return value as CampaignListMetricsTotalsResponse;
}

export function parseCampaignListMetricsBatchResponse(
  value: unknown
): CampaignListMetricsBatchResponse {
  if (
    !isRecord(value) ||
    !isRecord(value.items) ||
    typeof value.from !== 'string' ||
    typeof value.to !== 'string' ||
    typeof value.stale !== 'boolean'
  ) {
    invalidResponse('Campaign metrics batch response missing required fields');
  }
  return value as CampaignListMetricsBatchResponse;
}

function parseOptionalStringArray(value: unknown, label: string): string[] | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value) || value.some((item) => typeof item !== 'string')) {
    invalidResponse(`${label} must be a string array`);
  }
  return value;
}

function parseFlowPathFilters(value: unknown): FlowPathFilters | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (!isRecord(value)) {
    invalidResponse('Flow path filters must be an object');
  }
  return {
    countries: parseOptionalStringArray(value.countries, 'Flow path filters.countries'),
    devices: parseOptionalStringArray(value.devices, 'Flow path filters.devices'),
    os: parseOptionalStringArray(value.os, 'Flow path filters.os'),
    languages: parseOptionalStringArray(value.languages, 'Flow path filters.languages'),
  };
}

function parseFlowPathRef(value: unknown, label: string, idField: 'lander_id' | 'offer_id'): void {
  if (!isRecord(value) || typeof value[idField] !== 'string' || typeof value.weight !== 'number') {
    invalidResponse(`${label} missing ${idField} or weight`);
  }
}

export function parseFlowPath(value: unknown): FlowPath {
  if (
    !isRecord(value) ||
    typeof value.weight !== 'number' ||
    !Array.isArray(value.landers) ||
    !Array.isArray(value.offers)
  ) {
    invalidResponse('Flow path missing required fields');
  }
  for (const lander of value.landers) {
    parseFlowPathRef(lander, 'Flow path lander ref', 'lander_id');
  }
  for (const offer of value.offers) {
    parseFlowPathRef(offer, 'Flow path offer ref', 'offer_id');
  }
  const filters = parseFlowPathFilters(value.filters);
  return {
    ...(value as FlowPath),
    filters,
  };
}

export function parseFlowPathList(value: unknown): FlowPath[] {
  if (!Array.isArray(value)) {
    invalidResponse('Flow paths must be an array');
  }
  return value.map(parseFlowPath);
}

function parseFlowPathsField(value: unknown): Flow['paths'] {
  if (typeof value === 'string') {
    return value;
  }
  return parseFlowPathList(value);
}

export function parseFlow(value: unknown): Flow {
  if (
    !isRecord(value) ||
    typeof value.id !== 'string' ||
    typeof value.name !== 'string' ||
    typeof value.created_at !== 'string' ||
    value.paths === undefined
  ) {
    invalidResponse('Flow missing required fields');
  }
  const paths = parseFlowPathsField(value.paths);
  return {
    ...(value as Flow),
    paths,
  };
}

export function parseFlowList(value: unknown): Flow[] {
  if (!Array.isArray(value)) {
    invalidResponse('Expected flows array');
  }
  return value.map(parseFlow);
}

function parseDataFreshness(value: unknown): void {
  if (
    !isRecord(value) ||
    typeof value.as_of !== 'string' ||
    typeof value.consistency !== 'string' ||
    typeof value.stale !== 'boolean'
  ) {
    invalidResponse('Report freshness missing as_of, consistency, or stale');
  }
}

function parseFilterRejectRow(value: unknown): void {
  if (
    !isRecord(value) ||
    typeof value.reject_kind !== 'string' ||
    typeof value.reject_count !== 'number'
  ) {
    invalidResponse('Filter reject row missing reject_kind or reject_count');
  }
}

export function parseFraudCatalogReportResponse<K extends FraudCatalogReportKey>(
  key: K,
  value: unknown
): FraudCatalogReportResponseMap[K] {
  if (!isRecord(value) || !Array.isArray(value.rows)) {
    invalidResponse(`Fraud catalog report ${key} must include rows array`);
  }
  for (const row of value.rows) {
    if (!isRecord(row)) {
      invalidResponse(`Fraud catalog report ${key} row must be an object`);
    }
    if (key === 'filter-rejects') {
      parseFilterRejectRow(row);
    }
  }
  parseDataFreshness(value.freshness);
  if (key === 'layer-desync-drilldown' && value.series !== undefined) {
    if (!Array.isArray(value.series)) {
      invalidResponse('Layer desync drilldown series must be an array');
    }
    for (const point of value.series) {
      if (!isRecord(point)) {
        invalidResponse('Layer desync drilldown series point must be an object');
      }
    }
  }
  return value as FraudCatalogReportResponseMap[K];
}

function parseFraudBreakdownRow(value: unknown): FraudBreakdownRow {
  if (!isRecord(value)) {
    invalidResponse('Fraud breakdown row must be an object');
  }
  return value as FraudBreakdownRow;
}

function parseWireSignalBreakdownRow(value: unknown): WireSignalBreakdownRow {
  if (!isRecord(value)) {
    invalidResponse('Wire signal breakdown row must be an object');
  }
  return value as WireSignalBreakdownRow;
}

export function parseFraudBreakdownReportResponse(value: unknown): FraudBreakdownReportResponse {
  if (!isRecord(value) || !Array.isArray(value.rows)) {
    invalidResponse('Fraud breakdown report must include rows array');
  }
  for (const row of value.rows) {
    parseFraudBreakdownRow(row);
  }
  parseDataFreshness(value.freshness);
  return value as FraudBreakdownReportResponse;
}

export function parseWireSignalBreakdownReportResponse(
  value: unknown
): WireSignalBreakdownReportResponse {
  if (!isRecord(value) || !Array.isArray(value.rows)) {
    invalidResponse('Wire signal breakdown report must include rows array');
  }
  for (const row of value.rows) {
    parseWireSignalBreakdownRow(row);
  }
  parseDataFreshness(value.freshness);
  return value as WireSignalBreakdownReportResponse;
}

export function parseCampaignListResponse(value: unknown): CampaignListResponse {
  if (!isRecord(value) || !Array.isArray(value.items)) {
    invalidResponse('Campaign list response must include items array');
  }
  if (
    typeof value.total !== 'number' ||
    typeof value.limit !== 'number' ||
    typeof value.offset !== 'number'
  ) {
    invalidResponse('Campaign list response missing pagination fields');
  }
  for (const item of value.items) {
    parseCampaign(item);
  }
  return value as CampaignListResponse;
}

export function parseLander(value: unknown): Lander {
  return parseLanderRow(value);
}

function parseLanderRow(value: unknown): Lander {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.name !== 'string') {
    invalidResponse('Lander row missing id or name');
  }
  return value as Lander;
}

function parseLanderHostingCounts(value: unknown): LanderHostingCounts {
  if (
    !isRecord(value) ||
    typeof value.total !== 'number' ||
    typeof value.external !== 'number' ||
    typeof value.hosted !== 'number' ||
    typeof value.unconfigured !== 'number'
  ) {
    invalidResponse('Lander list response missing hosting_counts');
  }
  return value as LanderHostingCounts;
}

export function parseLanderListResponse(value: unknown): LanderListResponse {
  if (!isRecord(value) || !Array.isArray(value.items)) {
    invalidResponse('Lander list response must include items array');
  }
  if (
    typeof value.total !== 'number' ||
    typeof value.limit !== 'number' ||
    typeof value.offset !== 'number'
  ) {
    invalidResponse('Lander list response missing pagination fields');
  }
  parseLanderHostingCounts(value.hosting_counts);
  for (const item of value.items) {
    parseLanderRow(item);
  }
  return value as LanderListResponse;
}

function parseOfferRow(value: unknown): Offer {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.name !== 'string') {
    invalidResponse('Offer row missing id or name');
  }
  return value as Offer;
}

export function parseOfferList(value: unknown): Offer[] {
  if (!Array.isArray(value)) {
    invalidResponse('Expected offers array');
  }
  return value.map(parseOfferRow);
}

export function parseCampaignBulkActionResultRow(value: unknown): CampaignBulkActionResultRow {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.ok !== 'boolean') {
    invalidResponse('Campaign bulk action result row missing id or ok');
  }
  return value as CampaignBulkActionResultRow;
}

export function parseCampaignBulkActionResponse(value: unknown): CampaignBulkActionResponse {
  if (!isRecord(value) || !Array.isArray(value.results)) {
    invalidResponse('Campaign bulk action response must include results array');
  }
  for (const row of value.results) {
    parseCampaignBulkActionResultRow(row);
  }
  return value as CampaignBulkActionResponse;
}

export function parseCampaignBulkPatchResponse(value: unknown): CampaignBulkActionResponse {
  return parseCampaignBulkActionResponse(value);
}

export function parseCampaignPublishBlockedError(value: unknown): CampaignPublishBlockedError {
  if (!isRecord(value) || !isRecord(value.field_errors)) {
    invalidResponse('Publish blocked error missing field_errors');
  }
  return value as CampaignPublishBlockedError;
}

export function parseCampaignFlowValidateResponse(value: unknown): CampaignFlowValidateResponse {
  if (!isRecord(value) || typeof value.valid !== 'boolean') {
    invalidResponse('Flow validate response missing valid boolean');
  }
  return value as CampaignFlowValidateResponse;
}

export function parseCampaignValidateResponse(value: unknown): CampaignValidateResponse {
  if (!isRecord(value) || typeof value.valid !== 'boolean') {
    invalidResponse('Campaign validate response missing valid boolean');
  }
  return value as CampaignValidateResponse;
}

export function isCampaignWizardCommitResult(value: unknown): value is CampaignWizardCommitResult {
  if (!isRecord(value) || !isRecord(value.campaign)) {
    return false;
  }
  return typeof value.campaign.id === 'string' && typeof value.campaign.name === 'string';
}

export function isCampaignWizardSession(value: unknown): value is CampaignWizardSession {
  if (!isRecord(value) || typeof value.session_id !== 'string') {
    return false;
  }
  return (
    typeof value.current_step === 'string' &&
    Array.isArray(value.completed_steps) &&
    typeof value.ready_to_commit === 'boolean'
  );
}

export function parseCampaignWizardSession(value: unknown): CampaignWizardSession {
  if (!isCampaignWizardSession(value)) {
    invalidResponse('Wizard session missing session_id or step fields');
  }
  return value;
}

export function parseCampaignWizardCommitResult(value: unknown): CampaignWizardCommitResult {
  if (!isCampaignWizardCommitResult(value)) {
    invalidResponse('Wizard commit result missing campaign id or name');
  }
  return value;
}

export function parsePostbackDryRunResult(value: unknown): PostbackDryRunResult {
  if (
    !isRecord(value) ||
    typeof value.ok !== 'boolean' ||
    typeof value.provider !== 'string' ||
    typeof value.test_event !== 'boolean'
  ) {
    invalidResponse('Postback dry-run result missing ok, provider, or test_event');
  }
  return value as PostbackDryRunResult;
}

function parseTrafficOptimizerArm(value: unknown): TrafficOptimizerDryRunResult['arms'][number] {
  if (
    !isRecord(value) ||
    typeof value.entity_id !== 'string' ||
    typeof value.current_weight !== 'number' ||
    typeof value.proposed_weight !== 'number' ||
    typeof value.observed_value !== 'number'
  ) {
    invalidResponse('Traffic optimizer dry-run arm missing entity_id or weights');
  }
  return {
    entity_id: value.entity_id,
    current_weight: value.current_weight,
    proposed_weight: value.proposed_weight,
    observed_value: value.observed_value,
  };
}

export function parseTrafficOptimizerDryRunResult(value: unknown): TrafficOptimizerDryRunResult {
  if (!isRecord(value) || typeof value.stale_weights !== 'boolean' || !Array.isArray(value.arms)) {
    invalidResponse('Traffic optimizer dry-run result missing stale_weights or arms');
  }
  return {
    stale_weights: value.stale_weights,
    arms: value.arms.map(parseTrafficOptimizerArm),
  };
}

export function parseTrafficOptimizerPreset(value: unknown): TrafficOptimizerPreset {
  if (
    !isRecord(value) ||
    typeof value.key !== 'string' ||
    typeof value.title !== 'string' ||
    typeof value.description !== 'string' ||
    !Array.isArray(value.parameters_schema)
  ) {
    invalidResponse('Traffic optimizer preset missing key, title, or parameters_schema');
  }
  return value as TrafficOptimizerPreset;
}

export function parseTrafficOptimizerPresetList(value: unknown): TrafficOptimizerPreset[] {
  if (!Array.isArray(value)) {
    invalidResponse('Traffic optimizer presets must be an array');
  }
  return value.map(parseTrafficOptimizerPreset);
}

export function parseTrafficOptimizerRule(value: unknown): TrafficOptimizerRule {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.customer_id !== 'string') {
    invalidResponse('Traffic optimizer rule missing id or customer_id');
  }
  if (typeof value.name !== 'string' || typeof value.scope !== 'string') {
    invalidResponse('Traffic optimizer rule missing name or scope');
  }
  return value as TrafficOptimizerRule;
}

export function parseTrafficOptimizerRuleList(value: unknown): TrafficOptimizerRule[] {
  if (!Array.isArray(value)) {
    invalidResponse('Traffic optimizer rules must be an array');
  }
  return value.map(parseTrafficOptimizerRule);
}

export function parseMarginGuardPolicy(value: unknown): MarginGuardPolicy {
  if (!isRecord(value) || typeof value.campaign_id !== 'string' || typeof value.name !== 'string') {
    invalidResponse('Margin guard policy missing campaign_id or name');
  }
  return value as MarginGuardPolicy;
}

export function parseMarginGuardPolicyList(value: unknown): MarginGuardPolicy[] {
  if (!Array.isArray(value)) {
    invalidResponse('Margin guard policies must be an array');
  }
  return value.map(parseMarginGuardPolicy);
}

export function parseMarginGuardActivity(value: unknown): MarginGuardActivity {
  if (
    !isRecord(value) ||
    typeof value.id !== 'string' ||
    typeof value.campaign_id !== 'string' ||
    typeof value.placement_id !== 'string'
  ) {
    invalidResponse('Margin guard activity missing id, campaign_id, or placement_id');
  }
  return value as MarginGuardActivity;
}

export function parseMarginGuardActivityList(value: unknown): MarginGuardActivity[] {
  if (!Array.isArray(value)) {
    invalidResponse('Margin guard activity must be an array');
  }
  return value.map(parseMarginGuardActivity);
}

export function parseHostedEditorState(value: unknown): HostedEditorState {
  if (
    !isRecord(value) ||
    typeof value.lander_id !== 'string' ||
    typeof value.name !== 'string' ||
    !Array.isArray(value.files)
  ) {
    invalidResponse('Hosted editor state missing lander_id, name, or files');
  }
  return value as HostedEditorState;
}

export function parseHostedEditorFileBody(value: unknown): HostedEditorFileBody {
  if (!isRecord(value) || typeof value.content !== 'string') {
    invalidResponse('Hosted editor file body missing content');
  }
  return value as HostedEditorFileBody;
}

export function parseAutomationRule(value: unknown): AutomationRule {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.name !== 'string') {
    invalidResponse('Automation rule missing id or name');
  }
  return value as AutomationRule;
}

export function parseHostedEditorSaveResult(value: unknown): HostedEditorSaveResult {
  if (
    !isRecord(value) ||
    typeof value.draft_version !== 'number' ||
    typeof value.has_unpublished_draft !== 'boolean'
  ) {
    invalidResponse('Hosted editor save result missing draft_version');
  }
  return value as HostedEditorSaveResult;
}
