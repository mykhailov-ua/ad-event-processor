import * as auth from './auth.js';
import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type Lander = {
  id?: string;
  name?: string;
  url?: string;
  hosted_url?: string;
  created_at?: string;
};

export type Offer = {
  id?: string;
  name?: string;
  url?: string;
  created_at?: string;
};

export type FlowPathRef = {
  weight?: number;
  landers?: Array<{ lander_id?: string; weight?: number }>;
  offers?: Array<{ offer_id?: string; weight?: number }>;
};

export type Flow = {
  id?: string;
  name?: string;
  paths?: FlowPathRef[] | string;
  created_at?: string;
};

export async function fetchLanders(signal?: AbortSignal): Promise<Lander[]> {
  const result = await api<Lander[]>('/api/v1/landers', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchOffers(signal?: AbortSignal): Promise<Offer[]> {
  const result = await api<Offer[]>('/api/v1/offers', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchFlows(signal?: AbortSignal): Promise<Flow[]> {
  const result = await api<Flow[]>('/api/v1/flows', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createLander(body: { name: string; url?: string }): Promise<Lander> {
  const result = await apiConfirmed<Lander>('/api/v1/landers', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create lander failed');
  }
  return result.data;
}

export async function createOffer(body: { name: string; url: string }): Promise<Offer> {
  const result = await apiConfirmed<Offer>('/api/v1/offers', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create offer failed');
  }
  return result.data;
}

export async function createFlow(body: {
  name: string;
  paths: FlowPathRef[];
}): Promise<Flow> {
  const result = await apiConfirmed<Flow>('/api/v1/flows', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create flow failed');
  }
  return result.data;
}

export async function uploadLanderZip(landerId: string, file: File): Promise<Lander> {
  const form = new FormData();
  form.append('zip', file);
  const headers = new Headers();
  const csrf = auth.getCsrfToken();
  if (csrf) headers.set('X-CSRF-Token', csrf);

  const res = await fetch(`/api/v1/landers/${encodeURIComponent(landerId)}/hosted-upload`, {
    method: 'POST',
    headers,
    body: form,
    credentials: 'same-origin',
  });

  if (!res.ok) {
    throw new Error(res.statusText || 'upload failed');
  }

  const data = (await res.json()) as Lander;
  return data;
}

export function summarizeFlowPaths(paths: Flow['paths']): string {
  if (paths == null) return '-';
  if (typeof paths === 'string') {
    const trimmed = paths.trim();
    return trimmed.length > 80 ? `${trimmed.slice(0, 80)}...` : trimmed || '-';
  }
  if (!Array.isArray(paths) || paths.length === 0) return '0 paths';
  let landerRefs = 0;
  let offerRefs = 0;
  for (const path of paths) {
    landerRefs += path.landers?.length ?? 0;
    offerRefs += path.offers?.length ?? 0;
  }
  return `${paths.length} path(s), ${landerRefs} lander(s), ${offerRefs} offer(s)`;
}

export const DEFAULT_FLOW_PATHS: FlowPathRef[] = [{ weight: 100, landers: [], offers: [] }];

export const FLOW_ROW_WINDOW = 500;

export type FlowBuilderTab = 'graph' | 'validate' | 'catalog';

export const FLOW_BUILDER_TABS: Array<{ id: FlowBuilderTab; label: string }> = [
  { id: 'graph', label: 'Paths' },
  { id: 'catalog', label: 'Landers & offers' },
  { id: 'validate', label: 'Validate' },
];

export function parseFlowBuilderTab(raw: string | null): FlowBuilderTab {
  const allowed: FlowBuilderTab[] = ['graph', 'validate', 'catalog'];
  return allowed.includes(raw as FlowBuilderTab) ? (raw as FlowBuilderTab) : 'graph';
}

export async function fetchFlow(id: string, signal?: AbortSignal): Promise<Flow> {
  const result = await api<Flow>(`/api/v1/flows/${encodeURIComponent(id)}`, { signal });
  return result.data ?? {};
}

export async function updateFlow(
  id: string,
  body: { name: string; paths: FlowPathRef[] }
): Promise<Flow> {
  const result = await apiConfirmed<Flow>(`/api/v1/flows/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('update flow failed');
  }
  return result.data ?? {};
}

export async function validateCampaignFlow(
  campaignId: string,
  paths: FlowPathRef[]
): Promise<unknown> {
  const result = await apiConfirmed(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/flow/validate`,
    { method: 'POST', body: JSON.stringify({ paths }) }
  );
  return result.data;
}

export function windowRows<T>(items: T[]): { rows: T[]; truncated: boolean } {
  if (items.length <= FLOW_ROW_WINDOW) {
    return { rows: items, truncated: false };
  }
  return { rows: items.slice(0, FLOW_ROW_WINDOW), truncated: true };
}
