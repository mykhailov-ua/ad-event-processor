import { CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX } from '@/domains/campaigns/list/campaign_list_columns';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import type { DashboardRecentClickColumnId } from '@/domains/dashboards/dashboard_preferences';

export const DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX: Record<DashboardBreakdownColumnId, number> = {
  name: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.name,
  clicks: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.clicks,
  unique_clicks: 176,
  conversions: 96,
  cost: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.cost,
  revenue: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.revenue,
  profit: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.profit,
  cpc: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.cpc,
  cpa: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.cpa,
  cr: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.cr,
  epc: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.epc,
  roi: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.roi,
};

const DASHBOARD_BREAKDOWN_COLUMN_MAX_WIDTH_PX: Partial<Record<DashboardBreakdownColumnId, number>> =
  {
    name: 640,
    unique_clicks: 320,
    conversions: 160,
  };

const DASHBOARD_BREAKDOWN_COLUMN_USER_RESIZE_MAX_WIDTH_PX = 480;

export const DASHBOARD_RECENT_CLICK_COLUMN_WIDTH_PX: Record<DashboardRecentClickColumnId, number> =
  {
    click_id: 220,
    created_at: 168,
    campaign_id: 200,
    country: 72,
    sub1: 120,
    placement_id: 120,
    goal_name: 120,
    cost: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.cost,
    revenue: CAMPAIGN_LIST_COLUMN_MIN_WIDTH_PX.revenue,
  };

const DASHBOARD_RECENT_CLICK_COLUMN_MAX_WIDTH_PX: Partial<
  Record<DashboardRecentClickColumnId, number>
> = {
  click_id: 420,
  campaign_id: 420,
  created_at: 280,
  sub1: 320,
  placement_id: 320,
  goal_name: 320,
};

const CELL_HORIZONTAL_PADDING_PX = 32;
const BODY_TOOLS_GUTTER_PX = 28;
const NAME_ROW_MENU_PX = 28;
const NAME_ROW_MENU_GAP_PX = 16;

function estimateTextWidthPx(text: string): number {
  return Math.ceil(text.length * 7);
}

export function probeDashboardNameColumnWidthPx(
  labels: readonly string[],
  minWidth: number
): number {
  let longest = 'Total';
  for (const label of labels) {
    if (label.length > longest.length) {
      longest = label;
    }
  }
  return Math.max(
    minWidth,
    estimateTextWidthPx(longest) +
      CELL_HORIZONTAL_PADDING_PX +
      NAME_ROW_MENU_PX +
      NAME_ROW_MENU_GAP_PX
  );
}

export function resolveDashboardBreakdownColumnWidthPx(
  columnId: DashboardBreakdownColumnId,
  overrides: Readonly<Partial<Record<DashboardBreakdownColumnId, number>>>,
  nameProbeWidthPx?: number
): number {
  const override = overrides[columnId];
  if (override != null && Number.isFinite(override) && override > 0) {
    return clampDashboardBreakdownColumnWidthPx(columnId, override);
  }
  if (columnId === 'name' && nameProbeWidthPx != null) {
    return clampDashboardBreakdownColumnWidthPx(columnId, nameProbeWidthPx);
  }
  return DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX[columnId];
}

export function clampDashboardBreakdownColumnWidthPx(
  columnId: DashboardBreakdownColumnId,
  widthPx: number
): number {
  const minWidth = DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX[columnId];
  const maxWidth =
    DASHBOARD_BREAKDOWN_COLUMN_MAX_WIDTH_PX[columnId] ??
    DASHBOARD_BREAKDOWN_COLUMN_USER_RESIZE_MAX_WIDTH_PX;
  return Math.min(maxWidth, Math.max(minWidth, Math.trunc(widthPx)));
}

export function clampUserResizedDashboardBreakdownColumnWidthPx(
  columnId: DashboardBreakdownColumnId,
  widthPx: number
): number {
  return clampDashboardBreakdownColumnWidthPx(columnId, widthPx);
}

export function dashboardBreakdownTableWidthPx(
  columns: DashboardBreakdownColumnId[],
  widths: Readonly<Record<DashboardBreakdownColumnId, number>>
): number {
  return columns.reduce(
    (sum, columnId) => sum + (widths[columnId] ?? DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX[columnId]),
    0
  );
}

export function isDashboardBreakdownNumericColumn(columnId: DashboardBreakdownColumnId): boolean {
  return columnId !== 'name';
}

export function isDashboardBreakdownColumnResizable(
  columnId: DashboardBreakdownColumnId,
  columns: readonly DashboardBreakdownColumnId[]
): boolean {
  if (columns.length === 0) {
    return false;
  }
  return columns[columns.length - 1] !== columnId;
}

export function resolveDashboardRecentClickColumnWidthPx(
  columnId: DashboardRecentClickColumnId,
  overrides: Readonly<Partial<Record<DashboardRecentClickColumnId, number>>>
): number {
  const override = overrides[columnId];
  if (override != null && Number.isFinite(override) && override > 0) {
    return clampDashboardRecentClickColumnWidthPx(columnId, override);
  }
  return DASHBOARD_RECENT_CLICK_COLUMN_WIDTH_PX[columnId];
}

export function clampDashboardRecentClickColumnWidthPx(
  columnId: DashboardRecentClickColumnId,
  widthPx: number
): number {
  const minWidth = DASHBOARD_RECENT_CLICK_COLUMN_WIDTH_PX[columnId];
  const maxWidth =
    DASHBOARD_RECENT_CLICK_COLUMN_MAX_WIDTH_PX[columnId] ??
    DASHBOARD_BREAKDOWN_COLUMN_USER_RESIZE_MAX_WIDTH_PX;
  return Math.min(maxWidth, Math.max(minWidth, Math.trunc(widthPx)));
}

export function clampUserResizedDashboardRecentClickColumnWidthPx(
  columnId: DashboardRecentClickColumnId,
  widthPx: number
): number {
  return clampDashboardRecentClickColumnWidthPx(columnId, widthPx);
}

export function dashboardRecentClickTableWidthPx(
  columns: DashboardRecentClickColumnId[],
  widths: Readonly<Record<DashboardRecentClickColumnId, number>>
): number {
  return columns.reduce(
    (sum, columnId) => sum + (widths[columnId] ?? DASHBOARD_RECENT_CLICK_COLUMN_WIDTH_PX[columnId]),
    0
  );
}

export function isDashboardRecentClickNumericColumn(
  columnId: DashboardRecentClickColumnId
): boolean {
  return columnId === 'cost' || columnId === 'revenue';
}

export function isDashboardRecentClickCopyColumn(columnId: DashboardRecentClickColumnId): boolean {
  return columnId === 'click_id' || columnId === 'campaign_id';
}

export function isDashboardRecentClickColumnResizable(
  columnId: DashboardRecentClickColumnId,
  columns: readonly DashboardRecentClickColumnId[]
): boolean {
  if (columns.length === 0) {
    return false;
  }
  return columns[columns.length - 1] !== columnId;
}
