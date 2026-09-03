import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignMargin } from '@/api/types';
import { campaignListRevenueMicro } from '@/domains/campaigns/list/campaign_list_format';
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

/** Shared numeric row inputs for column width probe and row VM formatting. */
export function resolveCampaignListRowMetrics(
  metrics: CampaignListMetrics | undefined,
  margin: CampaignMargin | undefined,
): CampaignListRowMetrics {
  const clicks = metrics?.clicks ?? 0;
  const impressions = metrics?.impressions ?? 0;
  const blocks = metrics?.blocks ?? 0;
  const costMicro = margin?.rtb_cost_micro ?? 0;
  const profitMicro = margin?.operator_margin_micro ?? 0;
  const revenueMicro = campaignListRevenueMicro(margin);

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
