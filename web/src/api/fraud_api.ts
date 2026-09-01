import { apiJson } from './client.js';
import type {
  FraudDecision,
  FraudIntegration,
  FraudLabelsListResponse,
  FraudLabelsQuery,
  FraudManualLabelBulkRequest,
  FraudManualLabelBulkResponse,
  FraudManualLabelRequest,
  FraudOverrideRequest,
  FraudPolicyPreset,
  PatchFraudPolicyPresetRequest,
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
  signal?: AbortSignal,
): Promise<FraudIntegration[]> {
  return apiJson<FraudIntegration[]>(buildFraudIntegrationsPath(customerId), { signal });
}

export async function listFraudLabels(
  params: FraudLabelsQuery,
  signal?: AbortSignal,
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
    { signal },
  );
}

export async function upsertFraudLabel(
  customerId: string,
  body: FraudManualLabelRequest,
  signal?: AbortSignal,
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
  signal?: AbortSignal,
): Promise<FraudManualLabelBulkResponse> {
  return apiJson<FraudManualLabelBulkResponse>(
    withCustomerQuery('/api/v1/fraud/labels/bulk', customerId),
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function createFraudOverride(
  customerId: string,
  body: FraudOverrideRequest,
  signal?: AbortSignal,
): Promise<void> {
  await apiJson<void>(withCustomerQuery('/api/v1/fraud/overrides', customerId), {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function listFraudPresets(signal?: AbortSignal): Promise<FraudPolicyPreset[]> {
  return apiJson<FraudPolicyPreset[]>('/api/v1/fraud/presets', { signal });
}

export async function patchFraudPreset(
  name: string,
  body: PatchFraudPolicyPresetRequest,
  signal?: AbortSignal,
): Promise<FraudPolicyPreset> {
  return apiJson<FraudPolicyPreset>(
    `/api/v1/ops/fraud/presets/${encodeURIComponent(name)}`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export function buildFraudDecisionPath(params: {
  customer_id: string;
  ip_hash: string;
  campaign_id?: string;
  hours?: number;
}): string {
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
  params: {
    customer_id: string;
    ip_hash: string;
    campaign_id?: string;
    hours?: number;
  },
  signal?: AbortSignal,
): Promise<FraudDecision> {
  return apiJson(buildFraudDecisionPath(params), { signal });
}
