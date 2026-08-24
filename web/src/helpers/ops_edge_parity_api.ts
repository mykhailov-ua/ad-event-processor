import { api } from './api_client.js';

export type EdgeParityReport = {
  from: string;
  to: string;
  edge_ingress: number;
  tracker_events: number;
  divergence_pct: number;
  alert: boolean;
  blacklist_stale: number;
  edge_blocked_total: number;
  shard_mismatch_hint?: string;
};

/** Load edge ingress vs tracker event parity for the default 15-minute window. */
export async function fetchEdgeParityReport(): Promise<EdgeParityReport | null> {
  const res = await api<EdgeParityReport>('/api/v1/reports/edge-parity');
  return res.data ?? null;
}
