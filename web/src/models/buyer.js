/**
 * @typedef {{ id: string, name: string, reason: string }} BuyerAttentionDTO
 * @typedef {{ impressions: number, clicks: number }} BuyerCampaignStatsDTO
 * @typedef {{
 *   active: number,
 *   paused: number,
 *   archived: number,
 *   attention: BuyerAttentionDTO[],
 *   impressions7d: number,
 *   clicks7d: number,
 *   overspendCount: number,
 *   kpis: object|null,
 *   campaigns: object[],
 *   sampled: number,
 * }} BuyerPortfolioVM
 * @typedef {{ row: object, driftScore: number }} PortfolioRowVM
 */

/**
 * Read 7d delivery stats from a campaign row (no duplicate map).
 *
 * @param {{ impressions_7d?: number, clicks_7d?: number }|null|undefined} campaign
 * @returns {BuyerCampaignStatsDTO}
 */
export function buyerCampaignStat(campaign) {
  return {
    impressions: Number(campaign?.impressions_7d ?? 0),
    clicks: Number(campaign?.clicks_7d ?? 0),
  };
}

/**
 * Build a campaign-id index from dashboard rows (lazy, single pass).
 *
 * @param {object[]|null|undefined} campaigns
 * @param {Record<string, object>|null} [cache]
 * @returns {Record<string, object>}
 */
export function buyerCampaignIndex(campaigns, cache = null) {
  if (cache && cache.__src === campaigns) return cache;
  /** @type {Record<string, object>} */
  const index = { __src: campaigns };
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
 *
 * @param {object|null|undefined} data
 * @returns {BuyerPortfolioVM}
 */
export function mapBuyerDashboard(data) {
  const campaigns = data?.campaigns ?? [];
  return {
    active: Number(data?.active ?? 0),
    paused: Number(data?.paused ?? 0),
    archived: Number(data?.archived ?? 0),
    attention: data?.attention ?? [],
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
 *
 * @param {{ pacing_drift_pct?: number|null }} campaign
 * @returns {number|null}
 */
export function portfolioDriftPct(campaign) {
  const pct = campaign?.pacing_drift_pct;
  if (pct == null || Number.isNaN(Number(pct))) return null;
  return Math.abs(Number(pct));
}

/**
 * Estimate pacing drift priority when API field is absent (higher = needs attention).
 *
 * @param {{ status?: string, pacing_mode?: string, impressions_7d?: number, clicks_7d?: number }} campaign
 * @returns {number}
 */
export function pacingDriftScore(campaign) {
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
 *
 * @param {object[]} campaigns
 * @param {string} statusFilter
 * @returns {object[]}
 */
export function filterPortfolioCampaigns(campaigns, statusFilter) {
  if (!statusFilter) return campaigns;
  const want = statusFilter.toUpperCase();
  const out = [];
  for (let i = 0; i < campaigns.length; i++) {
    const row = campaigns[i];
    if (String(row.status ?? '').toUpperCase() === want) out.push(row);
  }
  return out;
}

/**
 * Sort key for portfolio rows: server pacing_drift_pct when present, else heuristic score.
 *
 * @param {object} campaign
 * @returns {number}
 */
export function portfolioDriftSortKey(campaign) {
  const apiPct = portfolioDriftPct(campaign);
  if (apiPct != null) return apiPct;
  return pacingDriftScore(campaign);
}

/**
 * Sort portfolio rows by pacing drift descending; scores computed once per row.
 *
 * @param {object[]} campaigns
 * @returns {PortfolioRowVM[]}
 */
export function sortPortfolioByDrift(campaigns) {
  const n = campaigns.length;
  const decorated = new Array(n);
  for (let i = 0; i < n; i++) {
    const row = campaigns[i];
    decorated[i] = { row, driftScore: portfolioDriftSortKey(row) };
  }
  decorated.sort((a, b) => b.driftScore - a.driftScore);
  return decorated;
}

/**
 * Memoized portfolio row list for table rendering.
 *
 * @param {BuyerPortfolioVM|null} portfolio
 * @param {string} statusFilter
 * @param {{ portfolio: BuyerPortfolioVM|null, filter: string, rows: PortfolioRowVM[]|null }} cache
 * @returns {PortfolioRowVM[]}
 */
export function visiblePortfolioRows(portfolio, statusFilter, cache) {
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
 *
 * @param {number} impressions7d
 * @param {string} status
 * @returns {number|null}
 */
export function estimateDeliveryPct(impressions7d, status) {
  if (String(status).toUpperCase() === 'PAUSED') return 0;
  if (impressions7d <= 0) return 0;
  if (impressions7d < 1000) return 35;
  if (impressions7d < 10000) return 68;
  return 92;
}
