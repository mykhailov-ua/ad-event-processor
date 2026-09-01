import { apiFetch, apiJson } from './client.js';
import type {
  OpenRtbValidationResult,
  RtbDeal,
  RtbDealCreateSpec,
  RtbDealUpdateSpec,
  RtbFloorsApplyRequest,
  RtbFloorsApplyResult,
  RtbIntegrationProfile,
  RtbReconcileExport,
  RtbShadowDiffSnapshot,
} from './types.js';

export async function listRtbDeals(signal?: AbortSignal): Promise<RtbDeal[]> {
  return apiJson<RtbDeal[]>('/api/v1/rtb/deals', { signal });
}

export async function getRtbDeal(id: number, signal?: AbortSignal): Promise<RtbDeal> {
  return apiJson<RtbDeal>(`/api/v1/rtb/deals/${encodeURIComponent(String(id))}`, { signal });
}

export async function createRtbDeal(
  body: RtbDealCreateSpec,
  signal?: AbortSignal,
): Promise<RtbDeal> {
  return apiJson<RtbDeal>('/api/v1/rtb/deals', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function patchRtbDeal(
  id: number,
  body: RtbDealUpdateSpec,
  signal?: AbortSignal,
): Promise<RtbDeal> {
  return apiJson<RtbDeal>(`/api/v1/rtb/deals/${encodeURIComponent(String(id))}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteRtbDeal(id: number, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/rtb/deals/${encodeURIComponent(String(id))}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function getRtbIntegrationProfile(
  signal?: AbortSignal,
): Promise<RtbIntegrationProfile> {
  return apiJson<RtbIntegrationProfile>('/api/v1/rtb/integration-profile', { signal });
}

export async function getRtbShadowDiff(
  window = '1h',
  signal?: AbortSignal,
): Promise<RtbShadowDiffSnapshot> {
  const search = new URLSearchParams();
  if (window) {
    search.set('window', window);
  }
  const query = search.toString();
  const path = query ? `/api/v1/rtb/shadow-diff?${query}` : '/api/v1/rtb/shadow-diff';
  return apiJson<RtbShadowDiffSnapshot>(path, { signal });
}

export async function getRtbReconcileExport(
  params: { window?: string; request_id?: string } = {},
  signal?: AbortSignal,
): Promise<RtbReconcileExport> {
  const search = new URLSearchParams();
  if (params.window) {
    search.set('window', params.window);
  }
  if (params.request_id) {
    search.set('request_id', params.request_id);
  }
  const query = search.toString();
  const path = query ? `/api/v1/rtb/reconcile/export?${query}` : '/api/v1/rtb/reconcile/export';
  return apiJson<RtbReconcileExport>(path, { signal });
}

export async function applyRtbFloors(
  body: RtbFloorsApplyRequest,
  dryRun = true,
  signal?: AbortSignal,
): Promise<RtbFloorsApplyResult> {
  const search = new URLSearchParams();
  if (dryRun) {
    search.set('dry_run', 'true');
  }
  const query = search.toString();
  const path = query ? `/api/v1/rtb/floors/apply?${query}` : '/api/v1/rtb/floors/apply';
  return apiJson<RtbFloorsApplyResult>(path, {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function validateRtbBidRequest(
  body: Record<string, unknown>,
  signal?: AbortSignal,
): Promise<OpenRtbValidationResult> {
  return apiJson<OpenRtbValidationResult>('/api/v1/rtb/validate-bid-request', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
