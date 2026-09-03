export type DevMockCampaignMetrics = {
  impressions: number;
  clicks: number;
  conversions: number;
  unique_clicks: number;
  blocks: number;
  rtb_cost_micro: number;
  profit_micro: number;
  revenue_micro: number;
  leads_raw: number;
  hold_leads: number;
  rejected_leads: number;
  lp_clicks: number;
  lp_views: number;
  bots: number;
};

export function devMockCampaignMetricsRangeScale(fromIso: string, toIso: string): number {
  const fromMs = Date.parse(fromIso);
  const toMs = Date.parse(toIso);
  if (!Number.isFinite(fromMs) || !Number.isFinite(toMs) || toMs <= fromMs) {
    return 1;
  }
  const days = Math.max(1, (toMs - fromMs) / 86_400_000);
  return days / 7;
}

export function buildDevMockCampaignMetrics(
  campaignId: string,
  seq: number,
  fromIso: string,
  toIso: string,
): DevMockCampaignMetrics {
  const scale = devMockCampaignMetricsRangeScale(fromIso, toIso);
  const scaleCount = (value: number) => Math.max(0, Math.round(value * scale));
  const impressions = scaleCount(12_000 + seq * 1_340);
  const clicks = scaleCount(420 + seq * 37);
  const conversions = scaleCount(18 + (seq % 11));
  const holdLeads = scaleCount(3 + (seq % 5));
  const rejectedLeads = scaleCount(2 + (seq % 4));
  const leadsRaw = conversions + holdLeads + rejectedLeads;
  const lpClicks = scaleCount(Math.floor((420 + seq * 37) * (0.42 + (seq % 7) * 0.04)));
  const lpViews = Math.max(lpClicks, scaleCount(Math.floor((420 + seq * 37) * 0.88)));
  const bots = scaleCount(4 + (seq % 9));
  const rtbCostMicro = scaleCount(1_200_000 + seq * 88_000);
  const profitMicro = scaleCount(180_000 + seq * 12_000 - (seq % 5) * 40_000);
  const revenueMicro = rtbCostMicro + profitMicro;

  return {
    impressions,
    clicks,
    conversions,
    unique_clicks: scaleCount(Math.floor((420 + seq * 37) * 0.82)),
    blocks: seq % 7,
    rtb_cost_micro: rtbCostMicro,
    profit_micro: profitMicro,
    revenue_micro: revenueMicro,
    leads_raw: leadsRaw,
    hold_leads: holdLeads,
    rejected_leads: rejectedLeads,
    lp_clicks: lpClicks,
    lp_views: lpViews,
    bots,
  };
}

export function syncDevMockCampaignLeadsRaw(metrics: DevMockCampaignMetrics): void {
  if (metrics.leads_raw > 0) {
    return;
  }
  const derived = metrics.conversions + metrics.hold_leads + metrics.rejected_leads;
  metrics.leads_raw = derived > 0 ? derived : metrics.conversions;
}

function devMockRatePercent(numerator: number, denominator: number): number | undefined {
  if (denominator <= 0 || numerator <= 0) {
    return undefined;
  }
  return (numerator / denominator) * 100;
}

// Same shape as enrichCampaignListMetricsRowDerived; keeps dev mock metrics batch aligned.
export function enrichDevMockCampaignMetricsDerived(
  row: Record<string, unknown>,
  metrics: DevMockCampaignMetrics,
): void {
  syncDevMockCampaignLeadsRaw(metrics);

  const revenueMicro = metrics.revenue_micro;
  const costMicro = metrics.rtb_cost_micro;
  const profitMicro = metrics.profit_micro;

  row.revenue_micro = revenueMicro;
  row.cost_micro = costMicro;
  row.profit_micro = profitMicro;

  if (metrics.clicks > 0) {
    row.epc_micro = Math.trunc(revenueMicro / metrics.clicks);
    row.cpc_micro = Math.trunc(costMicro / metrics.clicks);
  }
  if (metrics.leads_raw > 0) {
    row.cpa_micro = Math.trunc(costMicro / metrics.leads_raw);
  }
  if (metrics.conversions > 0) {
    row.ecpa_micro = Math.trunc(costMicro / metrics.conversions);
  }

  const ctrPct = devMockRatePercent(metrics.clicks, metrics.impressions);
  if (ctrPct != null) {
    row.ctr_pct = ctrPct;
  }
  const lpCtrPct = devMockRatePercent(metrics.lp_clicks, metrics.clicks);
  if (lpCtrPct != null) {
    row.lp_ctr_pct = lpCtrPct;
  }
  row.cr_pct = devMockRatePercent(metrics.conversions, metrics.clicks) ?? 0;
  const approveRatePct = devMockRatePercent(metrics.conversions, metrics.leads_raw);
  if (approveRatePct != null) {
    row.approve_rate_pct = approveRatePct;
  }
  const blockPct = devMockRatePercent(metrics.blocks, metrics.clicks);
  if (blockPct != null) {
    row.block_pct = blockPct;
  }
  const botPct = devMockRatePercent(metrics.bots, metrics.clicks);
  if (botPct != null) {
    row.bot_pct = botPct;
  }
  if (costMicro > 0) {
    row.roi_pct = (profitMicro / costMicro) * 100;
  }
  if (metrics.impressions > 0 && costMicro > 0) {
    const cpmMicro = (costMicro * 1000) / metrics.impressions;
    row.cpm_usd = (cpmMicro / 1_000_000).toFixed(2);
  }
}

