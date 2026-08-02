import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

/**
 * @param {number} ms
 */
export function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * @param {string} id
 */
export async function pauseCampaign(id) {
  await apiConfirmed(`/api/v1/selfserve/campaigns/${id}/pause`, {
    method: 'POST',
    body: JSON.stringify({}),
    idempotencyScope: `campaign-pause:${id}`,
  });
}

/**
 * @param {string} id
 */
export async function resumeCampaign(id) {
  await apiConfirmed(`/api/v1/selfserve/campaigns/${id}/resume`, {
    method: 'POST',
    body: JSON.stringify({}),
    idempotencyScope: `campaign-resume:${id}`,
  });
}

/**
 * @param {string} id
 * @param {string} expectedStatus
 * @param {number} [timeoutMs]
 */
export async function pollCampaignStatus(id, expectedStatus, timeoutMs = 30000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const { data } = await api(`/api/v1/campaigns/${id}`);
    if (data?.status === expectedStatus) return data;
    await sleep(2000);
  }
  const { data } = await api(`/api/v1/campaigns/${id}`);
  return data;
}
