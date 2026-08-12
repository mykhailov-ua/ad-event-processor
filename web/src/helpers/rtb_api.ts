import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

/**
 * Fetch the RTB integration profile.
 */
export async function fetchRtbIntegrationProfile(): Promise<unknown> {
  const { data } = await api('/api/v1/rtb/integration-profile');
  return data;
}

/**
 * Fetch RTB shadow-diff metrics for a time window.
 */
export async function fetchRtbShadowDiff(window = '1h'): Promise<unknown> {
  const params = new URLSearchParams({ window });
  const { data } = await api(`/api/v1/rtb/shadow-diff?${params.toString()}`);
  return data;
}

/**
 * Validate an OpenRTB bid request payload.
 */
export async function validateBidRequest(body: Record<string, unknown>): Promise<unknown> {
  const { data } = await api('/api/v1/rtb/validate-bid-request', {
    method: 'POST',
    body: JSON.stringify(body),
    headers: { 'Content-Type': 'application/json' },
  });
  return data;
}

/**
 * List RTB deals.
 */
export async function fetchRtbDeals(): Promise<unknown[]> {
  const { data } = await api('/api/v1/rtb/deals');
  return Array.isArray(data) ? data : [];
}

/**
 * Create an RTB deal.
 */
export async function createRtbDeal(spec: Record<string, unknown>): Promise<unknown> {
  const { data } = await apiConfirmed('/api/v1/rtb/deals', {
    method: 'POST',
    body: JSON.stringify(spec),
  });
  return data;
}

/**
 * Delete an RTB deal by id.
 */
export async function deleteRtbDeal(id: number): Promise<void> {
  await apiConfirmed(`/api/v1/rtb/deals/${id}`, { method: 'DELETE' });
}

/**
 * Update an RTB deal.
 */
export async function patchRtbDeal(id: number, spec: Record<string, unknown>): Promise<unknown> {
  const { data } = await apiConfirmed(`/api/v1/rtb/deals/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(spec),
  });
  return data;
}

/**
 * Preview or apply RTB floor suggestions.
 */
export async function applyRtbFloors(
  dryRun: boolean,
  placementIds: string[] = [],
): Promise<unknown> {
  const qs = dryRun ? '?dry_run=true' : '';
  const { data } = await apiConfirmed(`/api/v1/rtb/floors/apply${qs}`, {
    method: 'POST',
    body: JSON.stringify({ placement_ids: placementIds }),
  });
  return data;
}
