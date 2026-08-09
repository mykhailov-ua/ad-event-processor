import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

/**
 * Load postback config for a campaign.
 *
 * @param {string} campaignId
 * @returns {Promise<object|null>}
 */
export async function fetchPostbackConfig(campaignId) {
  const res = await api('/api/v1/postbacks/config');
  const rows = Array.isArray(res.data) ? res.data : [];
  return rows.find((row) => row.campaign_id === campaignId) ?? null;
}

/**
 * Save postback config for a campaign.
 *
 * @param {string} campaignId
 * @param {object} body
 * @returns {Promise<void>}
 */
export async function savePostbackConfig(campaignId, body) {
  await apiConfirmed(`/api/v1/postbacks/config/${campaignId}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

/**
 * List postback DLQ rows, optionally filtered by campaign.
 *
 * @param {string} [campaignId]
 * @returns {Promise<object[]>}
 */
export async function fetchPostbackDlq(campaignId) {
  const res = await api('/api/v1/postbacks/dlq');
  const rows = Array.isArray(res.data) ? res.data : [];
  if (!campaignId) return rows;
  return rows.filter((row) => row.campaign_id === campaignId);
}

/**
 * Retry a postback DLQ entry.
 *
 * @param {number|string} id
 * @returns {Promise<void>}
 */
export async function retryPostbackDlq(id) {
  await apiConfirmed(`/api/v1/postbacks/dlq/${id}/retry`, { method: 'POST', body: '{}' });
}
