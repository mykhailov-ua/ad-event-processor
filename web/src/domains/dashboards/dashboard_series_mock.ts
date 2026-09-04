import { eachDayOfInterval, format, getDate, getDay, getMonth } from 'date-fns';

import type {
  BuyerPortfolio,
  ClickLogEvent,
  DashboardBreakdownRow,
  DashboardBreakdownTable,
  DashboardBreakdownTotals,
  DashboardSeriesPoint,
} from '@/domains/dashboards/buyer_dashboard_types';

const MICRO_PER_USD = 1_000_000;

/** Inclusive mock window: 1 Jul through 1 Sep (dashboard preview). */
export const DASHBOARD_MOCK_DEFAULT_FROM = new Date(2026, 6, 1);
export const DASHBOARD_MOCK_DEFAULT_TO = new Date(2026, 8, 1);

const WEEKDAY_FACTORS = [0.72, 1.04, 1.08, 1.05, 1.0, 0.88, 0.69];

const CAMPAIGN_FIXTURES = [
  { id: 'cmp-us-fb-browser', name: 'US | Facebook | Browser', share: 0.31, roiSkew: 0.08 },
  { id: 'cmp-de-push-finance', name: 'DE | Push | Finance', share: 0.19, roiSkew: 0.04 },
  { id: 'cmp-br-tiktok-app', name: 'BR | TikTok | App install', share: 0.16, roiSkew: -0.06 },
  { id: 'cmp-pl-native-loan', name: 'PL | Native | Loan', share: 0.12, roiSkew: 0.02 },
  { id: 'cmp-gb-google-search', name: 'GB | Google | Search', share: 0.11, roiSkew: 0.11 },
  { id: 'cmp-ca-taboola-vsl', name: 'CA | Taboola | VSL', share: 0.07, roiSkew: -0.12 },
  { id: 'cmp-in-pop-travel', name: 'IN | Popunder | Travel', share: 0.04, roiSkew: -0.18 },
];

const LANDER_FIXTURES = [
  { id: 'lp-finance-v3', name: 'finance-offer-v3.html', share: 0.28 },
  { id: 'lp-sweeps-mob', name: 'sweeps-mobile-lp', share: 0.22 },
  { id: 'lp-vsl-long', name: 'vsl-17min-lander', share: 0.18 },
  { id: 'lp-app-pre', name: 'app-install-prelander', share: 0.14 },
  { id: 'lp-direct-offer', name: 'direct-offer-page', share: 0.11 },
  { id: 'lp-quiz-funnel', name: 'quiz-funnel-step-1', share: 0.07 },
];

const OFFER_FIXTURES = [
  { id: 'off-credit-line', name: 'CreditLine Pro (CPL)', share: 0.26 },
  { id: 'off-casino-welcome', name: 'Casino Welcome (CPA)', share: 0.21 },
  { id: 'off-travel-book', name: 'Travel Booking (CPS)', share: 0.17 },
  { id: 'off-app-finance', name: 'Finance App (CPI)', share: 0.15 },
  { id: 'off-insurance-quote', name: 'Insurance Quote (CPL)', share: 0.12 },
  { id: 'off-sweeps-entry', name: 'Sweeps Entry (CPL)', share: 0.09 },
];

const SOURCE_FIXTURES = [
  { id: 'src-facebook', name: 'facebook.com', share: 0.34 },
  { id: 'src-push', name: 'push_subscribers', share: 0.22 },
  { id: 'src-google', name: 'google_ads', share: 0.18 },
  { id: 'src-tiktok', name: 'tiktok_ads', share: 0.14 },
  { id: 'src-taboola', name: 'taboola', share: 0.07 },
  { id: 'src-outbrain', name: 'outbrain', share: 0.05 },
];

