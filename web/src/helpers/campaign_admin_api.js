import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate, clearScope } from './idempotency.js';

/**
 * Create a campaign via self-serve API.
 *
 * @param {string} customerId
 * @param {object} body
 * @returns {Promise<{ id: string }>}
 */
export async function createCampaign(customerId, body) {
  const scope = `create-campaign:${customerId}`;
  const res = await apiConfirmed('/api/v1/selfserve/campaigns', {
    method: 'POST',
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    body: JSON.stringify({ ...body, customer_id: customerId }),
    idempotencyScope: scope,
  });
  clearScope(scope);
  return res.data;
}

/**
 * Patch campaign settings (admin).
 *
 * @param {string} campaignId
 * @param {object} body
 * @returns {Promise<object>}
 */
export async function patchCampaign(campaignId, body) {
  const res = await apiConfirmed(`/api/v1/campaigns/${campaignId}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
  return res.data;
}
