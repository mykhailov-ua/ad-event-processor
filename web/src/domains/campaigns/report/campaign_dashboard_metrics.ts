import type { DashboardKpiTile } from '@/domains/dashboards/dashboard_kpi_strip';
import type { DashboardMetricId } from '@/domains/dashboards/dashboard_metrics';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import {
  formatDashboardCrPct,
  formatDashboardRoiPct,
  formatDashboardUsdFromMicro,
} from '@/domains/dashboards/dashboard_format';
import { displayCount } from '@/lib/display';

export type CampaignDashboardKpis = {
  clicks?: number;
  unique_clicks?: number;
  conversions?: number;
  cost_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
  cpc_micro?: number;
  cpa_micro?: number;
  epc_micro?: number;
  cr_pct?: number;
  roi_pct?: number;
};

export type CampaignDashboardMetricConfig = {
  id: DashboardMetricId;
  label: string;
  accent: DashboardKpiTile['accent'];
  value: (kpis: CampaignDashboardKpis) => string;
};

export const CAMPAIGN_DASHBOARD_KPI_METRICS: CampaignDashboardMetricConfig[] = [
  { id: 'clicks', label: 'Clicks', accent: 1, value: (kpis) => displayCount(kpis.clicks) },
  { id: 'unique_clicks', label: 'Unique clicks', accent: 2, value: (kpis) => displayCount(kpis.unique_clicks) },
  { id: 'conversions', label: 'Conversions', accent: 3, value: (kpis) => displayCount(kpis.conversions) },
  { id: 'cost', label: 'Cost', accent: 4, value: (kpis) => formatDashboardUsdFromMicro(kpis.cost_micro) },
  { id: 'revenue', label: 'Revenue', accent: 5, value: (kpis) => formatDashboardUsdFromMicro(kpis.revenue_micro) },
  { id: 'profit', label: 'Profit', accent: 1, value: (kpis) => formatDashboardUsdFromMicro(kpis.profit_micro) },
  { id: 'cpc', label: 'CPC', accent: 4, value: (kpis) => formatDashboardUsdFromMicro(kpis.cpc_micro) },
  { id: 'cpa', label: 'CPA', accent: 5, value: (kpis) => formatDashboardUsdFromMicro(kpis.cpa_micro) },
  { id: 'cr', label: 'CR', accent: 3, value: (kpis) => formatDashboardCrPct(kpis.cr_pct) },
  { id: 'epc', label: 'EPC', accent: 2, value: (kpis) => formatDashboardUsdFromMicro(kpis.epc_micro) },
  { id: 'roi', label: 'ROI', accent: 2, value: (kpis) => formatDashboardRoiPct(kpis.roi_pct ?? 0) },
];

export function buildCampaignDashboardKpiTiles(
  kpis: CampaignDashboardKpis | undefined,
  metricIds: DashboardMetricId[],
): DashboardKpiTile[] {
  const metricById = new Map(
    CAMPAIGN_DASHBOARD_KPI_METRICS.map((metric) => [metric.id, metric]),
  );
  return metricIds
    .map((id) => metricById.get(id))
    .filter((metric): metric is CampaignDashboardMetricConfig => metric != null)
    .map((metric) => ({
      id: metric.id,
      label: metric.label,
      value: metric.value(kpis ?? {}),
      accent: metric.accent,
    }));
}

export const DEFAULT_CAMPAIGN_DASHBOARD_KPI_METRICS: DashboardMetricId[] = [
  'clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
  'roi',
];

export const DEFAULT_CAMPAIGN_DASHBOARD_CHART_METRICS: DashboardMetricId[] = [
  'clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
];

export const DEFAULT_CAMPAIGN_DASHBOARD_BREAKDOWN_COLUMNS: DashboardBreakdownColumnId[] = [
  'name',
  'clicks',
  'conversions',
  'cost',
  'revenue',
  'profit',
  'roi',
];
