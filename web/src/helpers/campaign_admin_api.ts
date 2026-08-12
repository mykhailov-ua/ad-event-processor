import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate, clearScope } from './idempotency.js';
import type { CampaignDTO, CampaignPatchBody } from '../types/api/campaign.js';

export type CampaignCreateResult = { id: string };

/**
 * Create a campaign via self-serve API.
 */
export async function createCampaign(
  customerId: string,
  body: Record<string, unknown>,
): Promise<CampaignCreateResult> {
  const scope = `create-campaign:${customerId}`;
  const res = await apiConfirmed<CampaignCreateResult>('/api/v1/selfserve/campaigns', {
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
 */
export async function patchCampaign(
  campaignId: string,
  body: CampaignPatchBody,
): Promise<CampaignDTO | unknown> {
  const res = await apiConfirmed<CampaignDTO>(`/api/v1/campaigns/${campaignId}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
  return res.data;
}
