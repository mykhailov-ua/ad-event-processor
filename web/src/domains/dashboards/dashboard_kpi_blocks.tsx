import type { DashboardMetricsBlock } from '@/domains/dashboards/dashboard_types';
import { ReportKpiGrid, type ReportKpiItem } from '@/domains/reports/report_kpi_grid';
import { formatDashboardRoiPct } from '@/lib/display_metrics';
import { displayCount, displayMicro } from '@/lib/display';

type DashboardKpiBlocksProps = {
  kpis: DashboardMetricsBlock | undefined;
  stale: boolean;
  portfolioCounts?: {
    active?: number;
    paused?: number;
    archived?: number;
    impressions7d?: number;
    clicks7d?: number;
    overspendCount?: number;
  };
};

function staleAwareMicro(value: number | undefined, stale: boolean): string {
  if (stale && (value == null || value === 0)) {
    return '-';
  }
  return displayMicro(value);
}

function staleAwareRoi(value: number | undefined, stale: boolean): string {
  if (stale && (value == null || !Number.isFinite(value))) {
    return '-';
  }
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return formatDashboardRoiPct(value);
}

export function DashboardKpiBlocks({ kpis, stale, portfolioCounts }: DashboardKpiBlocksProps) {
  const items: ReportKpiItem[] = [
    { label: 'Spend', value: displayMicro(kpis?.spend_micro) },
    { label: 'Revenue', value: staleAwareMicro(kpis?.revenue_micro, stale) },
    { label: 'Profit', value: staleAwareMicro(kpis?.profit_micro, stale) },
    { label: 'ROI', value: staleAwareRoi(kpis?.roi_pct, stale) },
    { label: 'Conversions', value: displayCount(kpis?.conversions) },
    {
      label: 'Unique clicks',
      value: stale
        ? kpis?.unique_clicks
          ? displayCount(kpis.unique_clicks)
          : '-'
        : displayCount(kpis?.unique_clicks),
    },
  ];

  if (portfolioCounts) {
    items.push(
      { label: 'Active campaigns', value: displayCount(portfolioCounts.active) },
      { label: 'Paused', value: displayCount(portfolioCounts.paused) },
      { label: 'Impressions', value: displayCount(portfolioCounts.impressions7d) },
      { label: 'Clicks', value: displayCount(portfolioCounts.clicks7d) }
    );
    if (portfolioCounts.overspendCount != null && portfolioCounts.overspendCount > 0) {
      items.push({
        label: 'Overspend risk',
        value: displayCount(portfolioCounts.overspendCount),
      });
    }
  }

  return (
    <div data-role="kpi-blocks" data-testid="dashboard-kpis">
      <ReportKpiGrid items={items} />
    </div>
  );
}
