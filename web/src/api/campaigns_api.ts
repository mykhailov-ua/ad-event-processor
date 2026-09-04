import { ApiError, apiFetch, apiJson } from './client.js';
import type {
  ApplyCampaignTemplatesRequest,
  ApplyCampaignTemplatesResult,
  BlockCampaignPlacementRequest,
  Campaign,
  CampaignEventListQuery,
  CampaignEventListResponse,
  CampaignFraudConfig,
  CampaignFraudPreview,
  CampaignIntegrationHealth,
  CampaignIntegrationPanel,
  CampaignListQuery,
  CampaignListResponse,
  CampaignListMetricsBatchResponse,
  CampaignListMetricsQuery,
  CampaignListMetricsRow,
  CampaignStatusTotals,
  CampaignMargin,
  CampaignOnboardingTemplate,
  CampaignPublishBlockedError,
  CampaignPublishCheck,
  CampaignSmokeResult,
  CampaignStats,
  CampaignStatsQuery,
  CampaignValidateResponse,
  CampaignWizardCommitResult,
  CampaignWizardSession,
  CampaignWizardSessionRequest,
  ConversionMappingListResponse,
  ImportMigrationResult,
  ImportValidateJobRequest,
  MigrateImportRequest,
  MigratePreviewRequest,
  MigrationPreviewResult,
  MigrationSourcesResponse,
  PatchCampaignFraudRequest,
  PatchCampaignRequest,
  PreviewCampaignFraudRequest,
  ReplaceConversionMappingsRequest,
  ReportJobStatus,
  AssignCampaignOwnerRequest,
  CampaignEditorShell,
  CampaignExportBundle,
  ImportCampaignRequest,
  ImportCampaignResult,
  MigratePullRequest,
  PlacementBlockSuggestionsResponse,
  StatusOKResponse,
} from './types.js';

export function buildCampaignsListPath(params: CampaignListQuery = {}): string {
  const search = new URLSearchParams();

  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.status) {
    search.set('status', params.status);
  }
  if (params.q) {
    search.set('q', params.q);
  }
  if (params.sort) {
    search.set('sort', params.sort);
  }
  if (params.order) {
    search.set('order', params.order);
  }
  if (params.pacing_mode) {
    search.set('pacing_mode', params.pacing_mode);
  }
  if (params.budget_min_micro != null) {
    search.set('budget_min_micro', String(params.budget_min_micro));
  }
  if (params.budget_max_micro != null) {
    search.set('budget_max_micro', String(params.budget_max_micro));
  }
  if (params.owner_user_id) {
    search.set('owner_user_id', params.owner_user_id);
  }
  if (params.country) {
    search.set('country', params.country);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }

  const query = search.toString();
  return query ? `/api/v1/campaigns?${query}` : '/api/v1/campaigns';
}

export async function listCampaigns(
  params: CampaignListQuery = {},
  signal?: AbortSignal,
): Promise<CampaignListResponse> {
  return apiJson<CampaignListResponse>(buildCampaignsListPath(params), { signal });
}

export type CampaignListFacetOwner = {
  user_id: string;
  email?: string;
};

export type CampaignListFacetsResponse = {
  countries: string[];
  owners: CampaignListFacetOwner[];
};

export async function fetchCampaignListFacets(
  customerId: string | undefined,
  signal?: AbortSignal,
): Promise<CampaignListFacetsResponse> {
  const search = new URLSearchParams();
  if (customerId) {
    search.set('customer_id', customerId);
  }
  const query = search.toString();
  const path = query ? `/api/v1/campaigns/list-facets?${query}` : '/api/v1/campaigns/list-facets';
  return apiJson<CampaignListFacetsResponse>(path, { signal });
}

export type CampaignListMetricsTotalsResponse = {
  campaign_count: number;
  flow_count: number;
  margin_breach_count: number;
  totals: CampaignListMetricsRow;
  from: string;
  to: string;
  stale: boolean;
};

