import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { clearScope } from './idempotency.js';
import { invalidateBuyerDashboard } from './buyer_dashboard.js';

/**
 * Pause until the given number of milliseconds elapse.
 *
 * @param {number} ms
 * @returns {Promise<void>}
 */
export function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Pause a campaign after operator confirmation.
 *
 * @param {string} id
 * @returns {Promise<void>}
 */
export async function pauseCampaign(id) {
  const scope = `campaign-pause:${id}`;
  await apiConfirmed(`/api/v1/selfserve/campaigns/${id}/pause`, {
    method: 'POST',
    body: JSON.stringify({}),
    idempotencyScope: scope,
  });
  clearScope(scope);
  invalidateBuyerDashboard();
}

/**
 * Resume a paused campaign after operator confirmation.
 *
 * @param {string} id
 * @returns {Promise<void>}
 */
export async function resumeCampaign(id) {
  const scope = `campaign-resume:${id}`;
  await apiConfirmed(`/api/v1/selfserve/campaigns/${id}/resume`, {
    method: 'POST',
    body: JSON.stringify({}),
    idempotencyScope: scope,
  });
  clearScope(scope);
  invalidateBuyerDashboard();
}

/**
 * Poll campaign status until it matches expectedStatus or timeout.
 *
 * @param {string} id
 * @param {string} expectedStatus
 * @param {number} [timeoutMs]
 * @returns {Promise<object|undefined>} latest campaign payload
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
