export const DASHBOARD_TABLE_COLUMN_WIDTHS_STORAGE_KEY = 'buyer_dashboard_table_column_widths_v1';

export type DashboardTableWidthScope =
  | 'campaigns'
  | 'landers'
  | 'offers'
  | 'sources'
  | 'recent_clicks';

export type DashboardTableColumnWidthStore = Partial<
  Record<DashboardTableWidthScope, Partial<Record<string, number>>>
>;

function readStore(): DashboardTableColumnWidthStore {
  if (typeof window === 'undefined') {
    return {};
  }
  try {
    const raw = window.localStorage.getItem(DASHBOARD_TABLE_COLUMN_WIDTHS_STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw) as unknown;
    if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {};
    }
    return parsed as DashboardTableColumnWidthStore;
  } catch {
    return {};
  }
}

function writeStore(store: DashboardTableColumnWidthStore) {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(DASHBOARD_TABLE_COLUMN_WIDTHS_STORAGE_KEY, JSON.stringify(store));
}

export function loadDashboardTableColumnWidths(
  scope: DashboardTableWidthScope
): Partial<Record<string, number>> {
  const store = readStore();
  return store[scope] ?? {};
}

export function saveDashboardTableColumnWidth(
  scope: DashboardTableWidthScope,
  columnId: string,
  widthPx: number
): Partial<Record<string, number>> {
  const store = readStore();
  const scopeWidths = { ...(store[scope] ?? {}), [columnId]: widthPx };
  writeStore({ ...store, [scope]: scopeWidths });
  return scopeWidths;
}

export type DashboardBreakdownScope = Exclude<DashboardTableWidthScope, 'recent_clicks'>;
