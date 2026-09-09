// Runtime shape guards for high-traffic list/mutation responses.
// Mismatch throws ApiError(502, INVALID_RESPONSE); callers surface via ErrorBlock.
import { ApiError } from '@/api/api_error';
import type {
  AuditLog,
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
  PostbackDryRunResult,
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

function parseFlowPathRef(
  value: unknown,
  label: string,
  idField: 'lander_id' | 'offer_id'
): void {
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

export function isCampaignWizardCommitResult(
  value: unknown
): value is CampaignWizardCommitResult {
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
