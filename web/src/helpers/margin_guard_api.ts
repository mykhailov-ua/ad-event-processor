import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { to } from '../lib/to.js';
import { isParallelSlotError, parallelAll } from './request_multiplex.js';
import type { CampaignDTO, CampaignListResponse, CampaignMarginDTO } from '../types/campaign.js';
import type { components } from '../types/generated/openapi.js';

export type MarginGuardPolicy = components['schemas']['MarginGuardPolicy'];
export type MarginGuardActivity = components['schemas']['MarginGuardActivity'];
export type MarginGuardOverrideRequest = components['schemas']['MarginGuardOverrideRequest'];

export type MarginBreachRow = {
  campaign: CampaignDTO;
  margin: CampaignMarginDTO;
};

/**
 * List margin guard policies for one campaign.
 */
export async function fetchMarginGuardPolicies(campaignId: string): Promise<MarginGuardPolicy[]> {
  const res = await api<MarginGuardPolicy[]>(
    `/api/v1/margin-guard/policies?campaign_id=${encodeURIComponent(campaignId)}`
  );
  return Array.isArray(res.data) ? res.data : [];
}

/**
 * Create a margin guard policy row.
 */
export async function createMarginGuardPolicy(
  policy: Omit<MarginGuardPolicy, 'id'> & { id?: string }
): Promise<MarginGuardPolicy> {
  const res = await apiConfirmed('/api/v1/margin-guard/policies', {
    method: 'POST',
    body: JSON.stringify(policy),
  });
  return res.data as MarginGuardPolicy;
}

/**
 * List margin guard activity for one campaign.
 */
export async function fetchMarginGuardActivity(campaignId: string): Promise<MarginGuardActivity[]> {
  const res = await api(
    `/api/v1/margin-guard/activity?campaign_id=${encodeURIComponent(campaignId)}`
  );
  return Array.isArray(res.data) ? (res.data as MarginGuardActivity[]) : [];
}

/**
 * Clear a placement override for a campaign.
 */
export async function removeMarginGuardOverride(
  campaignId: string,
  placementId: string
): Promise<void> {
  const body: MarginGuardOverrideRequest = {
    campaign_id: campaignId,
    placement_id: placementId,
  };
  await apiConfirmed('/api/v1/margin-guard/overrides', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

/**
 * Fetch margin snapshot for one campaign.
 */
export async function fetchCampaignMargin(campaignId: string): Promise<CampaignMarginDTO> {
  const res = await api<CampaignMarginDTO>(`/api/v1/campaigns/${campaignId}/margin`);
  return res.data;
}

/**
 * Scan active campaigns for margin breach flag (uses campaign list + margin endpoint).
 */
export async function scanMarginBreaches(
  customerId: string,
  opts: { limit?: number; concurrency?: number } = {}
): Promise<{ rows: MarginBreachRow[]; error: Error | null }> {
  const limit = opts.limit ?? 100;
  const concurrency = opts.concurrency ?? 5;
  const [res, err] = await to(
    api<CampaignListResponse>(
      `/api/v1/campaigns?customer_id=${encodeURIComponent(customerId)}&status=ACTIVE&limit=${limit}&offset=0`
    )
  );
  if (err) return { rows: [], error: err };
  const campaigns = res?.data?.items ?? [];
  const tasks = campaigns.map((c) => async (): Promise<MarginBreachRow | null> => {
    const [marginRes, marginErr] = await to(fetchCampaignMargin(c.id));
    if (marginErr || !marginRes || !marginRes.margin_breach) return null;
    return { campaign: c, margin: marginRes };
  });
  const results = await parallelAll(tasks, concurrency);
  const rows = results.filter(
    (r): r is MarginBreachRow =>
      !isParallelSlotError(r) &&
      r != null &&
      typeof r === 'object' &&
      'campaign' in r &&
      'margin' in r
  );
  return { rows, error: null };
}
