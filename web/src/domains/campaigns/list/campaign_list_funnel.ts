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

// Keep aligned with campaignListSortMetrics.syncLeadsRaw in list_sort_extended.go.
export function syncCampaignFunnelLeadsRaw(
  approved: number,
  hold: number,
  rejected: number,
  leadsRaw?: number | null
): number {
  if (leadsRaw != null && leadsRaw > 0) {
    return leadsRaw;
  }
  const derived = approved + hold + rejected;
  return derived > 0 ? derived : approved;
}

export function resolveCampaignFunnelCounts(metrics?: CampaignListMetrics): CampaignFunnelCounts {
  const approved = metrics?.conversions ?? 0;
  const hold = metrics?.hold_leads ?? 0;
  const rejected = metrics?.rejected_leads ?? 0;
  const rawLeads = syncCampaignFunnelLeadsRaw(approved, hold, rejected, metrics?.leads_raw);
  const clicks = metrics?.clicks ?? 0;
  const lpClicks = metrics?.lp_clicks ?? 0;
  // API may omit lp_views; when lp_clicks exist, default to max(lp_clicks, clicks).
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