const RECENT_CLICK_COUNTRIES = ['US', 'DE', 'BR', 'PL', 'GB', 'CA', 'IN', 'AU'];
const RECENT_CLICK_SOURCES = [
  'facebook.com',
  'push_subscribers',
  'google_ads',
  'tiktok_ads',
  'taboola',
  'outbrain',
  'mgid',
  'revcontent',
];
const RECENT_CLICK_GOALS = ['lead', 'sale', 'install', 'signup', 'deposit'];

function usdToMicro(usd: number): number {
  return Math.round(usd * MICRO_PER_USD);
}

function hashUnit(seed: number): number {
  let value = Math.imul(seed ^ (seed >>> 16), 0x7feb352d);
  value = Math.imul(value ^ (value >>> 15), 0x846ca68b);
  value ^= value >>> 16;
  return (value >>> 0) / 0xffffffff;
}

function calendarDipFactor(day: Date): number {
  const month = getMonth(day);
  const dom = getDate(day);
  if (month === 6 && dom === 4) {
    return 0.76;
  }
  if (month === 6 && dom === 3) {
    return 0.9;
  }
  if (month === 7 && dom >= 20 && dom <= 27) {
    return 0.93;
  }
  if (dom === 1 || dom === 15) {
    return 1.08;
  }
  return 1;
}

function dayTrafficMultiplier(day: Date, index: number, totalDays: number): number {
  const weekday = getDay(day);
  let factor = WEEKDAY_FACTORS[weekday] ?? 1;
  factor *= calendarDipFactor(day);
  const trend = 0.9 + (index / Math.max(totalDays - 1, 1)) * 0.22;
  const noise = 0.9 + hashUnit(day.getTime() >>> 0) * 0.2;
  return factor * trend * noise;
}

export function isDashboardChartMockEnabled(): boolean {
  // Non-prod tier: synthetic dashboard data when ?chart_mock=1 (see web/WEB.md).
  if (typeof window === 'undefined') {
    return false;
  }
  const params = new URLSearchParams(window.location.search);
  if (params.get('chart_mock') === '0') {
    return false;
  }
  return params.get('chart_mock') === '1';
}

function isBuyerPortfolioEmpty(portfolio: BuyerPortfolio): boolean {
  const series = portfolio.series ?? [];
  const hasSeries = series.some(
    (point) =>
      (point.clicks ?? 0) > 0 ||
      (point.conversions ?? 0) > 0 ||
      (point.spend_micro ?? point.spend_micros ?? 0) > 0 ||
      (point.revenue_micro ?? 0) > 0,
  );
  const hasKpis =
    (portfolio.kpis?.conversions ?? 0) > 0 ||
    (portfolio.kpis?.cost_micro ?? portfolio.kpis?.spend_micro ?? 0) > 0 ||
    (portfolio.clicks_7d ?? 0) > 0;
  const hasBreakdown =
    (portfolio.breakdowns?.campaigns?.rows?.length ?? 0) > 0 ||
    (portfolio.breakdowns?.landers?.rows?.length ?? 0) > 0 ||
    (portfolio.breakdowns?.offers?.rows?.length ?? 0) > 0 ||
    (portfolio.breakdowns?.sources?.rows?.length ?? 0) > 0;
  const hasRecentClicks = (portfolio.recent_clicks?.length ?? 0) > 0;
  return !hasSeries && !hasKpis && !hasBreakdown && !hasRecentClicks;
}

function isLocalAdminDev(): boolean {
  if (typeof window === 'undefined') {
    return false;
  }
  const { hostname, port } = window.location;
  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    return true;
  }
  return port === '5173';
}

function shouldFillDashboardDemoData(portfolio: BuyerPortfolio): boolean {
  if (isDashboardChartMockEnabled()) {
    return true;
  }
  return isLocalAdminDev() && isBuyerPortfolioEmpty(portfolio);
}

