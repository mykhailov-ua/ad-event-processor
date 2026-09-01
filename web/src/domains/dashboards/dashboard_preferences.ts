import type { DashboardMetricId } from '@/domains/dashboards/dashboard_metrics';

export const BUYER_DASHBOARD_PREFS_STORAGE_KEY = 'buyer_dashboard_preferences_v2';

export type DashboardBreakdownEntityId = 'campaigns' | 'landers' | 'offers' | 'sources';

export type DashboardBreakdownColumnId =
  | 'name'
  | 'clicks'
  | 'unique_clicks'
  | 'conversions'
  | 'cost'
  | 'revenue'
  | 'profit'
  | 'cpc'
  | 'cpa'
  | 'cr'
  | 'epc'
  | 'roi';

export type DashboardRecentClickColumnId =
  | 'click_id'
  | 'created_at'
  | 'campaign_id'
  | 'country'
  | 'sub1'
  | 'placement_id'
  | 'goal_name'
  | 'cost'
  | 'revenue';

export type BuyerDashboardPreferences = {
  kpiMetrics: DashboardMetricId[];
  chartMetrics: DashboardMetricId[];
  breakdownEntities: DashboardBreakdownEntityId[];
  breakdownColumns: DashboardBreakdownColumnId[];
  recentClickColumns: DashboardRecentClickColumnId[];
};

export const ALL_KPI_METRIC_IDS: DashboardMetricId[] = [
  'clicks',
  'unique_clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
  'cpc',
  'cpa',
  'cr',
  'epc',
  'roi',
];

export const ALL_CHART_METRIC_IDS: DashboardMetricId[] = [
  'clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
];

export const ALL_BREAKDOWN_ENTITIES: DashboardBreakdownEntityId[] = [
  'campaigns',
  'landers',
  'offers',
  'sources',
];

export const ALL_BREAKDOWN_COLUMNS: DashboardBreakdownColumnId[] = [
  'name',
  'clicks',
  'unique_clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
  'cpc',
  'cpa',
  'cr',
  'epc',
  'roi',
];

export const ALL_RECENT_CLICK_COLUMNS: DashboardRecentClickColumnId[] = [
  'click_id',
  'created_at',
  'campaign_id',
  'country',
  'sub1',
  'placement_id',
  'goal_name',
  'cost',
  'revenue',
];

export const KPI_METRIC_LABELS: Record<DashboardMetricId, string> = {
  clicks: 'Clicks',
  unique_clicks: 'Unique clicks (campaign)',
  conversions: 'Conversions',
  cost: 'Cost',
  revenue: 'Revenue (confirmed)',
  profit: 'Profit/Loss (confirmed)',
  cpc: 'CPC',
  cpa: 'CPA',
  cr: 'CR',
  epc: 'EPC',
  roi: 'ROI (confirmed)',
};

export const BREAKDOWN_ENTITY_LABELS: Record<DashboardBreakdownEntityId, string> = {
  campaigns: 'Campaigns',
  landers: 'Landing pages',
  offers: 'Offers',
  sources: 'Sources',
};

export const BREAKDOWN_COLUMN_LABELS: Record<DashboardBreakdownColumnId, string> = {
  name: 'Name',
  clicks: 'Clicks',
  unique_clicks: 'Unique clicks (campaign)',
  conversions: 'Conversions',
  cost: 'Cost',
  revenue: 'Revenue',
  profit: 'Profit',
  cpc: 'CPC',
  cpa: 'CPA',
  cr: 'CR',
  epc: 'EPC',
  roi: 'ROI',
};

export const RECENT_CLICK_COLUMN_LABELS: Record<DashboardRecentClickColumnId, string> = {
  click_id: 'Event id',
  created_at: 'Date and time',
  campaign_id: 'Campaign',
  country: 'Country',
  sub1: 'Source',
  placement_id: 'Placement',
  goal_name: 'Goal',
  cost: 'Cost',
  revenue: 'Revenue',
};

export const DEFAULT_KPI_METRIC_IDS: DashboardMetricId[] = [
  'clicks',
  'unique_clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
  'roi',
];

export const DEFAULT_CHART_METRIC_IDS: DashboardMetricId[] = [
  'clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
];

export const DEFAULT_BASIC_BREAKDOWN_COLUMNS: DashboardBreakdownColumnId[] = [
  'name',
  'clicks',
  'unique_clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
  'roi',
];

export function defaultBuyerDashboardPreferences(): BuyerDashboardPreferences {
  return {
    kpiMetrics: [...DEFAULT_KPI_METRIC_IDS],
    chartMetrics: [...DEFAULT_CHART_METRIC_IDS],
    breakdownEntities: [...ALL_BREAKDOWN_ENTITIES],
    breakdownColumns: [...DEFAULT_BASIC_BREAKDOWN_COLUMNS],
    recentClickColumns: ['click_id', 'created_at', 'campaign_id', 'country', 'sub1'],
  };
}

function normalizeSelection<T extends string>(
  value: unknown,
  allowed: readonly T[],
  fallback: readonly T[],
): T[] {
  if (!Array.isArray(value)) {
    return [...fallback];
  }
  const allowedSet = new Set(allowed);
  const selected = value.filter((item): item is T => typeof item === 'string' && allowedSet.has(item as T));
  return selected.length > 0 ? selected : [...fallback];
}

export function parseBuyerDashboardPreferences(raw: string | null): BuyerDashboardPreferences {
  const defaults = defaultBuyerDashboardPreferences();
  if (!raw?.trim()) {
    return defaults;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<BuyerDashboardPreferences>;
    return {
      kpiMetrics: normalizeSelection(parsed.kpiMetrics, ALL_KPI_METRIC_IDS, defaults.kpiMetrics),
      chartMetrics: normalizeSelection(parsed.chartMetrics, ALL_CHART_METRIC_IDS, defaults.chartMetrics),
      breakdownEntities: normalizeSelection(
        parsed.breakdownEntities,
        ALL_BREAKDOWN_ENTITIES,
        defaults.breakdownEntities,
      ),
      breakdownColumns: normalizeSelection(
        parsed.breakdownColumns,
        ALL_BREAKDOWN_COLUMNS,
        defaults.breakdownColumns,
      ),
      recentClickColumns: normalizeSelection(
        parsed.recentClickColumns,
        ALL_RECENT_CLICK_COLUMNS,
        defaults.recentClickColumns,
      ),
    };
  } catch {
    return defaults;
  }
}

export function loadBuyerDashboardPreferences(): BuyerDashboardPreferences {
  if (typeof window === 'undefined') {
    return defaultBuyerDashboardPreferences();
  }
  return parseBuyerDashboardPreferences(window.localStorage.getItem(BUYER_DASHBOARD_PREFS_STORAGE_KEY));
}

export function saveBuyerDashboardPreferences(prefs: BuyerDashboardPreferences): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(BUYER_DASHBOARD_PREFS_STORAGE_KEY, JSON.stringify(prefs));
}
