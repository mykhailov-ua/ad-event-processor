import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate, clearScope } from './idempotency.js';
import type { CampaignDTO, CampaignPatchBody } from '../types/campaign.js';
import type { components } from '../types/generated/openapi.js';

export type CampaignCreateResult = { id: string };

export type CloneCampaignResult = components['schemas']['CloneCampaignResult'];

export type CampaignExportBundle = {
  export_version: number;
  exported_at?: string;
  campaign: Record<string, unknown>;
  flow?: Record<string, unknown>;
  landers?: Array<{ ref: string; name: string; url?: string }>;
  offers?: Array<{ ref: string; name: string; url: string }>;
  postback_config?: Record<string, unknown>;
  conversion_mappings?: Array<Record<string, unknown>>;
  integration_schema_name?: string;
  status_integration_schema_name?: string;
};

export type ImportCampaignResult = {
  id: string;
  name: string;
};

/**
 * Downloads a campaign JSON bundle for migration or backup.
 */
export async function exportCampaign(campaignId: string): Promise<CampaignExportBundle> {
  const res = await apiConfirmed<CampaignExportBundle>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/export`
  );
  return res.data;
}

/**
 * Creates a campaign from an exported JSON bundle under the given customer.
 */
export async function importCampaign(
  customerId: string,
  bundle: CampaignExportBundle,
  options?: { name_override?: string; budget_limit_micro?: number }
): Promise<ImportCampaignResult> {
  const scope = `campaign-import:${customerId}`;
  const res = await apiConfirmed<ImportCampaignResult>('/api/v1/campaigns/import', {
    method: 'POST',
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    body: JSON.stringify({
      customer_id: customerId,
      ...bundle,
      ...options,
    }),
    idempotencyScope: scope,
  });
  clearScope(scope);
  return res.data;
}

export async function cloneCampaign(
  campaignId: string,
  body?: { name_prefix?: string; name_suffix?: string }
): Promise<CloneCampaignResult> {
  const scope = `campaign-clone:${campaignId}`;
  const res = await apiConfirmed<CloneCampaignResult>(`/api/v1/campaigns/${campaignId}/clone`, {
    method: 'POST',
    headers: { 'Idempotency-Key': getOrCreate(scope) },
    body: JSON.stringify(body ?? {}),
    idempotencyScope: scope,
  });
  clearScope(scope);
  return res.data;
}

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
