import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { to } from '../lib/to.js';
import { isParallelSlotError, parallelAll } from './request_multiplex.js';
import type { CampaignDTO, CampaignListResponse, CampaignMarginDTO } from '../types/campaign.js';

export type MarginGuardPolicy = {
  id?: string;
  campaign_id: string;
  name?: string;
  min_clicks?: number;
  roi_floor_pct?: number;
  zero_conv_streak?: number;
  cost_over_revenue_threshold_bps?: number;
  is_active?: boolean;
  [key: string]: unknown;
};

export type MarginBreachRow = {
  campaign: CampaignDTO;
  margin: CampaignMarginDTO;
};

export async function fetchMarginGuardPolicies(campaignId: string): Promise<MarginGuardPolicy[]> {
  const res = await api<MarginGuardPolicy[]>(
    `/api/v1/margin-guard/policies?campaign_id=${encodeURIComponent(campaignId)}`
  );
  return Array.isArray(res.data) ? res.data : [];
}

export async function createMarginGuardPolicy(policy: MarginGuardPolicy): Promise<unknown> {
  const res = await apiConfirmed('/api/v1/margin-guard/policies', {
    method: 'POST',
    body: JSON.stringify(policy),
  });
  return res.data;
}

export async function fetchMarginGuardActivity(campaignId: string): Promise<unknown[]> {
  const res = await api(
    `/api/v1/margin-guard/activity?campaign_id=${encodeURIComponent(campaignId)}`
  );
  return Array.isArray(res.data) ? res.data : [];
}

export async function removeMarginGuardOverride(
  campaignId: string,
  placementId: string
): Promise<void> {
  await apiConfirmed('/api/v1/margin-guard/overrides', {
    method: 'POST',
    body: JSON.stringify({
      campaign_id: campaignId,
      placement_id: placementId,
    }),
  });
}

export async function fetchCampaignMargin(campaignId: string): Promise<CampaignMarginDTO> {
  const res = await api<CampaignMarginDTO>(`/api/v1/campaigns/${campaignId}/margin`);
  return res.data;
}

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
