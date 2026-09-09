import type { CampaignListMetrics } from '@/api/campaigns_api';
import {
  resolveCampaignFunnelCounts,
  type CampaignFunnelCounts,
} from '@/domains/campaigns/list/campaign_list_funnel';

export type CampaignListRowMetrics = {
  clicks: number;
  impressions: number;
  blocks: number;
  costMicro: number;
  profitMicro: number;
  revenueMicro: number;
  funnel: CampaignFunnelCounts;
};

/** Maps GET /campaigns/metrics batch fields; no client-side economics math. */
export function resolveCampaignListRowMetrics(
  metrics: CampaignListMetrics | undefined
): CampaignListRowMetrics {
  return {
    clicks: metrics?.clicks ?? 0,
    impressions: metrics?.impressions ?? 0,
    blocks: metrics?.blocks ?? 0,
    costMicro: metrics?.cost_micro ?? 0,
    revenueMicro: metrics?.revenue_micro ?? 0,
    profitMicro: metrics?.profit_micro ?? 0,
    funnel: resolveCampaignFunnelCounts(metrics),
  };
}
