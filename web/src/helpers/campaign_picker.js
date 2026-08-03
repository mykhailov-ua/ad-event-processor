import { api } from './api_client.js';

/**
 * Load campaign options for a report picker.
 *
 * @param {string} customerId
 * @returns {Promise<Array<{ id: string, name: string }>>}
 */
export async function fetchCampaignOptions(customerId) {
  if (!customerId) return [];
  const params = new URLSearchParams({ customer_id: customerId, limit: '100', offset: '0' });
  const res = await api(`/api/v1/campaigns?${params.toString()}`);
  const items = res?.data?.items ?? [];
  if (!Array.isArray(items)) return [];
  return items.map((c) => ({
    id: c.id ?? c.campaign_id ?? '',
    name: c.name ?? c.id ?? 'Campaign',
  })).filter((c) => c.id);
}