export function buildCampaignListMetricsTotalsPath(
  filter: Omit<CampaignListQuery, 'limit' | 'offset' | 'sort' | 'order'> = {},
  statsQuery: Pick<CampaignListMetricsQuery, 'from' | 'to'> = {},
): string {
  const search = new URLSearchParams();
  if (filter.customer_id) {
    search.set('customer_id', filter.customer_id);
  }
  if (filter.status) {
    search.set('status', filter.status);
  }
  if (filter.q) {
    search.set('q', filter.q);
  }
  if (filter.pacing_mode) {
    search.set('pacing_mode', filter.pacing_mode);
  }
  if (filter.budget_min_micro != null) {
    search.set('budget_min_micro', String(filter.budget_min_micro));
  }
  if (filter.budget_max_micro != null) {
    search.set('budget_max_micro', String(filter.budget_max_micro));
  }
  if (filter.owner_user_id) {
    search.set('owner_user_id', filter.owner_user_id);
  }
  if (filter.country) {
    search.set('country', filter.country);
  }
  if (statsQuery.from) {
    search.set('from', statsQuery.from);
  }
  if (statsQuery.to) {
    search.set('to', statsQuery.to);
  }
  const query = search.toString();
  return query
    ? `/api/v1/campaigns/metrics-totals?${query}`
    : '/api/v1/campaigns/metrics-totals';
}

export async function fetchCampaignListMetricsTotals(
  filter: Omit<CampaignListQuery, 'limit' | 'offset' | 'sort' | 'order'>,
  statsQuery: Pick<CampaignListMetricsQuery, 'from' | 'to'> = {},
  signal?: AbortSignal,
): Promise<CampaignListMetricsTotalsResponse> {
  return apiJson<CampaignListMetricsTotalsResponse>(
    buildCampaignListMetricsTotalsPath(filter, statsQuery),
    { signal },
  );
}

export type { CampaignStatusTotals };

export async function getCampaign(id: string, signal?: AbortSignal): Promise<Campaign> {
  return apiJson<Campaign>(`/api/v1/campaigns/${encodeURIComponent(id)}`, { signal });
}

