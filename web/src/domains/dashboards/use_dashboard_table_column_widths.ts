// L3 dashboard table widths: localStorage overrides per scope; name column uses probe width from row labels.
import { useCallback, useDeferredValue, useMemo, useState } from 'react';

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
  const deferredColumns = useDeferredValue(columns);
  const deferredNameLabels = useDeferredValue(nameLabels);
  const deferredProbeRows = useDeferredValue(widthProbeTable?.rows ?? []);
  const deferredProbeTotals = useDeferredValue(widthProbeTable?.totals);
  const [widthOverrides, setWidthOverrides] = useState(() => loadDashboardTableColumnWidths(scope));

  const nameProbeWidthPx = useMemo(
    () =>
      probeDashboardNameColumnWidthPx(
        deferredNameLabels,
        DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX.name
      ),
    [deferredNameLabels]
  );

  const dataProbeWidths = useMemo(() => {
    const widths = {} as Partial<Record<DashboardBreakdownColumnId, number>>;
    for (const columnId of deferredColumns) {
      if (columnId === 'name') {
        continue;
      }
      widths[columnId] = probeDashboardBreakdownDataColumnWidthPx(
        columnId,
        deferredProbeRows,
        deferredProbeTotals
      );
    }
    return widths;
  }, [deferredColumns, deferredProbeRows, deferredProbeTotals]);

  const columnWidths = useMemo((): Record<DashboardBreakdownColumnId, number> => {
    const widths = {} as Record<DashboardBreakdownColumnId, number>;
    for (const columnId of deferredColumns) {
      widths[columnId] = resolveDashboardBreakdownColumnWidthPx(
        columnId,
        widthOverrides as Partial<Record<DashboardBreakdownColumnId, number>>,
        columnId === 'name' ? nameProbeWidthPx : undefined,
        dataProbeWidths[columnId]
      );
    }
    return widths;
  }, [dataProbeWidths, deferredColumns, nameProbeWidthPx, widthOverrides]);

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
  const deferredColumns = useDeferredValue(columns);
  const scope: DashboardTableWidthScope = 'recent_clicks';
  const [widthOverrides, setWidthOverrides] = useState(() => loadDashboardTableColumnWidths(scope));

  const columnWidths = useMemo((): Record<DashboardRecentClickColumnId, number> => {
    const widths = {} as Record<DashboardRecentClickColumnId, number>;
    for (const columnId of deferredColumns) {
      widths[columnId] = resolveDashboardRecentClickColumnWidthPx(
        columnId,
        widthOverrides as Partial<Record<DashboardRecentClickColumnId, number>>
      );
    }
    return widths;
  }, [deferredColumns, widthOverrides]);

  const handleColumnWidthCommit = useCallback(
    (columnId: DashboardRecentClickColumnId, widthPx: number) => {
      const nextWidth = clampUserResizedDashboardRecentClickColumnWidthPx(columnId, widthPx);
      const nextOverrides = saveDashboardTableColumnWidth(scope, columnId, nextWidth);
      setWidthOverrides(nextOverrides);
    },
    []
  );

  return { columnWidths, handleColumnWidthCommit };
}
