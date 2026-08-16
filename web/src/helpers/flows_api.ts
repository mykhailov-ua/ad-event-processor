import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type LanderDTO = {
  id: string;
  name: string;
  url: string;
  created_at?: string;
};

export type OfferDTO = {
  id: string;
  name: string;
  url: string;
  created_at?: string;
};

export type FlowPathLanderRef = {
  lander_id: string;
  weight: number;
};

export type FlowPathOfferRef = {
  offer_id: string;
  weight: number;
};

export type FlowPathDTO = {
  weight: number;
  landers: FlowPathLanderRef[];
  offers: FlowPathOfferRef[];
};

export type FlowDTO = {
  id: string;
  name: string;
  paths: FlowPathDTO[] | string;
  created_at?: string;
};

export async function fetchLanders(): Promise<LanderDTO[]> {
  const res = await api<LanderDTO[]>('/api/v1/landers');
  return Array.isArray(res.data) ? res.data : [];
}

export async function createLander(name: string, url: string): Promise<LanderDTO> {
  const res = await apiConfirmed<LanderDTO>('/api/v1/landers', {
    method: 'POST',
    body: JSON.stringify({ name, url }),
  });
  return res.data;
}

export async function fetchOffers(): Promise<OfferDTO[]> {
  const res = await api<OfferDTO[]>('/api/v1/offers');
  return Array.isArray(res.data) ? res.data : [];
}

export async function createOffer(name: string, url: string): Promise<OfferDTO> {
  const res = await apiConfirmed<OfferDTO>('/api/v1/offers', {
    method: 'POST',
    body: JSON.stringify({ name, url }),
  });
  return res.data;
}

export async function fetchFlows(): Promise<FlowDTO[]> {
  const res = await api<FlowDTO[]>('/api/v1/flows');
  return Array.isArray(res.data) ? res.data : [];
}

export async function createFlow(name: string, paths: FlowPathDTO[]): Promise<FlowDTO> {
  const res = await apiConfirmed<FlowDTO>('/api/v1/flows', {
    method: 'POST',
    body: JSON.stringify({ name, paths }),
  });
  return res.data;
}

export function parseFlowPaths(raw: FlowDTO['paths']): FlowPathDTO[] {
  if (Array.isArray(raw)) return raw;
  if (typeof raw === 'string' && raw.trim()) {
    try {
      const parsed = JSON.parse(raw) as FlowPathDTO[];
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }
  return [];
}

export function summarizeFlowPaths(paths: FlowPathDTO[]): string {
  if (!paths.length) return '—';
  const path = paths[0];
  const landerCount = path.landers?.length ?? 0;
  const offerCount = path.offers?.length ?? 0;
  const extra = paths.length > 1 ? ` (+${paths.length - 1} paths)` : '';
  return `${landerCount} lander(s), ${offerCount} offer(s)${extra}`;
}
