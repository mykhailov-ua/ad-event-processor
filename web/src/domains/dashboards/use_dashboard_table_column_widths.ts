// dashboard table widths: localStorage overrides per scope; name column uses probe width from row labels.
import { useCallback, useMemo, useState } from 'react';

import type { DashboardBreakdownTable } from '@/domains/dashboards/buyer_dashboard_types';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import type { DashboardRecentClickColumnId } from '@/domains/dashboards/dashboard_preferences';
import {
  clampUserResizedDashboardBreakdownColumnWidthPx,
  clampUserResizedDashboardRecentClickColumnWidthPx,
  DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX,
  DASHBOARD_RECENT_CLICK_COLUMN_WIDTH_PX,
  probeDashboardBreakdownDataColumnWidthPx,
  probeDashboardNameColumnWidthPx,
  resolveDashboardBreakdownColumnWidthPx,
  resolveDashboardRecentClickColumnWidthPx,
} from '@/domains/dashboards/dashboard_table_column_widths';
import {
  loadDashboardTableColumnWidths,
  saveDashboardTableColumnWidth,
  type DashboardTableWidthScope,
} from '@/domains/dashboards/dashboard_table_column_prefs';

export function useDashboardBreakdownColumnWidths(
  scope: DashboardTableWidthScope,
  columns: readonly DashboardBreakdownColumnId[],
  nameLabels: readonly string[],
  widthProbeTable?: DashboardBreakdownTable
) {
  const [widthOverrides, setWidthOverrides] = useState(() => loadDashboardTableColumnWidths(scope));

  const nameProbeWidthPx = useMemo(
    () => probeDashboardNameColumnWidthPx(nameLabels, DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX.name),
    [nameLabels]
  );

  const dataProbeWidths = useMemo(() => {
    const widths = {} as Partial<Record<DashboardBreakdownColumnId, number>>;
    for (const columnId of columns) {
      if (columnId === 'name') {
        continue;
      }
      widths[columnId] = probeDashboardBreakdownDataColumnWidthPx(
        columnId,
        widthProbeTable?.rows ?? [],
        widthProbeTable?.totals
      );
    }
    return widths;
  }, [columns, widthProbeTable?.rows, widthProbeTable?.totals]);

  const columnWidths = useMemo((): Record<DashboardBreakdownColumnId, number> => {
    const widths = {} as Record<DashboardBreakdownColumnId, number>;
    for (const columnId of columns) {
      widths[columnId] = resolveDashboardBreakdownColumnWidthPx(
        columnId,
        widthOverrides as Partial<Record<DashboardBreakdownColumnId, number>>,
        columnId === 'name' ? nameProbeWidthPx : undefined,
        dataProbeWidths[columnId]
      );
    }
    return widths;
  }, [columns, dataProbeWidths, nameProbeWidthPx, widthOverrides]);

  const handleColumnWidthCommit = useCallback(
    (columnId: DashboardBreakdownColumnId, widthPx: number) => {
      const nextWidth = clampUserResizedDashboardBreakdownColumnWidthPx(columnId, widthPx);
      const nextOverrides = saveDashboardTableColumnWidth(scope, columnId, nextWidth);
      setWidthOverrides(nextOverrides);
    },
    [scope]
  );

  return { columnWidths, handleColumnWidthCommit };
}

export function useDashboardRecentClickColumnWidths(
  columns: readonly DashboardRecentClickColumnId[]
) {
  const scope: DashboardTableWidthScope = 'recent_clicks';
  const [widthOverrides, setWidthOverrides] = useState(() => loadDashboardTableColumnWidths(scope));

  const columnWidths = useMemo((): Record<DashboardRecentClickColumnId, number> => {
    const widths = {} as Record<DashboardRecentClickColumnId, number>;
    for (const columnId of columns) {
      widths[columnId] = resolveDashboardRecentClickColumnWidthPx(
        columnId,
        widthOverrides as Partial<Record<DashboardRecentClickColumnId, number>>
      );
    }
    return widths;
  }, [columns, widthOverrides]);

  const handleColumnWidthCommit = useCallback(
    (columnId: DashboardRecentClickColumnId, widthPx: number) => {
      const nextWidth = clampUserResizedDashboardRecentClickColumnWidthPx(columnId, widthPx);
      const nextOverrides = saveDashboardTableColumnWidth(scope, columnId, nextWidth);
      setWidthOverrides(nextOverrides);
    },
    [scope]
  );

  return { columnWidths, handleColumnWidthCommit };
}
