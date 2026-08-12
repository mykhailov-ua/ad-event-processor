import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type PostbackConfigRow = {
  campaign_id?: string;
  [key: string]: unknown;
};

/**
 * Load postback config for a campaign.
 */
export async function fetchPostbackConfig(campaignId: string): Promise<PostbackConfigRow | null> {
  const res = await api('/api/v1/postbacks/config');
  const rows = Array.isArray(res.data) ? (res.data as PostbackConfigRow[]) : [];
  return rows.find((row) => row.campaign_id === campaignId) ?? null;
}

/**
 * Save postback config for a campaign.
 */
export async function savePostbackConfig(
  campaignId: string,
  body: Record<string, unknown>,
): Promise<void> {
  await apiConfirmed(`/api/v1/postbacks/config/${campaignId}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

/**
 * List postback DLQ rows, optionally filtered by campaign.
 */
export async function fetchPostbackDlq(campaignId?: string): Promise<PostbackConfigRow[]> {
  const res = await api('/api/v1/postbacks/dlq');
  const rows = Array.isArray(res.data) ? (res.data as PostbackConfigRow[]) : [];
  if (!campaignId) return rows;
  return rows.filter((row) => row.campaign_id === campaignId);
}

/**
 * Retry a postback DLQ entry.
 */
export async function retryPostbackDlq(id: number | string): Promise<void> {
  await apiConfirmed(`/api/v1/postbacks/dlq/${id}/retry`, { method: 'POST', body: '{}' });
}
