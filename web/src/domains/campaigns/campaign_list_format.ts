import {
  formatDashboardCrPct,
  formatDashboardRoiPct,
  formatDashboardUsdFromMicro,
} from '@/domains/dashboards/dashboard_format';
import { displayCount, displayMoneyDecimal } from '@/lib/display';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignMargin } from '@/api/types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/campaign_metrics_shared';

export function formatCampaignListCr(clicks?: number, conversions?: number): string {
  if (clicks == null || clicks <= 0) {
    return conversions ? '0.00%' : '';
  }
  return formatDashboardCrPct(((conversions ?? 0) / clicks) * 100);
}

export function formatCampaignListRoi(profitMicro?: number, costMicro?: number): string {
  if (costMicro == null || costMicro <= 0 || profitMicro == null) {
    return '';
  }
  return formatDashboardRoiPct((profitMicro / costMicro) * 100);
}

export function campaignListRevenueMicro(margin?: CampaignMargin): number {
  if (!margin) {
    return 0;
  }
  return (margin.advertiser_spend_micro ?? 0) + (margin.operator_margin_micro ?? 0);
}

export function campaignListCostLabel(campaign: CampaignWithMoneyDisplay): string {
  return displayMoneyDecimal(campaign.current_spend, campaign.current_spend_display);
}

export function campaignListRevenueLabel(margin?: CampaignMargin): string {
  const revenueMicro = campaignListRevenueMicro(margin);
  if (revenueMicro === 0) {
    return '';
  }
  return formatDashboardUsdFromMicro(revenueMicro);
}

export function campaignListProfitLabel(margin?: CampaignMargin): string {
  if (!margin?.operator_margin_micro) {
    return '';
  }
  return formatDashboardUsdFromMicro(margin.operator_margin_micro);
}

export function campaignListMarginCostLabel(margin?: CampaignMargin): string {
  if (!margin?.rtb_cost_micro) {
    return '';
  }
  return formatDashboardUsdFromMicro(margin.rtb_cost_micro);
}

export function campaignListClicksLabel(metrics?: CampaignListMetrics): string {
  return displayCount(metrics?.clicks);
}

export function campaignListConversionsLabel(metrics?: CampaignListMetrics): string {
  return displayCount(metrics?.conversions);
}

export type CampaignListTotals = {
  flows: number;
  clicks: number;
  conversions: number;
  revenueMicro: number;
  costMicro: number;
  profitMicro: number;
};

export function emptyCampaignListTotals(): CampaignListTotals {
  return {
    flows: 0,
    clicks: 0,
    conversions: 0,
    revenueMicro: 0,
    costMicro: 0,
    profitMicro: 0,
  };
}

export function sumCampaignListTotals(
  items: CampaignWithMoneyDisplay[],
  metricsById: Record<string, CampaignListMetrics>,
  marginsById: Record<string, CampaignMargin>,
): CampaignListTotals {
  const totals = emptyCampaignListTotals();
  for (const campaign of items) {
    if (campaign.flow_id) {
      totals.flows += 1;
    }
    const metrics = metricsById[campaign.id];
    totals.clicks += metrics?.clicks ?? 0;
    totals.conversions += metrics?.conversions ?? 0;
    const margin = marginsById[campaign.id];
    totals.revenueMicro += campaignListRevenueMicro(margin);
    totals.costMicro += margin?.rtb_cost_micro ?? 0;
    totals.profitMicro += margin?.operator_margin_micro ?? 0;
  }
  return totals;
}
