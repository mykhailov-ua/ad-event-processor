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

export function metricsBatchHasMoneyFields(metrics?: CampaignListMetrics): boolean {
  if (!metrics) {
    return false;
  }
  return (
    metrics.revenue_micro != null ||
    metrics.cost_micro != null ||
    metrics.profit_micro != null ||
    metrics.roi_pct != null
  );
}

export function resolveCampaignListRowMetrics(
  metrics: CampaignListMetrics | undefined,
  margin: CampaignMargin | undefined
): CampaignListRowMetrics {
  const clicks = metrics?.clicks ?? 0;
  const impressions = metrics?.impressions ?? 0;
  const blocks = metrics?.blocks ?? 0;
  // Prefer metrics batch totals; margin fields are a fallback before the batch resolves.
  const costMicro = metrics?.cost_micro ?? margin?.rtb_cost_micro ?? 0;
  const revenueMicro =
    metrics?.revenue_micro ??
    (margin ? (margin.advertiser_spend_micro ?? 0) + (margin.operator_margin_micro ?? 0) : 0);

  let profitMicro = 0;
  if (metrics?.revenue_micro != null && metrics?.cost_micro != null) {
    profitMicro = revenueMicro - costMicro;
  } else if (metrics?.profit_micro != null && Number.isFinite(metrics.profit_micro)) {
    profitMicro = metrics.profit_micro;
  } else if (margin != null && metrics == null) {
    profitMicro = revenueMicro - costMicro;
  } else if (metricsBatchHasMoneyFields(metrics)) {
    profitMicro = revenueMicro - costMicro;
  }

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
