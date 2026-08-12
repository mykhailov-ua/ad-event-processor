import { api } from './api_client.js';

export type CampaignOption = {
  id: string;
  name: string;
};

/**
 * Load campaign options for a report picker.
 */
export async function fetchCampaignOptions(customerId: string): Promise<CampaignOption[]> {
  if (!customerId) return [];
  const params = new URLSearchParams({ customer_id: customerId, limit: '100', offset: '0' });
  const res = await api(`/api/v1/campaigns?${params.toString()}`);
  const payload = res.data as { items?: unknown[] } | null | undefined;
  const items = payload?.items ?? [];
  if (!Array.isArray(items)) return [];
  return items.map((raw) => {
    const c = raw as { id?: string; campaign_id?: string; name?: string };
    return {
      id: c.id ?? c.campaign_id ?? '',
      name: c.name ?? c.id ?? 'Campaign',
    };
  }).filter((c) => c.id);
}
