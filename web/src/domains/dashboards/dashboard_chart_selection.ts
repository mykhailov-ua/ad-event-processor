import type { DashboardMetricId } from '@/domains/dashboards/dashboard_metrics';

const STORAGE_KEY = 'buyer_dashboard_chart_metrics_v1';

export const DEFAULT_CHART_METRIC_IDS: DashboardMetricId[] = ['clicks', 'conversions', 'revenue'];

const ALL_CHART_METRIC_IDS: DashboardMetricId[] = [
  'clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
];

export function parseChartMetricSelection(raw: string | null): DashboardMetricId[] {
  if (!raw?.trim()) {
    return DEFAULT_CHART_METRIC_IDS;
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return DEFAULT_CHART_METRIC_IDS;
    }
    const allowed = new Set(ALL_CHART_METRIC_IDS);
    const selected = parsed.filter(
      (item): item is DashboardMetricId => typeof item === 'string' && allowed.has(item as DashboardMetricId),
    );
    return selected.length > 0 ? selected : DEFAULT_CHART_METRIC_IDS;
  } catch {
    return DEFAULT_CHART_METRIC_IDS;
  }
}

export function loadChartMetricSelection(): DashboardMetricId[] {
  if (typeof window === 'undefined') {
    return DEFAULT_CHART_METRIC_IDS;
  }
  return parseChartMetricSelection(window.localStorage.getItem(STORAGE_KEY));
}

export function saveChartMetricSelection(ids: DashboardMetricId[]): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
}

export function toggleChartMetric(
  current: DashboardMetricId[],
  id: DashboardMetricId,
): DashboardMetricId[] {
  if (current.includes(id)) {
    const next = current.filter((item) => item !== id);
    return next.length > 0 ? next : current;
  }
  return [...current, id];
}