export async function patchCampaign(
  id: string,
  body: PatchCampaignRequest,
  signal?: AbortSignal,
): Promise<Campaign> {
  return apiJson<Campaign>(`/api/v1/campaigns/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}

export type CampaignBulkAction = 'pause' | 'resume' | 'archive';

export type CampaignBulkActionRequest = {
  action: CampaignBulkAction;
  campaign_ids: string[];
};

export type CampaignBulkActionResultRow = {
  id: string;
  ok: boolean;
  error_code?: string;
};

export type CampaignBulkActionResponse = {
  results: CampaignBulkActionResultRow[];
};

export const CAMPAIGN_BULK_ACTION_MAX_IDS = 50;

export function summarizeCampaignBulkResults(results: CampaignBulkActionResultRow[]): {
  succeeded: CampaignBulkActionResultRow[];
  failed: CampaignBulkActionResultRow[];
} {
  const succeeded: CampaignBulkActionResultRow[] = [];
  const failed: CampaignBulkActionResultRow[] = [];
  for (const row of results) {
    if (row.ok) {
      succeeded.push(row);
    } else {
      failed.push(row);
    }
  }
  return { succeeded, failed };
}

export async function validateCampaignPatch(
  id: string,
  body: PatchCampaignRequest,
  signal?: AbortSignal,
): Promise<CampaignValidateResponse> {
  return apiJson<CampaignValidateResponse>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/validate`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function checkCampaignPublish(
  id: string,
  signal?: AbortSignal,
): Promise<CampaignPublishCheck> {
  return apiJson<CampaignPublishCheck>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/publish-check`,
    { signal },
  );
}

export type PublishCampaignResult =
  | { status: 'published'; campaign: Campaign }
  | { status: 'blocked'; error: CampaignPublishBlockedError };

async function parseApiError(response: Response): Promise<ApiError> {
  let code = 'HTTP_ERROR';
  let message = response.statusText || `HTTP ${response.status}`;

  try {
    const body: unknown = await response.json();
    if (body && typeof body === 'object') {
      const record = body as Record<string, unknown>;
      const errorField = record.error;
      if (errorField && typeof errorField === 'object') {
        const errObj = errorField as Record<string, unknown>;
        if (typeof errObj.code === 'string') {
          code = errObj.code;
        }
        if (typeof errObj.message === 'string') {
          message = errObj.message;
        }
      } else if (typeof errorField === 'string') {
        message = errorField;
      }
    }
  } catch {
    // Non-JSON error body; keep status text.
  }

  return new ApiError(response.status, code, message);
}

export type MacroPreviewRequest = {
  sub1?: string;
  country?: string;
  click_id?: string;
  user_id?: string;
  fbclid?: string;
  gclid?: string;
  ttclid?: string;
};

export type MacroPreviewResponse = {
  resolved_click_url?: string;
  resolved_postback_url?: string;
  unresolved_macros?: string[];
  warnings?: string[];
};

export async function previewCampaignMacros(
  id: string,
  body: MacroPreviewRequest,
  signal?: AbortSignal,
): Promise<MacroPreviewResponse> {
  return apiJson<MacroPreviewResponse>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/macro-preview`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export type CloneCampaignOptions = {
  include_flow?: boolean;
  include_postbacks?: boolean;
  include_fraud?: boolean;
  include_placement_blocks?: boolean;
  reset_spend?: boolean;
};

export type CloneCampaignRequest = {
  name_prefix?: string;
  name_suffix?: string;
  options?: CloneCampaignOptions;
};

export type CloneCampaignPreview = {
  source_id: string;
  name: string;
  would_create: CloneCampaignOptions;
};

export type CloneCampaignResult = {
  id: string;
  source_id: string;
  name: string;
};

export async function previewCampaignClone(
  id: string,
  body: CloneCampaignRequest = {},
  signal?: AbortSignal,
): Promise<CloneCampaignPreview> {
  return apiJson<CloneCampaignPreview>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/clone-preview`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function cloneCampaign(
  id: string,
  body: CloneCampaignRequest = {},
  options: { idempotencyKey: string; signal?: AbortSignal },
): Promise<CloneCampaignResult> {
  return apiJson<CloneCampaignResult>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/clone`,
    {
      method: 'POST',
      headers: { 'Idempotency-Key': options.idempotencyKey },
      body: JSON.stringify(body),
      signal: options.signal,
    },
  );
}

export type CampaignDiffRow = {
  path: string;
  label: string;
  left_display: string;
  right_display: string;
  severity: string;
};

export type CampaignDiffResponse = {
  rows: CampaignDiffRow[];
  truncated?: boolean;
};

export async function getCampaignDiff(
  id: string,
  againstId: string,
  signal?: AbortSignal,
): Promise<CampaignDiffResponse> {
  const search = new URLSearchParams();
  search.set('against', againstId);
  return apiJson<CampaignDiffResponse>(
    `/api/v1/campaigns/${encodeURIComponent(id)}/diff?${search.toString()}`,
    { signal },
  );
}

export async function publishCampaign(
  id: string,
  options: { force?: boolean } = {},
  signal?: AbortSignal,
): Promise<PublishCampaignResult> {
  const search = new URLSearchParams();
  if (options.force) {
    search.set('force', 'true');
  }
  const query = search.toString();
  const path = query
    ? `/api/v1/campaigns/${encodeURIComponent(id)}/publish?${query}`
    : `/api/v1/campaigns/${encodeURIComponent(id)}/publish`;

  const response = await apiFetch(path, { method: 'POST', signal });

  if (response.ok) {
    const campaign = (await response.json()) as Campaign;
    return { status: 'published', campaign };
  }

  if (response.status === 422) {
    const error = (await response.json()) as CampaignPublishBlockedError;
    return { status: 'blocked', error };
  }

  throw await parseApiError(response);
}

export async function bulkCampaignAction(
  body: CampaignBulkActionRequest,
  signal?: AbortSignal,
): Promise<CampaignBulkActionResponse> {
  return apiJson<CampaignBulkActionResponse>('/api/v1/campaigns/bulk', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

/** @deprecated Use bulkCampaignAction */
export const bulkCampaignMutate = bulkCampaignAction;

export type CampaignGeoSummary = Record<string, unknown>;

export type CampaignFraudEditorSummary = Record<string, unknown>;

export async function getCampaignGeoSummary(
  campaignId: string,
  params: { expand?: boolean } = {},
  signal?: AbortSignal,
): Promise<CampaignGeoSummary> {
  const search = new URLSearchParams();
  if (params.expand) {
    search.set('expand', '1');
  }
  const query = search.toString();
  const base = `/api/v1/campaigns/${encodeURIComponent(campaignId)}/geo-summary`;
  const path = query ? `${base}?${query}` : base;
  return apiJson<CampaignGeoSummary>(path, { signal });
}

export async function getCampaignFraudEditorSummary(
  campaignId: string,
  params: { preview?: boolean } = {},
  signal?: AbortSignal,
): Promise<CampaignFraudEditorSummary> {
  const search = new URLSearchParams();
  if (params.preview) {
    search.set('preview', '1');
  }
  const query = search.toString();
  const base = `/api/v1/campaigns/${encodeURIComponent(campaignId)}/fraud-editor`;
  const path = query ? `${base}?${query}` : base;
  return apiJson<CampaignFraudEditorSummary>(path, { signal });
}

export async function listMigrationSources(signal?: AbortSignal): Promise<MigrationSourcesResponse> {
  return apiJson<MigrationSourcesResponse>('/api/v1/campaigns/migrate/sources', { signal });
}

export async function previewCampaignMigration(
  body: MigratePreviewRequest,
  signal?: AbortSignal,
): Promise<MigrationPreviewResult> {
  return apiJson<MigrationPreviewResult>('/api/v1/campaigns/migrate/preview', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function validateCampaignImport(
  body: MigratePreviewRequest,
  signal?: AbortSignal,
): Promise<MigrationPreviewResult> {
  return apiJson<MigrationPreviewResult>('/api/v1/campaigns/import/validate', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function createCampaignImportValidateJob(
  body: ImportValidateJobRequest,
  idempotencyKey?: string,
  signal?: AbortSignal,
): Promise<ReportJobStatus> {
  const headers: Record<string, string> = {};
  if (idempotencyKey) {
    headers['Idempotency-Key'] = idempotencyKey;
  }
  return apiJson<ReportJobStatus>('/api/v1/campaigns/import/validate/jobs', {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
    signal,
  });
}

export async function getCampaignImportValidateJob(
  jobId: string,
  signal?: AbortSignal,
): Promise<ReportJobStatus> {
  return apiJson<ReportJobStatus>(
    `/api/v1/campaigns/import/validate/jobs/${encodeURIComponent(jobId)}`,
    { signal },
  );
}

export async function importCampaignMigration(
  body: MigrateImportRequest,
  idempotencyKey: string,
  signal?: AbortSignal,
): Promise<ImportMigrationResult> {
  return apiJson<ImportMigrationResult>('/api/v1/campaigns/migrate/import', {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
    body: JSON.stringify(body),
    signal,
  });
}

export async function listCampaignOnboardingTemplates(
  signal?: AbortSignal,
): Promise<CampaignOnboardingTemplate[]> {
  return apiJson<CampaignOnboardingTemplate[]>('/api/v1/campaigns/onboarding-templates', {
    signal,
  });
}

export async function getCampaignWizardSession(
  sessionId: string,
  signal?: AbortSignal,
): Promise<CampaignWizardSession> {
  const search = new URLSearchParams({ session_id: sessionId });
  return apiJson<CampaignWizardSession>(
    `/api/v1/campaigns/wizard/session?${search.toString()}`,
    { signal },
  );
}

export async function postCampaignWizardSession(
  body: CampaignWizardSessionRequest,
  signal?: AbortSignal,
): Promise<CampaignWizardSession | CampaignWizardCommitResult> {
  return apiJson<CampaignWizardSession | CampaignWizardCommitResult>(
    '/api/v1/campaigns/wizard/session',
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function getCampaignIntegrationPanel(
  campaignId: string,
  signal?: AbortSignal,
): Promise<CampaignIntegrationPanel> {
  return apiJson<CampaignIntegrationPanel>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/integration-panel`,
    { signal },
  );
}

export async function getCampaignIntegrationHealth(
  campaignId: string,
  signal?: AbortSignal,
): Promise<CampaignIntegrationHealth> {
  return apiJson<CampaignIntegrationHealth>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/integration-health`,
    { signal },
  );
}

export async function applyCampaignTemplates(
  campaignId: string,
  body: ApplyCampaignTemplatesRequest,
  signal?: AbortSignal,
): Promise<ApplyCampaignTemplatesResult> {
  return apiJson<ApplyCampaignTemplatesResult>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/apply-templates`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function getCampaignFraud(
  campaignId: string,
  signal?: AbortSignal,
): Promise<CampaignFraudConfig> {
  return apiJson<CampaignFraudConfig>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/fraud`,
    { signal },
  );
}

export async function patchCampaignFraud(
  campaignId: string,
  body: PatchCampaignFraudRequest,
  signal?: AbortSignal,
): Promise<CampaignFraudConfig> {
  return apiJson<CampaignFraudConfig>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/fraud`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function previewCampaignFraud(
  campaignId: string,
  body: PreviewCampaignFraudRequest = {},
  signal?: AbortSignal,
): Promise<CampaignFraudPreview> {
  return apiJson<CampaignFraudPreview>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/fraud/preview`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function getCampaignStats(
  campaignId: string,
  params: CampaignStatsQuery = {},
  signal?: AbortSignal,
): Promise<CampaignStats> {
  const search = new URLSearchParams();
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  if (params.granularity) {
    search.set('granularity', params.granularity);
  }
  const query = search.toString();
  const base = `/api/v1/campaigns/${encodeURIComponent(campaignId)}/stats`;
  const path = query ? `${base}?${query}` : base;
  return apiJson<CampaignStats>(path, { signal });
}

export type CampaignListMetrics = Pick<
  CampaignListMetricsRow,
  | 'impressions'
  | 'clicks'
  | 'conversions'
  | 'unique_clicks'
  | 'blocks'
  | 'leads_raw'
  | 'hold_leads'
  | 'rejected_leads'
  | 'lp_clicks'
  | 'lp_views'
  | 'bots'
  | 'stale'
  | 'revenue_micro'
  | 'cost_micro'
  | 'profit_micro'
  | 'epc_micro'
  | 'cpc_micro'
  | 'cpa_micro'
  | 'ecpa_micro'
  | 'ctr_pct'
  | 'lp_ctr_pct'
  | 'cr_pct'
  | 'approve_rate_pct'
  | 'block_pct'
  | 'bot_pct'
  | 'roi_pct'
  | 'cpm_usd'
>;

export function buildCampaignListMetricsPath(
  campaignIds: string[],
  params: Pick<CampaignListMetricsQuery, 'from' | 'to'> = {},
): string {
  const search = new URLSearchParams();
  search.set('ids', campaignIds.join(','));
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  return `/api/v1/campaigns/metrics?${search.toString()}`;
}

function marginFromMetricsRow(row: CampaignListMetricsRow): CampaignMargin {
  return {
    campaign_id: row.campaign_id ?? '',
    window_start: new Date(0).toISOString(),
    window_hours: 0,
    advertiser_spend_micro: row.advertiser_spend_micro ?? 0,
    rtb_cost_micro: row.rtb_cost_micro ?? 0,
    operator_margin_micro: row.operator_margin_micro ?? 0,
    publisher_payout_micro: row.publisher_payout_micro ?? 0,
    cost_over_revenue_limit: 0,
    threshold_bps: 0,
    margin_breach: row.margin_breach ?? false,
  };
}

export async function fetchCampaignListMetricsBatch(
  campaignIds: string[],
  statsQuery: CampaignStatsQuery = {},
  signal?: AbortSignal,
): Promise<{
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
  stale: boolean;
}> {
  if (campaignIds.length === 0) {
    return { metricsById: {}, marginsById: {}, stale: false };
  }

  const batch = await apiJson<CampaignListMetricsBatchResponse>(
    buildCampaignListMetricsPath(campaignIds, statsQuery),
    { signal },
  );

  const metricsById: Record<string, CampaignListMetrics> = {};
  const marginsById: Record<string, CampaignMargin> = {};
  for (const [campaignId, row] of Object.entries(batch.items ?? {})) {
    metricsById[campaignId] = {
      impressions: row.impressions,
      clicks: row.clicks,
      conversions: row.conversions,
      unique_clicks: row.unique_clicks,
      blocks: row.blocks,
      leads_raw: row.leads_raw,
      hold_leads: row.hold_leads,
      rejected_leads: row.rejected_leads,
      lp_clicks: row.lp_clicks,
      lp_views: row.lp_views,
      bots: row.bots,
      stale: row.stale ?? batch.stale,
      revenue_micro: row.revenue_micro,
      cost_micro: row.cost_micro,
      profit_micro: row.profit_micro,
      epc_micro: row.epc_micro,
      cpc_micro: row.cpc_micro,
      cpa_micro: row.cpa_micro,
      ecpa_micro: row.ecpa_micro,
      ctr_pct: row.ctr_pct,
      lp_ctr_pct: row.lp_ctr_pct,
      cr_pct: row.cr_pct,
      approve_rate_pct: row.approve_rate_pct,
      block_pct: row.block_pct,
      bot_pct: row.bot_pct,
      roi_pct: row.roi_pct,
      cpm_usd: row.cpm_usd,
    };
    marginsById[campaignId] = marginFromMetricsRow({ ...row, campaign_id: campaignId });
  }

  return { metricsById, marginsById, stale: batch.stale };
}

export async function fetchCampaignListMetrics(
  campaignIds: string[],
  statsQuery: CampaignStatsQuery = {},
  signal?: AbortSignal,
): Promise<Record<string, CampaignListMetrics>> {
  const batch = await fetchCampaignListMetricsBatch(campaignIds, statsQuery, signal);
  return batch.metricsById;
}

export async function listCampaignEvents(
  campaignId: string,
  params: CampaignEventListQuery = {},
  signal?: AbortSignal,
): Promise<CampaignEventListResponse> {
  const search = new URLSearchParams();
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  const query = search.toString();
  const base = `/api/v1/campaigns/${encodeURIComponent(campaignId)}/events`;
  const path = query ? `${base}?${query}` : base;
  return apiJson<CampaignEventListResponse>(path, { signal });
}

export async function getCampaignMargin(
  campaignId: string,
  signal?: AbortSignal,
): Promise<CampaignMargin> {
  return apiJson<CampaignMargin>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/margin`,
    { signal },
  );
}

export async function fetchCampaignListMargins(
  campaignIds: string[],
  statsQuery: CampaignStatsQuery = {},
  signal?: AbortSignal,
): Promise<Record<string, CampaignMargin>> {
  const batch = await fetchCampaignListMetricsBatch(campaignIds, statsQuery, signal);
  return batch.marginsById;
}

export async function listCampaignConversionMappings(
  campaignId: string,
  signal?: AbortSignal,
): Promise<ConversionMappingListResponse> {
  return apiJson<ConversionMappingListResponse>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/conversion-mappings`,
    { signal },
  );
}

export async function replaceCampaignConversionMappings(
  campaignId: string,
  body: ReplaceConversionMappingsRequest,
  signal?: AbortSignal,
): Promise<ConversionMappingListResponse> {
  return apiJson<ConversionMappingListResponse>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/conversion-mappings`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function blockCampaignPlacement(
  campaignId: string,
  body: BlockCampaignPlacementRequest,
  signal?: AbortSignal,
): Promise<void> {
  await apiJson<unknown>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/placement-blocks`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function runCampaignSmoke(
  campaignId: string,
  signal?: AbortSignal,
): Promise<CampaignSmokeResult> {
  return apiJson<CampaignSmokeResult>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/smoke`,
    { method: 'POST', signal },
  );
}

export async function validateCampaignFlow(
  campaignId: string,
  body: Record<string, unknown> = {},
  signal?: AbortSignal,
): Promise<Record<string, unknown>> {
  return apiJson<Record<string, unknown>>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/flow/validate`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function importCampaign(
  body: ImportCampaignRequest,
  idempotencyKey: string,
  signal?: AbortSignal,
): Promise<ImportCampaignResult> {
  return apiJson<ImportCampaignResult>('/api/v1/campaigns/import', {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
    body: JSON.stringify(body),
    signal,
  });
}

export async function previewCampaignMigrationPull(
  body: MigratePullRequest,
  signal?: AbortSignal,
): Promise<MigrationPreviewResult> {
  return apiJson<MigrationPreviewResult>('/api/v1/campaigns/migrate/pull/preview', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function importCampaignMigrationPull(
  body: MigratePullRequest,
  idempotencyKey: string,
  signal?: AbortSignal,
): Promise<ImportMigrationResult> {
  return apiJson<ImportMigrationResult>('/api/v1/campaigns/migrate/pull/import', {
    method: 'POST',
    headers: { 'Idempotency-Key': idempotencyKey },
    body: JSON.stringify(body),
    signal,
  });
}

export async function putCampaignOwner(
  campaignId: string,
  body: AssignCampaignOwnerRequest,
  signal?: AbortSignal,
): Promise<StatusOKResponse> {
  return apiJson<StatusOKResponse>(`/api/v1/campaigns/${encodeURIComponent(campaignId)}/owner`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
}

export async function exportCampaign(
  campaignId: string,
  signal?: AbortSignal,
): Promise<CampaignExportBundle> {
  return apiJson<CampaignExportBundle>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/export`,
    { signal },
  );
}

export type CampaignExportBatchResponse = {
  items: Record<string, CampaignExportBundle>;
  errors?: { id: string; error_code?: string }[];
};

export async function exportCampaignsBatch(
  campaignIds: string[],
  signal?: AbortSignal,
): Promise<CampaignExportBatchResponse> {
  const search = new URLSearchParams();
  search.set('ids', campaignIds.join(','));
  return apiJson<CampaignExportBatchResponse>(`/api/v1/campaigns/export?${search.toString()}`, {
    signal,
  });
}

export async function getCampaignEditorShell(
  campaignId: string,
  signal?: AbortSignal,
): Promise<CampaignEditorShell> {
  return apiJson<CampaignEditorShell>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/editor`,
    { signal },
  );
}

export async function getPlacementBlockSuggestions(
  campaignId: string,
  params: { from?: string; to?: string } = {},
  signal?: AbortSignal,
): Promise<PlacementBlockSuggestionsResponse> {
  const search = new URLSearchParams();
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  const query = search.toString();
  const base = `/api/v1/campaigns/${encodeURIComponent(campaignId)}/placement-block-suggestions`;
  const path = query ? `${base}?${query}` : base;
  return apiJson<PlacementBlockSuggestionsResponse>(path, { signal });
}
