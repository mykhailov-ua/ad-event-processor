import type { DashboardKpiTile } from '@/domains/dashboards/dashboard_kpi_strip';
import type { BuyerPortfolio } from '@/domains/dashboards/buyer_dashboard_types';
import {
  derivePortfolioRoiPct,
  formatDashboardCrPct,
  formatDashboardRoiPct,
  formatDashboardUsdFromMicro,
  portfolioCostMicro,
  resolvePortfolioProfitMicro,
} from '@/domains/dashboards/dashboard_format';
import { displayCount } from '@/lib/display';

export type DashboardMetricId =
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

export type DashboardMetricConfig = {
  id: DashboardMetricId;
  label: string;
  axis: 'volume' | 'money' | 'percent';
  accent: DashboardKpiTile['accent'];
  color: string;
  seriesKey?: string;
};

export const DASHBOARD_KPI_METRICS: DashboardMetricConfig[] = [
  { id: 'clicks', label: 'Clicks', axis: 'volume', accent: 1, color: 'hsl(var(--chart-1))', seriesKey: 'clicks' },
  {
    id: 'unique_clicks',
    label: 'UC (campaign)',
    axis: 'volume',
    accent: 2,
    color: 'hsl(var(--chart-2))',
  },
  {
    id: 'conversions',
    label: 'Conversions',
    axis: 'volume',
    accent: 3,
    color: 'hsl(var(--chart-3))',
    seriesKey: 'conversions',
  },
  { id: 'cost', label: 'Cost', axis: 'money', accent: 4, color: 'hsl(var(--chart-4))', seriesKey: 'cost_micro' },
  {
    id: 'revenue',
    label: 'Revenue',
    axis: 'money',
    accent: 5,
    color: 'hsl(var(--chart-5))',
    seriesKey: 'revenue_micro',
  },
  { id: 'profit', label: 'Profit', axis: 'money', accent: 1, color: 'hsl(var(--chart-1))', seriesKey: 'profit_micro' },
  { id: 'cpc', label: 'CPC', axis: 'money', accent: 4, color: 'hsl(var(--chart-4))' },
  { id: 'cpa', label: 'CPA', axis: 'money', accent: 5, color: 'hsl(var(--chart-5))' },
  { id: 'cr', label: 'CR', axis: 'percent', accent: 3, color: 'hsl(var(--chart-3))' },
  { id: 'epc', label: 'EPC', axis: 'money', accent: 2, color: 'hsl(var(--chart-2))' },
  { id: 'roi', label: 'ROI', axis: 'percent', accent: 2, color: 'hsl(var(--chart-2))' },
];

export const DASHBOARD_CHART_METRICS = DASHBOARD_KPI_METRICS.filter((metric) => metric.seriesKey);

export type DashboardChartSeriesStyle = {
  id: DashboardMetricId;
  seriesKey: string;
  label: string;
  axis: 'volume' | 'money';
  stroke: string;
  fill: string;
};

export const DASHBOARD_VOLUME_AXIS_COLOR = 'hsla(204, 50%, 68%, 0.95)';
export const DASHBOARD_MONEY_AXIS_COLOR = 'hsla(24, 55%, 62%, 0.95)';

export const DASHBOARD_CHART_SERIES_STYLES: DashboardChartSeriesStyle[] = [
  {
    id: 'clicks',
    seriesKey: 'clicks',
    label: 'Clicks',
    axis: 'volume',
    stroke: 'hsla(204, 42%, 68%, 0.78)',
    fill: 'hsla(204, 42%, 68%, 0.12)',
  },
  {
    id: 'conversions',
    seriesKey: 'conversions',
    label: 'Conversions',
    axis: 'volume',
    stroke: 'hsla(158, 32%, 62%, 0.78)',
    fill: 'hsla(158, 32%, 62%, 0.12)',
  },
  {
    id: 'cost',
    seriesKey: 'cost_micro',
    label: 'Cost',
    axis: 'money',
    stroke: 'hsla(18, 46%, 66%, 0.78)',
    fill: 'hsla(18, 46%, 66%, 0.12)',
  },
  {
    id: 'revenue',
    seriesKey: 'revenue_micro',
    label: 'Revenue',
    axis: 'money',
    stroke: 'hsla(132, 30%, 60%, 0.78)',
    fill: 'hsla(132, 30%, 60%, 0.12)',
  },
  {
    id: 'profit',
    seriesKey: 'profit_micro',
    label: 'Profit',
    axis: 'money',
    stroke: 'hsla(268, 36%, 72%, 0.78)',
    fill: 'hsla(268, 36%, 72%, 0.12)',
  },
];

export function formatRoi(value: number | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return formatDashboardRoiPct(value);
}

function buildKpiTileValue(portfolio: BuyerPortfolio, metricId: DashboardMetricId): string {
  const kpis = portfolio.kpis;
  const uniqueClicks = kpis?.unique_clicks ?? portfolio.unique_clicks_7d;
  const roiPct = derivePortfolioRoiPct(portfolio);
  switch (metricId) {
    case 'clicks':
      return displayCount(portfolio.clicks_7d);
    case 'unique_clicks':
      return displayCount(uniqueClicks);
    case 'conversions':
      return displayCount(kpis?.conversions);
    case 'cost':
      return formatDashboardUsdFromMicro(portfolioCostMicro(portfolio));
    case 'revenue':
      return formatDashboardUsdFromMicro(kpis?.revenue_micro);
    case 'profit':
      return formatDashboardUsdFromMicro(resolvePortfolioProfitMicro(portfolio));
    case 'cpc':
      return formatDashboardUsdFromMicro(kpis?.cpc_micro);
    case 'cpa':
      return formatDashboardUsdFromMicro(kpis?.cpa_micro);
    case 'cr':
      return formatDashboardCrPct(kpis?.cr_pct);
    case 'epc':
      return formatDashboardUsdFromMicro(kpis?.epc_micro);
    case 'roi':
      return formatRoi(roiPct);
    default:
      return '';
  }
}

export function buildKpiTiles(
  portfolio: BuyerPortfolio,
  visibleMetricIds?: DashboardMetricId[],
): DashboardKpiTile[] {
  const metricIds = visibleMetricIds ?? DASHBOARD_KPI_METRICS.map((metric) => metric.id);
  const metricById = new Map(DASHBOARD_KPI_METRICS.map((metric) => [metric.id, metric]));

  return metricIds
    .map((metricId) => metricById.get(metricId))
    .filter((metric): metric is DashboardMetricConfig => metric != null)
    .map((metric) => ({
      id: metric.id,
      label: metric.label,
      value: buildKpiTileValue(portfolio, metric.id),
      accent: metric.accent,
    }));
}
