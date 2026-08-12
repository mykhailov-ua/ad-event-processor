import { api } from './api_client.js';
import type { ReconRunDTO } from '../types/api/ops_extra.js';

export type ReconRunsPage = {
  items: ReconRunDTO[];
  total: number;
};

/**
 * List reconciliation runs (management + payment).
 */
export async function fetchReconRuns(
  service: 'all' | 'management' | 'payment' = 'all',
  limit = 50,
  offset = 0,
): Promise<ReconRunsPage> {
  const params = new URLSearchParams({
    service,
    limit: String(limit),
    offset: String(offset),
  });
  const res = await api<ReconRunDTO[]>(`/api/v1/recon/runs?${params.toString()}`);
  const items = Array.isArray(res?.data) ? res.data : [];
  const hdr = res?.headers?.get?.('X-Total-Count');
  const total = hdr ? Number(hdr) : items.length;
  return { items, total };
}