export function compareDevMockCampaignMetricSort(
  sort: string,
  left: DevMockCampaignMetrics,
  right: DevMockCampaignMetrics,
): number {
  syncDevMockCampaignLeadsRaw(left);
  syncDevMockCampaignLeadsRaw(right);

  const value = (metrics: DevMockCampaignMetrics): number => {
    switch (sort) {
      case 'clicks':
        return metrics.clicks;
      case 'impressions':
        return metrics.impressions;
      case 'conversions':
        return metrics.conversions;
      case 'unique_clicks':
        return metrics.unique_clicks;
      case 'blocks':
        return metrics.blocks;
      case 'cost':
        return metrics.rtb_cost_micro;
      case 'revenue':
        return metrics.revenue_micro;
      case 'profit':
        return metrics.profit_micro;
      case 'ctr':
        return metrics.impressions > 0 ? metrics.clicks / metrics.impressions : 0;
      case 'cr':
        return metrics.clicks > 0 ? metrics.conversions / metrics.clicks : 0;
      case 'block_pct':
        return metrics.clicks > 0 ? metrics.blocks / metrics.clicks : 0;
      case 'cpc':
        return metrics.clicks > 0 ? metrics.rtb_cost_micro / metrics.clicks : 0;
      case 'cpa':
      case 'ecpa':
        return metrics.conversions > 0 ? metrics.rtb_cost_micro / metrics.conversions : 0;
      case 'epc':
        return metrics.clicks > 0 ? metrics.revenue_micro / metrics.clicks : 0;
      case 'cpm':
        return metrics.impressions > 0 && metrics.rtb_cost_micro > 0
          ? metrics.rtb_cost_micro / metrics.impressions
          : 0;
      case 'roi':
        return metrics.rtb_cost_micro > 0 ? metrics.profit_micro / metrics.rtb_cost_micro : 0;
      case 'leads':
        return metrics.leads_raw;
      case 'approved':
        return metrics.conversions;
      case 'hold_leads':
        return metrics.hold_leads;
      case 'rejected_leads':
        return metrics.rejected_leads;
      case 'approve_rate':
        return metrics.leads_raw > 0 ? metrics.conversions / metrics.leads_raw : 0;
      case 'lp_clicks':
        return metrics.lp_clicks;
      case 'lp_views':
        return metrics.lp_views;
      case 'lp_ctr':
        return metrics.clicks > 0 ? metrics.lp_clicks / metrics.clicks : 0;
      case 'bots':
        return metrics.bots;
      case 'bot_pct':
        return metrics.clicks > 0 ? metrics.bots / metrics.clicks : 0;
      default:
        return 0;
    }
  };

  return value(left) - value(right);
}

export function devMockCampaignMetricSortNeedsWindow(sort: string): boolean {
  switch (sort) {
    case 'clicks':
    case 'impressions':
    case 'conversions':
    case 'unique_clicks':
    case 'blocks':
    case 'cost':
    case 'revenue':
    case 'profit':
    case 'roi':
    case 'ctr':
    case 'cr':
    case 'cpc':
    case 'cpa':
    case 'ecpa':
    case 'epc':
    case 'cpm':
    case 'block_pct':
    case 'leads':
    case 'approved':
    case 'hold_leads':
    case 'rejected_leads':
    case 'approve_rate':
    case 'lp_clicks':
    case 'lp_views':
    case 'lp_ctr':
    case 'bots':
    case 'bot_pct':
      return true;
    default:
      return false;
  }
}
