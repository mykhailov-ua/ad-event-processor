import { apiJson } from './client.js';
import type {
  MarginGuardActivity,
  MarginGuardListActivityQuery,
  MarginGuardListPoliciesQuery,
  MarginGuardOverrideRequest,
  MarginGuardPolicy,
} from './types.js';

export async function listMarginGuardPolicies(
  params: MarginGuardListPoliciesQuery,
  signal?: AbortSignal,
): Promise<MarginGuardPolicy[]> {
  const search = new URLSearchParams();
  search.set('campaign_id', params.campaign_id);
  return apiJson<MarginGuardPolicy[]>(`/api/v1/margin-guard/policies?${search.toString()}`, {
    signal,
  });
}

export async function listMarginGuardActivity(
  params: MarginGuardListActivityQuery,
  signal?: AbortSignal,
): Promise<MarginGuardActivity[]> {
  const search = new URLSearchParams();
  search.set('campaign_id', params.campaign_id);
  return apiJson<MarginGuardActivity[]>(`/api/v1/margin-guard/activity?${search.toString()}`, {
    signal,
  });
}

export async function createMarginGuardPolicy(
  body: MarginGuardPolicy,
  signal?: AbortSignal,
): Promise<MarginGuardPolicy> {
  return apiJson<MarginGuardPolicy>('/api/v1/margin-guard/policies', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function removeMarginGuardOverride(
  body: MarginGuardOverrideRequest,
  signal?: AbortSignal,
): Promise<void> {
  await apiJson<void>('/api/v1/margin-guard/overrides', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