export function buildDashboardMockSeries(
  from = DASHBOARD_MOCK_DEFAULT_FROM,
  to = DASHBOARD_MOCK_DEFAULT_TO,
): DashboardSeriesPoint[] {
  const days = eachDayOfInterval({ start: from, end: to });
  const totalDays = days.length;

  return days.map((day, index) => {
    const traffic = dayTrafficMultiplier(day, index, totalDays);
    const clicksBase = 38_600 + index * 142;
    const clicks = Math.max(
      120,
      Math.round(clicksBase * traffic + (hashUnit(index * 17 + 3) > 0.96 ? 6_200 : 0)),
    );

    const cpcUsd = 0.11 + hashUnit(index * 7 + 2) * 0.14;
    const conversionRate = 0.0034 + hashUnit(index * 9 + 5) * 0.0028;
    const conversions = Math.max(1, Math.round(clicks * conversionRate));

    const costUsd = clicks * cpcUsd * (0.96 + hashUnit(index * 13 + 1) * 0.08);
    const payoutUsd = 19 + hashUnit(index * 11 + 7) * 31;
    const revenueUsd =
      conversions * payoutUsd * (0.92 + hashUnit(index * 19 + 4) * 0.16) +
      (hashUnit(index * 23) > 0.94 ? 180 : 0);
    const profitUsd = revenueUsd - costUsd;

    return {
      label: format(day, 'yyyy-MM-dd'),
      clicks,
      conversions,
      spend_micro: usdToMicro(costUsd),
      revenue_micro: usdToMicro(revenueUsd),
      profit_micro: usdToMicro(profitUsd),
    };
  });
}

type AggregateTotals = {
  clicks: number;
  unique_clicks: number;
  conversions: number;
  cost_micro: number;
  revenue_micro: number;
  profit_micro: number;
  roi_pct: number;
};

function aggregateSeries(series: DashboardSeriesPoint[]): AggregateTotals {
  const clicks = series.reduce((sum, point) => sum + (point.clicks ?? 0), 0);
  const conversions = series.reduce((sum, point) => sum + (point.conversions ?? 0), 0);
  const cost_micro = series.reduce((sum, point) => sum + (point.spend_micro ?? 0), 0);
  const revenue_micro = series.reduce((sum, point) => sum + (point.revenue_micro ?? 0), 0);
  const profit_micro = revenue_micro - cost_micro;
  const unique_clicks = Math.round(clicks * (0.857 + hashUnit(clicks) * 0.04));
  const roi_pct = cost_micro > 0 ? (profit_micro / cost_micro) * 100 : 0;
  return { clicks, unique_clicks, conversions, cost_micro, revenue_micro, profit_micro, roi_pct };
}

function computeCpcMicro(costMicro: number, clicks: number): number {
  if (clicks <= 0 || costMicro <= 0) {
    return 0;
  }
  return Math.round(costMicro / clicks);
}

function computeCpaMicro(costMicro: number, conversions: number): number {
  if (conversions <= 0 || costMicro <= 0) {
    return 0;
  }
  return Math.round(costMicro / conversions);
}

function computeEpcMicro(revenueMicro: number, clicks: number): number {
  if (clicks <= 0 || revenueMicro <= 0) {
    return 0;
  }
  return Math.round(revenueMicro / clicks);
}

function computeCrPct(conversions: number, clicks: number): number {
  if (clicks <= 0 || conversions <= 0) {
    return 0;
  }
  return (conversions / clicks) * 100;
}

function enrichEconomicsRow(row: DashboardBreakdownRow): DashboardBreakdownRow {
  const profit_micro =
    row.profit_micro ?? (row.revenue_micro ?? 0) - (row.cost_micro ?? 0);
  const clicks = row.clicks ?? 0;
  const conversions = row.conversions ?? 0;
  const cost_micro = row.cost_micro ?? 0;
  const revenue_micro = row.revenue_micro ?? 0;
  return {
    ...row,
    profit_micro,
    roi_pct: cost_micro > 0 ? (profit_micro / cost_micro) * 100 : 0,
    cpc_micro: computeCpcMicro(cost_micro, clicks),
    cpa_micro: computeCpaMicro(cost_micro, conversions),
    cr_pct: computeCrPct(conversions, clicks),
    epc_micro: computeEpcMicro(revenue_micro, clicks),
  };
}

