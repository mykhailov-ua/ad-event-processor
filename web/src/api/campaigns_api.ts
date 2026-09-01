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
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
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

export type CampaignStatusTotals = {
  active: number;
  paused: number;
  archived: number;
  total: number;
};

export async function fetchCampaignStatusTotals(
  base: Pick<CampaignListQuery, 'customer_id' | 'q'>,
  signal?: AbortSignal,
): Promise<CampaignStatusTotals> {
  const common = { ...base, limit: 1, offset: 0 };
  const [active, paused, archived, all] = await Promise.all([
    listCampaigns({ ...common, status: 'ACTIVE' }, signal),
    listCampaigns({ ...common, status: 'PAUSED' }, signal),
    listCampaigns({ ...common, status: 'ARCHIVED' }, signal),
    listCampaigns(common, signal),
  ]);

  return {
    active: active.total,
    paused: paused.total,
    archived: archived.total,
    total: all.total,
  };
}

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

export type CampaignBulkAction = 'pause' | 'resume';

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
  return apiJson<CampaignBulkActionResponse>('/api/v1/campaigns/bulk-action', {
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

export type CampaignListMetrics = {
  impressions?: number;
  clicks?: number;
  conversions?: number;
  stale?: boolean;
};

export async function fetchCampaignListMetrics(
  campaignIds: string[],
  signal?: AbortSignal,
): Promise<Record<string, CampaignListMetrics>> {
  if (campaignIds.length === 0) {
    return {};
  }

  const settled = await Promise.allSettled(
    campaignIds.map(async (campaignId) => {
      const stats = await getCampaignStats(campaignId, {}, signal);
      return {
        campaignId,
        metrics: {
          impressions: stats.metrics?.impressions,
          clicks: stats.metrics?.clicks,
          conversions: stats.metrics?.conversions,
          stale: stats.stale,
        } satisfies CampaignListMetrics,
      };
    }),
  );

  const metricsById: Record<string, CampaignListMetrics> = {};
  for (const result of settled) {
    if (result.status !== 'fulfilled') {
      continue;
    }
    metricsById[result.value.campaignId] = result.value.metrics;
  }
  return metricsById;
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
