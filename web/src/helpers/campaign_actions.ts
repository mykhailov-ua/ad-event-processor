import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { clearScope } from './idempotency.js';
import { invalidateBuyerDashboard } from './buyer_dashboard.js';

export type CampaignStatusPayload = {
  status?: string;
  [key: string]: unknown;
};

/**
 * Pause until the given number of milliseconds elapse.
 */
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Pause a campaign after operator confirmation.
 */
export async function pauseCampaign(id: string): Promise<void> {
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
 */
export async function resumeCampaign(id: string): Promise<void> {
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
 */
export async function pollCampaignStatus(
  id: string,
  expectedStatus: string,
  timeoutMs = 30000,
): Promise<CampaignStatusPayload | undefined> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const { data } = await api(`/api/v1/campaigns/${id}`);
    const payload = data as CampaignStatusPayload | null | undefined;
    if (payload?.status === expectedStatus) return payload;
    await sleep(2000);
  }
  const { data } = await api(`/api/v1/campaigns/${id}`);
  return data as CampaignStatusPayload | undefined;
}
