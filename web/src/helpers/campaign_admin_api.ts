import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate, clearScope } from './idempotency.js';
import type { CampaignDTO, CampaignPatchBody } from '../types/campaign.js';

export type CampaignCreateResult = { id: string };

export async function createCampaign(
  customerId: string,
  body: Record<string, unknown>
): Promise<CampaignCreateResult> {
  const scope = `create-campaign:${customerId}`;
  const payload: Record<string, unknown> = { ...body, customer_id: customerId };
  if (!payload.template_id) {
    throw new Error('template_id is required for self-serve campaign create');
  }
  const res = await apiConfirmed<CampaignCreateResult>('/api/v1/selfserve/campaigns', {
    method: 'POST',
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    body: JSON.stringify(payload),
    idempotencyScope: scope,
  });
  clearScope(scope);
  return res.data;
}

export async function patchCampaign(
  campaignId: string,
  body: CampaignPatchBody
): Promise<CampaignDTO | unknown> {
  const res = await apiConfirmed<CampaignDTO>(`/api/v1/campaigns/${campaignId}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
  return res.data;
}