function enrichEconomicsTotals(totals: DashboardBreakdownTotals): DashboardBreakdownTotals {
  return enrichEconomicsRow(totals);
}

function buildBreakdownRows(
  fixtures: { id: string; name: string; share: number; roiSkew?: number }[],
  totals: AggregateTotals,
): DashboardBreakdownRow[] {
  return fixtures.map((fixture, index) => {
    const shareJitter = 0.94 + hashUnit(index * 23 + 1) * 0.12;
    const share = fixture.share * shareJitter;
    const clicks = Math.max(0, Math.round(totals.clicks * share));
    const unique_clicks = Math.max(0, Math.round(clicks * (0.82 + hashUnit(index + 4) * 0.1)));
    const conversions = Math.max(0, Math.round(clicks * (0.0038 + hashUnit(index + 9) * 0.0024)));
    const cost_micro = Math.max(0, Math.round(totals.cost_micro * share));
    const revenueSkew = 1 + (fixture.roiSkew ?? 0) + (hashUnit(index + 15) - 0.5) * 0.08;
    const revenue_micro = Math.max(0, Math.round(totals.revenue_micro * share * revenueSkew));
    const profit_micro = revenue_micro - cost_micro;
    const roi_pct = cost_micro > 0 ? (profit_micro / cost_micro) * 100 : 0;
    return enrichEconomicsRow({
      id: fixture.id,
      name: fixture.name,
      clicks,
      unique_clicks,
      conversions,
      cost_micro,
      revenue_micro,
      profit_micro,
      roi_pct,
    });
  });
}

function sumBreakdownRows(rows: DashboardBreakdownRow[]): DashboardBreakdownTotals {
  const totals = rows.reduce<DashboardBreakdownTotals>(
    (acc, row) => ({
      clicks: (acc.clicks ?? 0) + (row.clicks ?? 0),
      unique_clicks: (acc.unique_clicks ?? 0) + (row.unique_clicks ?? 0),
      conversions: (acc.conversions ?? 0) + (row.conversions ?? 0),
      cost_micro: (acc.cost_micro ?? 0) + (row.cost_micro ?? 0),
      revenue_micro: (acc.revenue_micro ?? 0) + (row.revenue_micro ?? 0),
      profit_micro: (acc.profit_micro ?? 0) + (row.profit_micro ?? 0),
    }),
    {
      clicks: 0,
      unique_clicks: 0,
      conversions: 0,
      cost_micro: 0,
      revenue_micro: 0,
      profit_micro: 0,
    },
  );
  const costMicro = totals.cost_micro ?? 0;
  const profitMicro = totals.profit_micro ?? 0;
  const roi_pct = costMicro > 0 ? (profitMicro / costMicro) * 100 : 0;
  return enrichEconomicsTotals({ ...totals, roi_pct });
}

function buildBreakdownTable(
  fixtures: { id: string; name: string; share: number; roiSkew?: number }[],
  totals: AggregateTotals,
): DashboardBreakdownTable {
  const rows = buildBreakdownRows(fixtures, totals);
  return {
    rows,
    totals: sumBreakdownRows(rows),
    truncated: false,
    total: rows.length,
  };
}

function clickId(seed: number): string {
  const part = Math.floor(hashUnit(seed) * 0xffff_ffff).toString(16).padStart(8, '0');
  return `clk_${part}${seed.toString(16).padStart(4, '0')}`;
}

