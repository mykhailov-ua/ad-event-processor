import type { CampaignListMetrics } from '@/api/campaigns_api';

export type CampaignFunnelCounts = {
  rawLeads: number;
  approved: number;
  hold: number;
  rejected: number;
  lpClicks: number;
  lpViews: number;
  bots: number;
};

export function resolveCampaignFunnelCounts(metrics?: CampaignListMetrics): CampaignFunnelCounts {
  const approved = metrics?.conversions ?? 0;
  const hold = metrics?.hold_leads ?? 0;
  const rejected = metrics?.rejected_leads ?? 0;
  const derivedRaw = approved + hold + rejected;
  const rawLeads = metrics?.leads_raw ?? (derivedRaw > 0 ? derivedRaw : approved);
  const clicks = metrics?.clicks ?? 0;
  const lpClicks = metrics?.lp_clicks ?? 0;
  const lpViews = metrics?.lp_views ?? (lpClicks > 0 ? Math.max(lpClicks, clicks) : 0);
  const bots = metrics?.bots ?? 0;

  return {
    rawLeads,
    approved,
    hold,
    rejected,
    lpClicks,
    lpViews,
    bots,
  };
}

export function formatPercentRate(numerator: number, denominator: number, digits = 2): string {
  if (denominator <= 0 || numerator <= 0) {
    return '0.00%';
  }
  return `${((numerator / denominator) * 100).toFixed(digits)}%`;
}

export function formatApproveRate(approved: number, rawLeads: number): string {
  return formatPercentRate(approved, rawLeads);
}

export function formatSourceCtr(clicks: number, impressions: number): string {
  return formatPercentRate(clicks, impressions);
}

export function formatLpCtr(lpClicks: number, clicks: number): string {
  return formatPercentRate(lpClicks, clicks);
}

export function formatRelativeRate(count: number, clicks: number): string {
  return formatPercentRate(count, clicks);
}

export function formatCpmUsd(costMicro: number, impressions: number): string {
  if (impressions <= 0 || costMicro <= 0) {
    return '0.00';
  }
  const cpmMicro = (costMicro * 1000) / impressions;
  return (cpmMicro / 1_000_000).toFixed(2);
}

export function sumCampaignFunnelTotals(
  items: ReadonlyArray<{ id: string }>,
  metricsById: Readonly<Record<string, CampaignListMetrics>>,
): CampaignFunnelCounts {
  const totals: CampaignFunnelCounts = {
    rawLeads: 0,
    approved: 0,
    hold: 0,
    rejected: 0,
    lpClicks: 0,
    lpViews: 0,
    bots: 0,
  };
  for (const item of items) {
    const funnel = resolveCampaignFunnelCounts(metricsById[item.id]);
    totals.rawLeads += funnel.rawLeads;
    totals.approved += funnel.approved;
    totals.hold += funnel.hold;
    totals.rejected += funnel.rejected;
    totals.lpClicks += funnel.lpClicks;
    totals.lpViews += funnel.lpViews;
    totals.bots += funnel.bots;
  }
  return totals;
}
