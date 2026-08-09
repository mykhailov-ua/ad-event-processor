import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

/**
 * List margin guard policies for a campaign.
 *
 * @param {string} campaignId
 * @returns {Promise<object[]>}
 */
export async function fetchMarginGuardPolicies(campaignId) {
  const res = await api(`/api/v1/margin-guard/policies?campaign_id=${encodeURIComponent(campaignId)}`);
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Create a margin guard policy.
 *
 * @param {object} policy
 * @returns {Promise<object>}
 */
export async function createMarginGuardPolicy(policy) {
  const res = await apiConfirmed('/api/v1/margin-guard/policies', {
    method: 'POST',
    body: JSON.stringify(policy),
  });
  return res.data;
}

/**
 * List margin guard activity rows for a campaign.
 *
 * @param {string} campaignId
 * @returns {Promise<object[]>}
 */
export async function fetchMarginGuardActivity(campaignId) {
  const res = await api(`/api/v1/margin-guard/activity?campaign_id=${encodeURIComponent(campaignId)}`);
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Remove a placement pause override (resume placement).
 *
 * @param {string} campaignId
 * @param {string} placementId
 * @returns {Promise<void>}
 */
export async function removeMarginGuardOverride(campaignId, placementId) {
  await apiConfirmed('/api/v1/margin-guard/overrides', {
    method: 'POST',
    body: JSON.stringify({
      campaign_id: campaignId,
      placement_id: placementId,
    }),
  });
}
