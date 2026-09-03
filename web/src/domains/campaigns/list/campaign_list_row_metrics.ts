import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignMargin } from '@/api/types';
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

export function resolveCampaignListRowMetrics(
  metrics: CampaignListMetrics | undefined,
  margin: CampaignMargin | undefined,
): CampaignListRowMetrics {
  const clicks = metrics?.clicks ?? 0;
  const impressions = metrics?.impressions ?? 0;
  const blocks = metrics?.blocks ?? 0;
  // Prefer metrics batch totals; margin fields are a fallback before the batch resolves.
  const costMicro = metrics?.cost_micro ?? margin?.rtb_cost_micro ?? 0;
  const profitMicro = metrics?.profit_micro ?? margin?.operator_margin_micro ?? 0;
  const revenueMicro =
    metrics?.revenue_micro ??
    (margin ? (margin.advertiser_spend_micro ?? 0) + (margin.operator_margin_micro ?? 0) : 0);

  return {
    clicks,
    impressions,
    blocks,
    costMicro,
    profitMicro,
    revenueMicro,
    funnel: resolveCampaignFunnelCounts(metrics),
  };
}

export function campaignListRowWithoutTraffic(row: CampaignListRowMetrics): boolean {
  return (
    row.clicks === 0 &&
    row.costMicro === 0 &&
    row.profitMicro === 0 &&
    row.revenueMicro === 0
  );
}
