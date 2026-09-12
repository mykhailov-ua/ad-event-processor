import { apiFetch, apiJsonValidated, parseApiError } from './client.js';
import {
  parseMarginGuardActivityList,
  parseMarginGuardPolicy,
  parseMarginGuardPolicyList,
} from './validate.js';
import type {
  MarginGuardActivity,
  MarginGuardListActivityQuery,
  MarginGuardListPoliciesQuery,
  MarginGuardPolicy,
} from './types.js';

export async function listMarginGuardPolicies(
  params: MarginGuardListPoliciesQuery,
  signal?: AbortSignal
): Promise<MarginGuardPolicy[]> {
  const search = new URLSearchParams();
  if (params.campaign_id) {
    search.set('campaign_id', params.campaign_id);
  }
  const query = search.toString();
  const path = query ? `/api/v1/margin-guard/policies?${query}` : '/api/v1/margin-guard/policies';
  return apiJsonValidated(path, { signal }, parseMarginGuardPolicyList);
}

export async function listMarginGuardActivity(
  params: MarginGuardListActivityQuery,
  signal?: AbortSignal
): Promise<MarginGuardActivity[]> {
  const search = new URLSearchParams();
  if (params.campaign_id) {
    search.set('campaign_id', params.campaign_id);
  }
  const query = search.toString();
  const path = query ? `/api/v1/margin-guard/activity?${query}` : '/api/v1/margin-guard/activity';
  return apiJsonValidated(path, { signal }, parseMarginGuardActivityList);
}

export async function createMarginGuardPolicy(
  body: MarginGuardPolicy,
  signal?: AbortSignal
): Promise<MarginGuardPolicy> {
  return apiJsonValidated(
    '/api/v1/margin-guard/policies',
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
    parseMarginGuardPolicy
  );
}

export async function removeMarginGuardOverride(
  body: { campaign_id: string; placement_id: string },
  signal?: AbortSignal
): Promise<void> {
  const response = await apiFetch('/api/v1/margin-guard/overrides', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok) {
    throw await parseApiError(response);
  }
}
