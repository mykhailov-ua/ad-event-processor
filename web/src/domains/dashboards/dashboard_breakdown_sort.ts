import type { DashboardBreakdownRow } from '@/domains/dashboards/buyer_dashboard_types';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import {
  resolveBreakdownProfitMicro,
  resolveBreakdownRoiPct,
} from '@/domains/dashboards/dashboard_format';

export type DashboardBreakdownSortOrder = 'asc' | 'desc';

export type DashboardBreakdownSortState = {
  column: DashboardBreakdownColumnId;
  order: DashboardBreakdownSortOrder;
};

export const DEFAULT_CAMPAIGNS_BREAKDOWN_SORT: DashboardBreakdownSortState = {
  column: 'profit',
  order: 'asc',
};

const SORTABLE_BREAKDOWN_COLUMNS = new Set<DashboardBreakdownColumnId>([
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
]);

export function isDashboardBreakdownSortableColumn(columnId: DashboardBreakdownColumnId): boolean {
  return SORTABLE_BREAKDOWN_COLUMNS.has(columnId);
}

function breakdownSortValue(
  row: DashboardBreakdownRow,
  columnId: DashboardBreakdownColumnId
): number {
  switch (columnId) {
    case 'name':
      return 0;
    case 'clicks':
      return row.clicks ?? 0;
    case 'unique_clicks':
      return row.unique_clicks ?? 0;
    case 'conversions':
      return row.conversions ?? 0;
    case 'cost':
      return row.cost_micro ?? 0;
    case 'revenue':
      return row.revenue_micro ?? 0;
    case 'profit':
      return resolveBreakdownProfitMicro(row) ?? 0;
    case 'cpc':
      return row.cpc_micro ?? 0;
    case 'cpa':
      return row.cpa_micro ?? 0;
    case 'cr':
      return row.cr_pct ?? 0;
    case 'epc':
      return row.epc_micro ?? 0;
    case 'roi':
      return resolveBreakdownRoiPct(row) ?? 0;
    default:
      return 0;
  }
}

function compareBreakdownRowNames(
  left: DashboardBreakdownRow,
  right: DashboardBreakdownRow
): number {
  const leftName = (left.name ?? '').toLocaleLowerCase();
  const rightName = (right.name ?? '').toLocaleLowerCase();
  return leftName.localeCompare(rightName);
}

export function compareDashboardBreakdownRows(
  left: DashboardBreakdownRow,
  right: DashboardBreakdownRow,
  state: DashboardBreakdownSortState
): number {
  if (state.column === 'name') {
    const byName = compareBreakdownRowNames(left, right);
    return state.order === 'desc' ? -byName : byName;
  }

  const leftValue = breakdownSortValue(left, state.column);
  const rightValue = breakdownSortValue(right, state.column);
  if (leftValue !== rightValue) {
    return state.order === 'desc' ? rightValue - leftValue : leftValue - rightValue;
  }

  const byName = compareBreakdownRowNames(left, right);
  return state.order === 'desc' ? -byName : byName;
}

export function sortDashboardBreakdownRows(
  rows: readonly DashboardBreakdownRow[],
  state: DashboardBreakdownSortState
): DashboardBreakdownRow[] {
  return [...rows].sort((left, right) => compareDashboardBreakdownRows(left, right, state));
}

export function toggleDashboardBreakdownSort(
  state: DashboardBreakdownSortState,
  column: DashboardBreakdownColumnId
): DashboardBreakdownSortState {
  if (state.column === column) {
    return { column, order: state.order === 'asc' ? 'desc' : 'asc' };
  }
  return { column, order: column === 'profit' ? 'asc' : 'desc' };
}
