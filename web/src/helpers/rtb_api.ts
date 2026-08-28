import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type RtbDeal = {
  id?: number;
  deal_id?: string;
  floor_micro?: number;
  customer_id?: string;
  pacing?: string;
  seats?: number;
  updated_at?: string;
};

export type RtbDealCreateSpec = {
  deal_id: string;
  customer_id: string;
  floor_micro?: number;
  pacing?: string;
  seats?: number;
};

export type RtbDealUpdateSpec = {
  deal_id?: string;
  customer_id?: string;
  floor_micro?: number;
  pacing?: string;
  seats?: number;
};

export const RTB_ROW_WINDOW = 500;

export function windowRtbRows<T>(items: T[]): { rows: T[]; truncated: boolean } {
  if (items.length <= RTB_ROW_WINDOW) {
    return { rows: items, truncated: false };
  }
  return { rows: items.slice(0, RTB_ROW_WINDOW), truncated: true };
}

export async function fetchRtbDeals(signal?: AbortSignal): Promise<RtbDeal[]> {
  const result = await api<RtbDeal[]>('/api/v1/rtb/deals', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createRtbDeal(body: RtbDealCreateSpec): Promise<RtbDeal> {
  const result = await apiConfirmed<RtbDeal>('/api/v1/rtb/deals', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create deal failed');
  }
  return result.data;
}

export async function patchRtbDeal(id: number, body: RtbDealUpdateSpec): Promise<RtbDeal> {
  const result = await apiConfirmed<RtbDeal>(`/api/v1/rtb/deals/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('update deal failed');
  }
  return result.data;
}

export async function deleteRtbDeal(id: number): Promise<void> {
  const result = await apiConfirmed(`/api/v1/rtb/deals/${id}`, { method: 'DELETE' });
  if (result.status !== 204 && (result.status < 200 || result.status >= 300)) {
    throw new Error('delete deal failed');
  }
}

export type RtbIntegrationProfile = Record<string, unknown>;

export type RtbShadowDiff = {
  parity_rate?: number;
  mismatch_rate?: number;
  shadow_evals?: number;
  window?: string;
};

export type RtbValidationResult = {
  valid?: boolean;
  errors?: string[];
};

export type RtbDetailTab = 'profile' | 'shadow' | 'validate' | 'floors';

export const RTB_DETAIL_TABS: Array<{ id: RtbDetailTab; label: string }> = [
  { id: 'profile', label: 'Profile' },
  { id: 'shadow', label: 'Shadow diff' },
  { id: 'validate', label: 'Bid validator' },
  { id: 'floors', label: 'Floors apply' },
];

export function parseRtbDetailTab(raw: string | null): RtbDetailTab {
  const allowed: RtbDetailTab[] = ['profile', 'shadow', 'validate', 'floors'];
  return allowed.includes(raw as RtbDetailTab) ? (raw as RtbDetailTab) : 'profile';
}

export async function fetchRtbProfile(signal?: AbortSignal): Promise<RtbIntegrationProfile> {
  const result = await api<RtbIntegrationProfile>('/api/v1/rtb/integration-profile', { signal });
  return (result.data ?? {}) as RtbIntegrationProfile;
}

export async function fetchRtbShadowDiff(
  window = '24h',
  signal?: AbortSignal
): Promise<RtbShadowDiff> {
  const qs = new URLSearchParams({ window });
  const result = await api<RtbShadowDiff>(`/api/v1/rtb/shadow-diff?${qs.toString()}`, { signal });
  return result.data ?? {};
}

export async function validateRtbBidRequest(body: string): Promise<RtbValidationResult> {
  const result = await apiConfirmed<RtbValidationResult>('/api/v1/rtb/validate-bid-request', {
    method: 'POST',
    body,
    headers: { 'Content-Type': 'application/json' },
  });
  return result.data ?? {};
}

export async function applyRtbFloors(
  placementIds: string[],
  dryRun: boolean
): Promise<unknown> {
  const qs = new URLSearchParams({ dry_run: dryRun ? 'true' : 'false' });
  const result = await apiConfirmed(`/api/v1/rtb/floors/apply?${qs.toString()}`, {
    method: 'POST',
    body: JSON.stringify({ placement_ids: placementIds }),
  });
  return result.data;
}
