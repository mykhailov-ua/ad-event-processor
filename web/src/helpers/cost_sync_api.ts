import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type CostSyncNetwork = {
  id: string;
  label: string;
};

export async function fetchCostSyncCredentials(customerId = ''): Promise<unknown[]> {
  const qs = customerId ? `?customer_id=${encodeURIComponent(customerId)}` : '';
  const { data } = await api(`/api/v1/cost-sync/credentials${qs}`);
  return (data as unknown[] | null | undefined) ?? [];
}

export async function upsertCostSyncCredential(
  network: string,
  body: Record<string, unknown>
): Promise<unknown> {
  const res = await apiConfirmed(`/api/v1/cost-sync/credentials/${encodeURIComponent(network)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  return res.data;
}

export async function deleteCostSyncCredential(network: string, customerId: string): Promise<void> {
  await apiConfirmed(
    `/api/v1/cost-sync/credentials/${encodeURIComponent(network)}?customer_id=${encodeURIComponent(customerId)}`,
    { method: 'DELETE' }
  );
}

export async function runCostSync(body: Record<string, unknown>): Promise<unknown> {
  const res = await apiConfirmed('/api/v1/cost-sync/run', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data;
}

export async function fetchCostSyncHistory(customerId = ''): Promise<unknown[]> {
  const qs = customerId ? `?customer_id=${encodeURIComponent(customerId)}` : '';
  const { data } = await api(`/api/v1/cost-sync/history${qs}`);
  return (data as unknown[] | null | undefined) ?? [];
}

export const COST_SYNC_NETWORKS: CostSyncNetwork[] = [
  { id: 'facebook', label: 'Facebook' },
  { id: 'google', label: 'Google Ads' },
  { id: 'taboola', label: 'Taboola' },
  { id: 'outbrain', label: 'Outbrain' },
  { id: 'tonic_rsoc', label: 'Tonic RSOC' },
  { id: 'system1_rsoc', label: 'System1 RSOC' },
];