function buildRecentClicks(rangeEnd: Date): ClickLogEvent[] {
  const events: ClickLogEvent[] = [];
  for (let index = 0; index < 10; index += 1) {
    const hoursAgo = Math.round(index * 53 + hashUnit(index * 31) * 140);
    const createdAt = new Date(rangeEnd.getTime() - hoursAgo * 3_600_000);
    const campaign = CAMPAIGN_FIXTURES[index % CAMPAIGN_FIXTURES.length];
    const hasRevenue = hashUnit(index + 8) > 0.28;
    events.push({
      event_type: 'click',
      click_id: clickId(index + 41),
      campaign_id: campaign.id,
      placement_id: `plc_${(index + 3).toString().padStart(4, '0')}`,
      created_at: createdAt.toISOString(),
      country: RECENT_CLICK_COUNTRIES[index % RECENT_CLICK_COUNTRIES.length],
      sub1: RECENT_CLICK_SOURCES[index % RECENT_CLICK_SOURCES.length],
      goal_name: RECENT_CLICK_GOALS[index % RECENT_CLICK_GOALS.length],
      attributed_cost_micro: usdToMicro(0.09 + hashUnit(index + 2) * 0.19),
      revenue_micro: hasRevenue ? usdToMicro(12 + hashUnit(index + 19) * 38) : 0,
      inbound_status: 'accepted',
    });
  }
  return events;
}

function parsePeriodDate(value: string | undefined): Date | undefined {
  if (!value) {
    return undefined;
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return undefined;
  }
  return parsed;
}

function resolveMockRange(portfolio: BuyerPortfolio): { from: Date; to: Date } {
  const from = parsePeriodDate(portfolio.period?.from) ?? DASHBOARD_MOCK_DEFAULT_FROM;
  const to = parsePeriodDate(portfolio.period?.to) ?? DASHBOARD_MOCK_DEFAULT_TO;
  if (from <= to) {
    return { from, to };
  }
  return { from: to, to: from };
}

export function buildDashboardMockPortfolio(portfolio: BuyerPortfolio): BuyerPortfolio {
  const { from, to } = resolveMockRange(portfolio);
  const series = buildDashboardMockSeries(from, to);
  const totals = aggregateSeries(series);

  return {
    ...portfolio,
    period: {
      ...portfolio.period,
      from: from.toISOString(),
      to: to.toISOString(),
    },
    clicks_7d: totals.clicks,
    unique_clicks_7d: totals.unique_clicks,
    impressions_7d: Math.round(totals.clicks * (1.12 + hashUnit(totals.clicks) * 0.08)),
    kpis: {
      ...portfolio.kpis,
      conversions: totals.conversions,
      unique_clicks: totals.unique_clicks,
      cost_micro: totals.cost_micro,
      spend_micro: totals.cost_micro,
      revenue_micro: totals.revenue_micro,
      profit_micro: totals.profit_micro,
      roi_pct: totals.roi_pct,
      cpc_micro: computeCpcMicro(totals.cost_micro, totals.clicks),
      cpa_micro: computeCpaMicro(totals.cost_micro, totals.conversions),
      cr_pct: computeCrPct(totals.conversions, totals.clicks),
      epc_micro: computeEpcMicro(totals.revenue_micro, totals.clicks),
      freshness: { stale: false, label: 'Mock preview' },
    },
    series,
    breakdowns: {
      campaigns: buildBreakdownTable(CAMPAIGN_FIXTURES, totals),
      landers: buildBreakdownTable(LANDER_FIXTURES, totals),
      offers: buildBreakdownTable(OFFER_FIXTURES, totals),
      sources: buildBreakdownTable(SOURCE_FIXTURES, totals),
    },
    recent_clicks: buildRecentClicks(to),
  };
}

export function resolveDashboardChartSeries(
  series: DashboardSeriesPoint[] | undefined,
): DashboardSeriesPoint[] {
  if (series && series.length > 0) {
    return series;
  }
  if (isDashboardChartMockEnabled()) {
    return buildDashboardMockSeries();
  }
  return [];
}

export function resolveBuyerDashboardPortfolio(portfolio: BuyerPortfolio): BuyerPortfolio {
  if (shouldFillDashboardDemoData(portfolio)) {
    return buildDashboardMockPortfolio(portfolio);
  }
  return portfolio;
}
