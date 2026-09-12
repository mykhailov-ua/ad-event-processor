import { apiJson, apiJsonArray } from './client.js';
import type {
  CrowdWaveSummary,
  FraudDecision,
  FraudDecisionQuery,
  FraudIntegration,
  FraudLabelsListResponse,
  FraudLabelsQuery,
  FraudManualLabelBulkRequest,
  FraudManualLabelBulkResponse,
  FraudManualLabelRequest,
  FraudOverrideRequest,
  FraudPolicyPreset,
  PatchFraudPolicyPresetRequest,
  ProbeClusterSummary,
} from './types.js';

function withCustomerQuery(path: string, customerId: string, extra?: URLSearchParams): string {
  const search = extra ?? new URLSearchParams();
  search.set('customer_id', customerId);
  return `${path}?${search.toString()}`;
}

export function buildFraudIntegrationsPath(customerId: string): string {
  return withCustomerQuery('/api/v1/fraud/integrations', customerId);
}

export async function listFraudIntegrations(
  customerId: string,
  signal?: AbortSignal
): Promise<FraudIntegration[]> {
  return apiJsonArray<FraudIntegration>(buildFraudIntegrationsPath(customerId), { signal });
}

export async function listFraudLabels(
  params: FraudLabelsQuery,
  signal?: AbortSignal
): Promise<FraudLabelsListResponse> {
  const search = new URLSearchParams();
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  return apiJson<FraudLabelsListResponse>(
    withCustomerQuery('/api/v1/fraud/labels', params.customer_id, search),
    { signal }
  );
}

export async function upsertFraudLabel(
  customerId: string,
  body: FraudManualLabelRequest,
  signal?: AbortSignal
): Promise<void> {
  await apiJson<void>(withCustomerQuery('/api/v1/fraud/labels', customerId), {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function bulkUpsertFraudLabels(
  customerId: string,
  body: FraudManualLabelBulkRequest,
  signal?: AbortSignal
): Promise<FraudManualLabelBulkResponse> {
  return apiJson<FraudManualLabelBulkResponse>(
    withCustomerQuery('/api/v1/fraud/labels/bulk', customerId),
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    }
  );
}

export async function createFraudOverride(
  customerId: string,
  body: FraudOverrideRequest,
  signal?: AbortSignal
): Promise<void> {
  await apiJson<void>(withCustomerQuery('/api/v1/fraud/overrides', customerId), {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function listFraudPresets(signal?: AbortSignal): Promise<FraudPolicyPreset[]> {
  return apiJsonArray<FraudPolicyPreset>('/api/v1/fraud/presets', { signal });
}

export async function patchFraudPreset(
  name: string,
  body: PatchFraudPolicyPresetRequest,
  signal?: AbortSignal
): Promise<FraudPolicyPreset> {
  return apiJson<FraudPolicyPreset>(`/api/v1/ops/fraud/presets/${encodeURIComponent(name)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}

export async function listModeratorCorpus(
  params: { limit?: number; offset?: number },
  signal?: AbortSignal
): Promise<import('./types.js').ModeratorCorpusListResponse> {
  const search = new URLSearchParams();
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  const qs = search.toString();
  const path = qs ? `/api/v1/fraud/moderator-corpus?${qs}` : '/api/v1/fraud/moderator-corpus';
  return apiJson(path, { signal });
}

export async function upsertModeratorCorpus(
  body: import('./types.js').ModeratorCorpusUpsertRequest,
  signal?: AbortSignal
): Promise<import('./types.js').ModeratorCorpusTuple> {
  return apiJson('/api/v1/fraud/moderator-corpus', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function importModeratorCorpus(
  body: import('./types.js').ModeratorCorpusImportRequest,
  signal?: AbortSignal
): Promise<import('./types.js').ModeratorCorpusImportResponse> {
  return apiJson('/api/v1/fraud/moderator-corpus/import', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function previewModeratorCorpus(
  ja3: string,
  signal?: AbortSignal
): Promise<import('./types.js').ModeratorCorpusPreviewResponse> {
  const search = new URLSearchParams({ ja3 });
  return apiJson(`/api/v1/fraud/moderator-corpus/preview?${search.toString()}`, { signal });
}

export function buildFraudDecisionPath(params: FraudDecisionQuery): string {
  const search = new URLSearchParams({
    customer_id: params.customer_id,
    ip_hash: params.ip_hash,
  });
  if (params.campaign_id) {
    search.set('campaign_id', params.campaign_id);
  }
  if (params.hours != null) {
    search.set('hours', String(params.hours));
  }
  return `/api/v1/fraud/decisions?${search.toString()}`;
}

export async function getFraudDecision(
  params: FraudDecisionQuery,
  signal?: AbortSignal
): Promise<FraudDecision> {
  return apiJson(buildFraudDecisionPath(params), { signal });
}

export async function getProbeClusterSummary(
  clusterId: string,
  signal?: AbortSignal
): Promise<ProbeClusterSummary> {
  const id = clusterId.trim();
  return apiJson<ProbeClusterSummary>(`/api/v1/fraud/probe-clusters/${encodeURIComponent(id)}`, {
    signal,
  });
}

export async function getCrowdWaveSummary(
  campaignId: string,
  signal?: AbortSignal
): Promise<CrowdWaveSummary> {
  return apiJson<CrowdWaveSummary>(`/api/v1/fraud/crowd-waves/${encodeURIComponent(campaignId)}`, {
    signal,
  });
}
