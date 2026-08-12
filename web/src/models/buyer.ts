import type { BuyerCampaignPortfolioRow, BuyerPortfolioResponse } from '../types/api/campaign.js';

export type BuyerAttentionDTO = {
  id: string;
  name: string;
  reason: string;
};

export type BuyerCampaignStatsDTO = {
  impressions: number;
  clicks: number;
};

/** @deprecated Prefer BuyerCampaignPortfolioRow from types/api */
export type BuyerCampaignRow = BuyerCampaignPortfolioRow;

export type BuyerPortfolioVM = {
  active: number;
  paused: number;
  archived: number;
  attention: BuyerAttentionDTO[];
  impressions7d: number;
  clicks7d: number;
  overspendCount: number;
  kpis: Record<string, unknown> | null;
  recommendations: unknown[];
  alerts: unknown[];
  campaigns: BuyerCampaignPortfolioRow[];
  sampled: number;
};

export type PortfolioRowVM = {
  row: BuyerCampaignPortfolioRow;
  driftScore: number;
};

type CampaignIndex = {
  __src?: BuyerCampaignPortfolioRow[] | null | undefined;
  [id: string]: BuyerCampaignPortfolioRow | BuyerCampaignPortfolioRow[] | null | undefined;
};

/**
 * Read 7d delivery stats from a campaign row (no duplicate map).
 */
export function buyerCampaignStat(
  campaign: { impressions_7d?: number; clicks_7d?: number } | null | undefined,
): BuyerCampaignStatsDTO {
  return {
    impressions: Number(campaign?.impressions_7d ?? 0),
    clicks: Number(campaign?.clicks_7d ?? 0),
  };
}

/**
 * Build a campaign-id index from dashboard rows (lazy, single pass).
 */
export function buyerCampaignIndex(
  campaigns: BuyerCampaignPortfolioRow[] | null | undefined,
  cache: CampaignIndex | null = null,
): CampaignIndex {
  if (cache && cache.__src === campaigns) return cache;
  const index: CampaignIndex = { __src: campaigns };
  const rows = campaigns ?? [];
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    const id = row.id ?? '';
    if (id) index[id] = row;
  }
  return index;
}

/**
 * Map buyer dashboard API payload into view-model fields.
 */
export function mapBuyerDashboard(data: BuyerPortfolioResponse | null | undefined): BuyerPortfolioVM {
  const campaigns = data?.campaigns ?? [];
  return {
    active: Number(data?.active ?? 0),
    paused: Number(data?.paused ?? 0),
    archived: Number(data?.archived ?? 0),
    attention: (data?.attention ?? []).map((a) => ({
      id: a.id,
      name: a.name ?? '',
      reason: a.reason ?? '',
    })),
    impressions7d: Number(data?.impressions_7d ?? 0),
    clicks7d: Number(data?.clicks_7d ?? 0),
    overspendCount: Number(data?.overspend_count ?? 0),
    kpis: data?.kpis ?? null,
    recommendations: data?.recommendations ?? [],
    alerts: data?.alerts ?? [],
    campaigns,
    sampled: campaigns.length,
  };
}

/**
 * Return server pacing drift percent for sorting (absolute value; higher = more drift).
 */
export function portfolioDriftPct(campaign: { pacing_drift_pct?: number | null }): number | null {
  const pct = campaign?.pacing_drift_pct;
  if (pct == null || Number.isNaN(Number(pct))) return null;
  return Math.abs(Number(pct));
}

/**
 * Estimate pacing drift priority when API field is absent (higher = needs attention).
 */
export function pacingDriftScore(campaign: {
  status?: string;
  pacing_mode?: string;
  impressions_7d?: number;
  clicks_7d?: number;
}): number {
  let score = 0;
  const status = String(campaign.status ?? '').toUpperCase();
  const mode = String(campaign.pacing_mode ?? 'even').toLowerCase();
  if (status === 'PAUSED') score += 100;
  if (mode !== 'even' && mode !== '') score += 50;
  const impr = Number(campaign.impressions_7d ?? 0);
  if (status === 'ACTIVE' && impr === 0) score += 75;
  const clicks = Number(campaign.clicks_7d ?? 0);
  if (impr > 0 && clicks === 0) score += 25;
  return score;
}

/**
 * Filter portfolio campaign rows by status (no copy when filter empty).
 */
export function filterPortfolioCampaigns(
  campaigns: BuyerCampaignRow[],
  statusFilter: string,
): BuyerCampaignRow[] {
  if (!statusFilter) return campaigns;
  const want = statusFilter.toUpperCase();
  const out: BuyerCampaignRow[] = [];
  for (let i = 0; i < campaigns.length; i++) {
    const row = campaigns[i];
    if (String(row.status ?? '').toUpperCase() === want) out.push(row);
  }
  return out;
}

/**
 * Sort key for portfolio rows: server pacing_drift_pct when present, else heuristic score.
 */
export function portfolioDriftSortKey(campaign: BuyerCampaignRow): number {
  const apiPct = portfolioDriftPct(campaign);
  if (apiPct != null) return apiPct;
  return pacingDriftScore(campaign);
}

/**
 * Sort portfolio rows by pacing drift descending; scores computed once per row.
 */
export function sortPortfolioByDrift(campaigns: BuyerCampaignRow[]): PortfolioRowVM[] {
  const n = campaigns.length;
  const decorated = new Array<PortfolioRowVM>(n);
  for (let i = 0; i < n; i++) {
    const row = campaigns[i];
    decorated[i] = { row, driftScore: portfolioDriftSortKey(row) };
  }
  decorated.sort((a, b) => b.driftScore - a.driftScore);
  return decorated;
}

export type PortfolioRowsCache = {
  portfolio: BuyerPortfolioVM | null;
  filter: string;
  rows: PortfolioRowVM[] | null;
};

/**
 * Memoized portfolio row list for table rendering.
 */
export function visiblePortfolioRows(
  portfolio: BuyerPortfolioVM | null,
  statusFilter: string,
  cache: PortfolioRowsCache,
): PortfolioRowVM[] {
  if (cache.portfolio === portfolio && cache.filter === statusFilter && cache.rows) {
    return cache.rows;
  }
  const base = portfolio?.campaigns ?? [];
  const filtered = filterPortfolioCampaigns(base, statusFilter);
  const rows = sortPortfolioByDrift(filtered);
  cache.portfolio = portfolio;
  cache.filter = statusFilter;
  cache.rows = rows;
  return rows;
}

/**
 * Estimate delivery percent from impressions when budget is hidden.
 */
export function estimateDeliveryPct(impressions7d: number, status: string): number | null {
  if (String(status).toUpperCase() === 'PAUSED') return 0;
  if (impressions7d <= 0) return 0;
  if (impressions7d < 1000) return 35;
  if (impressions7d < 10000) return 68;
  return 92;
}
