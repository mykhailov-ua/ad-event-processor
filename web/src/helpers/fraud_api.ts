import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { can } from './permissions.js';

export type MLManualLabelDTO = {
  ip_hash: string;
  label: number;
  reason: string;
  source: string;
  created_at: string;
};

export type FraudManualLabelRow = {
  ip_hash: string;
  label: number;
  reason: string;
};

export type FraudManualLabelRequest = {
  ip_hash: string;
  label: number;
  reason: string;
};

export type FraudManualLabelBulkRequest = {
  rows: FraudManualLabelRow[];
};

export type FraudManualLabelBulkResponse = {
  upserted: number;
};

export type FraudTierThresholdsDTO = {
  scope?: string;
  pass_max: number;
  suspect_max: number;
  ivt_max: number;
  block_above: number;
};

export type FraudDecisionDTO = {
  ip_hash: string;
  campaign_id: string;
  window_start: string;
  evaluated_at: string;
  disclaimer: string;
  tier: string;
  score: number;
  ml_probability: number;
  adjusted_probability: number;
  residential_proxy: boolean;
  structural_fraud: boolean;
  fp_guard_applied: boolean;
  model_score?: number;
  model_name?: string;
  score_missing: boolean;
  features: Record<string, number>;
  campaign_thresholds: FraudTierThresholdsDTO;
};

export type FraudIntegrationDTO = {
  campaign_id: string;
  name: string;
  provider?: string;
  configured: boolean;
  health_status: string;
  last_success_at?: string;
  dlq_count: number;
  last_error?: string;
};

export type FraudOverrideRequest = {
  campaign_id?: string;
  ip?: string;
  ip_hash?: string;
};

export type FraudPolicyPresetDTO = {
  name: string;
  pass: number;
  suspect: number;
  ivt: number;
  block: number;
  updated_at: string;
};

const IP_HASH_RE = /^[0-9a-f]{32}$/;

export function isValidFraudIPHash(value: string): boolean {
  return IP_HASH_RE.test(value.trim().toLowerCase());
}

export function canWriteFraudLabels(permissions: string[]): boolean {
  return can(permissions, 'campaigns:write') || can(permissions, 'shards:write');
}

export function canApplyFraudOverride(permissions: string[]): boolean {
  return (
    can(permissions, 'audit:write') ||
    can(permissions, 'campaigns:write') ||
    can(permissions, 'shards:write')
  );
}

function withCustomerQuery(customerId: string, extra?: Record<string, string>): string {
  const qs = new URLSearchParams({ customer_id: customerId });
  if (extra) {
    for (const [key, value] of Object.entries(extra)) {
      if (value) qs.set(key, value);
    }
  }
  return qs.toString();
}

export function buildFraudDecisionUrl(params: {
  customerId: string;
  ipHash: string;
  hours?: number;
  campaignId?: string;
}): string {
  const qs = new URLSearchParams({
    customer_id: params.customerId,
    ip_hash: params.ipHash.trim().toLowerCase(),
  });
  if (params.hours != null && params.hours > 0) qs.set('hours', String(params.hours));
  if (params.campaignId?.trim()) qs.set('campaign_id', params.campaignId.trim());
  return `/api/v1/fraud/decisions?${qs.toString()}`;
}

export async function fetchFraudDecision(
  params: {
    customerId: string;
    ipHash: string;
    hours?: number;
    campaignId?: string;
  },
  signal?: AbortSignal
): Promise<FraudDecisionDTO> {
  const result = await api<FraudDecisionDTO>(buildFraudDecisionUrl(params), { signal });
  return result.data;
}

export function buildFraudLabelsUrl(params: { customerId: string; limit?: number }): string {
  const qs = new URLSearchParams({ customer_id: params.customerId });
  if (params.limit != null && params.limit > 0) qs.set('limit', String(params.limit));
  return `/api/v1/fraud/labels?${qs.toString()}`;
}

export async function fetchFraudLabels(
  params: { customerId: string; limit?: number },
  signal?: AbortSignal
): Promise<MLManualLabelDTO[]> {
  const result = await api<MLManualLabelDTO[]>(buildFraudLabelsUrl(params), { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function postFraudLabel(
  customerId: string,
  body: FraudManualLabelRequest
): Promise<void> {
  await apiConfirmed(`/api/v1/fraud/labels?${withCustomerQuery(customerId)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      ip_hash: body.ip_hash.trim().toLowerCase(),
      label: body.label,
      reason: body.reason,
    }),
  });
}

export async function postFraudLabelsBulk(
  customerId: string,
  rows: FraudManualLabelRow[]
): Promise<FraudManualLabelBulkResponse> {
  const result = await apiConfirmed<FraudManualLabelBulkResponse>(
    `/api/v1/fraud/labels/bulk?${withCustomerQuery(customerId)}`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rows }),
    }
  );
  return result.data;
}

export async function postFraudOverride(
  customerId: string,
  body: FraudOverrideRequest
): Promise<void> {
  await apiConfirmed(`/api/v1/fraud/overrides?${withCustomerQuery(customerId)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export async function fetchFraudPresets(signal?: AbortSignal): Promise<FraudPolicyPresetDTO[]> {
  const result = await api<FraudPolicyPresetDTO[]>('/api/v1/fraud/presets', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchFraudIntegrations(
  customerId: string,
  signal?: AbortSignal
): Promise<FraudIntegrationDTO[]> {
  const result = await api<FraudIntegrationDTO[]>(
    `/api/v1/fraud/integrations?${withCustomerQuery(customerId)}`,
    { signal }
  );
  return Array.isArray(result.data) ? result.data : [];
}
