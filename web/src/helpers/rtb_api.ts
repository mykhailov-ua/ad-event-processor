import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export async function fetchRtbIntegrationProfile(): Promise<unknown> {
  const { data } = await api('/api/v1/rtb/integration-profile');
  return data;
}

export async function fetchRtbShadowDiff(window = '1h'): Promise<unknown> {
  const params = new URLSearchParams({ window });
  const { data } = await api(`/api/v1/rtb/shadow-diff?${params.toString()}`);
  return data;
}

export async function validateBidRequest(body: Record<string, unknown>): Promise<unknown> {
  const { data } = await api('/api/v1/rtb/validate-bid-request', {
    method: 'POST',
    body: JSON.stringify(body),
    headers: { 'Content-Type': 'application/json' },
  });
  return data;
}

export async function fetchRtbDeals(): Promise<unknown[]> {
  const { data } = await api('/api/v1/rtb/deals');
  return Array.isArray(data) ? data : [];
}

export async function createRtbDeal(spec: Record<string, unknown>): Promise<unknown> {
  const { data } = await apiConfirmed('/api/v1/rtb/deals', {
    method: 'POST',
    body: JSON.stringify(spec),
  });
  return data;
}

export async function deleteRtbDeal(id: number): Promise<void> {
  await apiConfirmed(`/api/v1/rtb/deals/${id}`, { method: 'DELETE' });
}

export async function patchRtbDeal(id: number, spec: Record<string, unknown>): Promise<unknown> {
  const { data } = await apiConfirmed(`/api/v1/rtb/deals/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(spec),
  });
  return data;
}

export async function applyRtbFloors(
  dryRun: boolean,
  placementIds: string[] = []
): Promise<unknown> {
  const qs = dryRun ? '?dry_run=true' : '';
  const path = `/api/v1/rtb/floors/apply${qs}`;
  const init = {
    method: 'POST',
    body: JSON.stringify({ placement_ids: placementIds }),
  };
  const { data } = dryRun ? await api(path, init) : await apiConfirmed(path, init);
  return data;
}

export async function fetchRtbReconcileExport(window = '24h'): Promise<unknown> {
  const params = new URLSearchParams({ window });
  const { data } = await api(`/api/v1/rtb/reconcile/export?${params.toString()}`);
  return data;
}
