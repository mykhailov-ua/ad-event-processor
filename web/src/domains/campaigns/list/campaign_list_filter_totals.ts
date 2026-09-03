import type { CampaignListMetricsTotalsResponse } from '@/api/campaigns_api';
import { resolveCampaignFunnelCounts } from '@/domains/campaigns/list/campaign_list_funnel';
import {
  emptyCampaignListTotals,
  type CampaignListTotals,
} from '@/domains/campaigns/list/campaign_list_format';
import type { CampaignFunnelCounts } from '@/domains/campaigns/list/campaign_list_funnel';

export type CampaignListFilterTotalsView = {
  totals: CampaignListTotals;
  funnelTotals: CampaignFunnelCounts;
  campaignCount: number;
  marginBreachCount: number;
  stale: boolean;
};

export function campaignListFilterTotalsFromApi(
  response: CampaignListMetricsTotalsResponse | undefined,
): CampaignListFilterTotalsView | undefined {
  if (!response) {
    return undefined;
  }
  const row = response.totals;
  const funnelTotals = resolveCampaignFunnelCounts({
    impressions: row.impressions,
    clicks: row.clicks,
    conversions: row.conversions,
    unique_clicks: row.unique_clicks,
    blocks: row.blocks,
    leads_raw: row.leads_raw,
    hold_leads: row.hold_leads,
    rejected_leads: row.rejected_leads,
    lp_clicks: row.lp_clicks,
    lp_views: row.lp_views,
    bots: row.bots,
    stale: response.stale,
    revenue_micro: row.revenue_micro,
    cost_micro: row.cost_micro,
    profit_micro: row.profit_micro,
  });
  const totals = emptyCampaignListTotals();
  totals.flows = response.flow_count;
  totals.clicks = row.clicks ?? 0;
  totals.impressions = row.impressions ?? 0;
  totals.blocks = row.blocks ?? 0;
  totals.conversions = row.conversions ?? 0;
  totals.revenueMicro = row.revenue_micro ?? 0;
  totals.costMicro = row.cost_micro ?? 0;
  totals.profitMicro = row.profit_micro ?? 0;
  return {
    totals,
    funnelTotals,
    campaignCount: response.campaign_count,
    marginBreachCount: response.margin_breach_count,
    stale: response.stale,
  };
}
