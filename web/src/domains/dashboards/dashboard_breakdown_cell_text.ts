import {
  profitToneClassFromMicro,
  roiToneClassFromRate,
} from '@/domains/campaigns/list/campaign_list_tone';
import type { DashboardBreakdownRow } from '@/domains/dashboards/buyer_dashboard_types';
import {
  formatDashboardCrPct,
  formatDashboardUsdFromMicro,
  resolveBreakdownProfitMicro,
  resolveBreakdownRoiPct,
} from '@/domains/dashboards/dashboard_format';
import { formatRoi } from '@/domains/dashboards/dashboard_metrics';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import { displayCount } from '@/lib/display';

export function dashboardBreakdownMetricToneClass(
  columnId: DashboardBreakdownColumnId,
  row: DashboardBreakdownRow,
  isTotal = false
): string | undefined {
  if (isTotal) {
    return undefined;
  }
  if (columnId === 'profit') {
    return profitToneClassFromMicro(resolveBreakdownProfitMicro(row));
  }
  if (columnId === 'roi') {
    const roiPct = resolveBreakdownRoiPct(row) ?? 0;
    const roiText = formatRoi(roiPct);
    return roiToneClassFromRate({
      text: roiText,
      valPct: roiPct,
      isZero: roiPct === 0 || roiText === '-',
    });
  }
  return undefined;
}

export function formatDashboardBreakdownCellText(
  columnId: DashboardBreakdownColumnId,
  row: DashboardBreakdownRow,
  isTotal = false
): string {
  switch (columnId) {
    case 'name':
      return isTotal ? 'Total' : (row.name ?? '');
    case 'clicks':
      return displayCount(row.clicks);
    case 'unique_clicks':
      return displayCount(row.unique_clicks);
    case 'conversions':
      return displayCount(row.conversions);
    case 'cost':
      return formatDashboardUsdFromMicro(row.cost_micro);
    case 'revenue':
      return formatDashboardUsdFromMicro(row.revenue_micro);
    case 'profit':
      return formatDashboardUsdFromMicro(resolveBreakdownProfitMicro(row));
    case 'cpc':
      return formatDashboardUsdFromMicro(row.cpc_micro);
    case 'cpa':
      return formatDashboardUsdFromMicro(row.cpa_micro);
    case 'cr':
      return formatDashboardCrPct(row.cr_pct);
    case 'epc':
      return formatDashboardUsdFromMicro(row.epc_micro);
    case 'roi':
      return formatRoi(resolveBreakdownRoiPct(row));
    default:
      return '';
  }
}
