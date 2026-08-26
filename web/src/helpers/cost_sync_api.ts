import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type { components } from '../types/generated/openapi.js';

/** Static network picker labels; authoritative schemas come from GET /cost-sync/networks. */
export type CostSyncNetwork = {
  id: string;
  label: string;
};

export type CostSyncExtraField = components['schemas']['CostSyncExtraField'];
export type CostSyncNetworkSchema = components['schemas']['CostSyncNetworkSchema'];
export type CostSyncTokenMapping = components['schemas']['CostSyncTokenMapping'];
export type CostSyncCredentialResponse = components['schemas']['CostSyncCredential'];
export type UpsertCostSyncCredentialRequest =
  components['schemas']['UpsertCostSyncCredentialRequest'];
export type RunCostSyncRequest = components['schemas']['RunCostSyncRequest'];
export type CostSyncRun = components['schemas']['CostSyncRun'];
export type CostSyncSyncInterval = NonNullable<
  UpsertCostSyncCredentialRequest['sync_interval_minutes']
>;

/**
 * Fetch per-network credential form schemas from the control plane.
 */
export async function fetchCostSyncNetworks(): Promise<CostSyncNetworkSchema[]> {
  const { data } = await api('/api/v1/cost-sync/networks');
  return (data as CostSyncNetworkSchema[] | null | undefined) ?? [];
}

/**
 * List stored cost sync credentials, optionally scoped to one customer.
 */
export async function fetchCostSyncCredentials(
  customerId = ''
): Promise<CostSyncCredentialResponse[]> {
  const qs = customerId ? `?customer_id=${encodeURIComponent(customerId)}` : '';
  const { data } = await api(`/api/v1/cost-sync/credentials${qs}`);
  return (data as CostSyncCredentialResponse[] | null | undefined) ?? [];
}

/**
 * Create or update credentials for an ad network.
 */
export async function upsertCostSyncCredential(
  network: string,
  body: UpsertCostSyncCredentialRequest
): Promise<CostSyncCredentialResponse> {
  const res = await apiConfirmed(`/api/v1/cost-sync/credentials/${encodeURIComponent(network)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  return res.data as CostSyncCredentialResponse;
}

/**
 * Delete stored credentials for a customer and network.
 */
export async function deleteCostSyncCredential(network: string, customerId: string): Promise<void> {
  await apiConfirmed(
    `/api/v1/cost-sync/credentials/${encodeURIComponent(network)}?customer_id=${encodeURIComponent(customerId)}`,
    { method: 'DELETE' }
  );
}

/**
 * Trigger a manual cost sync run for a date range.
 */
export async function runCostSync(
  body: RunCostSyncRequest
): Promise<components['schemas']['RunCostSyncAcceptedResponse']> {
  const res = await apiConfirmed('/api/v1/cost-sync/run', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data as components['schemas']['RunCostSyncAcceptedResponse'];
}

/**
 * Fetch paginated cost sync run history.
 */
export async function fetchCostSyncHistory(customerId = ''): Promise<CostSyncRun[]> {
  const qs = customerId ? `?customer_id=${encodeURIComponent(customerId)}` : '';
  const { data } = await api(`/api/v1/cost-sync/history${qs}`);
  return (data as CostSyncRun[] | null | undefined) ?? [];
}

export const COST_SYNC_NETWORKS: CostSyncNetwork[] = [
  { id: 'facebook', label: 'Facebook' },
  { id: 'google', label: 'Google Ads' },
  { id: 'tiktok', label: 'TikTok Ads' },
  { id: 'microsoft_ads', label: 'Microsoft Ads' },
  { id: 'snapchat', label: 'Snapchat Ads' },
  { id: 'linkedin', label: 'LinkedIn Ads' },
  { id: 'pinterest', label: 'Pinterest Ads' },
  { id: 'trafficstars', label: 'TrafficStars' },
  { id: 'richads', label: 'RichAds' },
  { id: 'galaksion', label: 'Galaksion' },
  { id: 'propellerads', label: 'PropellerAds' },
  { id: 'mgid', label: 'MGID' },
  { id: 'adsterra', label: 'Adsterra' },
  { id: 'exoclick', label: 'ExoClick' },
  { id: 'hilltopads', label: 'HilltopAds' },
  { id: 'clickadu', label: 'Clickadu' },
  { id: 'popads', label: 'PopAds' },
  { id: 'revcontent', label: 'Revcontent' },
  { id: 'taboola', label: 'Taboola' },
  { id: 'outbrain', label: 'Outbrain' },
  { id: 'tonic_rsoc', label: 'Tonic RSOC' },
  { id: 'system1_rsoc', label: 'System1 RSOC' },
  { id: 'mondiad', label: 'Mondiad' },
  { id: 'juicyads', label: 'JuicyAds' },
  { id: 'evadav', label: 'Evadav' },
];
