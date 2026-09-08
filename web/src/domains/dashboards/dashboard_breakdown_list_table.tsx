import { useCallback, useMemo, useRef, type ReactNode } from 'react';

import {
  campaignListCellContentClass,
  campaignListCellContentNumClass,
  campaignListHeaderShellClass,
  campaignListHeaderLabelClass,
  campaignListNameRowCellClass,
  campaignListNameRowTextClass,
  campaignListNameTextClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { campaignListRowDataAttributes } from '@/domains/campaigns/list/campaign_list_row_tone';
import {
  dashboardTableClass,
  dashboardTableSurfaceClass,
  dashboardTableTdClass,
  dashboardTableTfootTdClass,
  dashboardTableThClass,
} from '@/domains/dashboards/dashboard_classes';
import {
  dashboardBreakdownMetricToneClass,
  formatDashboardBreakdownCellText,
} from '@/domains/dashboards/dashboard_breakdown_cell_text';
import {
  isDashboardBreakdownSortableColumn,
  type DashboardBreakdownSortState,
  toggleDashboardBreakdownSort,
} from '@/domains/dashboards/dashboard_breakdown_sort';
import { CampaignListColumnResizeHandle } from '@/domains/campaigns/list/campaign_list_table_header_cell';
import type {
  DashboardBreakdownRow,
  DashboardBreakdownTable,
} from '@/domains/dashboards/buyer_dashboard_types';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import { BREAKDOWN_COLUMN_LABELS } from '@/domains/dashboards/dashboard_preferences';
import type { DashboardBreakdownScope } from '@/domains/dashboards/dashboard_table_column_prefs';
import {
  clampUserResizedDashboardBreakdownColumnWidthPx,
  DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX,
  isDashboardBreakdownColumnResizable,
} from '@/domains/dashboards/dashboard_table_column_widths';
import { useDashboardBreakdownColumnWidths } from '@/domains/dashboards/use_dashboard_table_column_widths';
import { DirectoryDataTableEngine } from '@/shell/directory_data_table/directory_data_table_engine';
import { SortableTableHead } from '@/shell/directory_table';
import { resolveDashboardBreakdownLayoutMaxWidths } from '@/shell/directory_data_table/layout';
import { useDirectoryColumnResize } from '@/shell/directory_column_resize';
import { cn } from '@/lib/utils';

export type DashboardBreakdownListTableProps = {
  scope: DashboardBreakdownScope;
  table: DashboardBreakdownTable | undefined;
  widthProbeTable?: DashboardBreakdownTable;
  columns: DashboardBreakdownColumnId[];
  nameLink?: (row: { id?: string; name?: string }) => ReactNode;
  fillContainer?: boolean;
  selectedRowId?: string | null;
  onRowSelect?: (row: DashboardBreakdownRow) => void;
  sortState?: DashboardBreakdownSortState;
  onSortChange?: (state: DashboardBreakdownSortState) => void;
};

type BreakdownCellContext = {
  row: DashboardBreakdownRow;
  isTotal?: boolean;
};

function renderBreakdownCell(
  columnId: DashboardBreakdownColumnId,
  ctx: BreakdownCellContext
): ReactNode {
  const { row } = ctx;
  if (columnId === 'name') {
    return ctx.isTotal ? 'Total' : row.name;
  }
  return formatDashboardBreakdownCellText(columnId, row, ctx.isTotal);
}

export function DashboardBreakdownListTable({
  scope,
  table,
  widthProbeTable,
  columns,
  nameLink,
  fillContainer = true,
  selectedRowId = null,
  onRowSelect,
  sortState,
  onSortChange,
}: DashboardBreakdownListTableProps) {
  const rows = table?.rows ?? [];
  const totals = table?.totals;
  const probeSource = widthProbeTable ?? table;
  const visibleColumns =
    columns.length > 0
      ? columns
      : (['name', 'clicks', 'conversions'] as DashboardBreakdownColumnId[]);
  const nameLabels = useMemo(
    () => ['Total', ...(probeSource?.rows ?? []).map((row) => row.name ?? '')],
    [probeSource?.rows]
  );
  const { columnWidths, handleColumnWidthCommit } = useDashboardBreakdownColumnWidths(
    scope,
    visibleColumns,
    nameLabels,
    probeSource
  );
  const tableRef = useRef<HTMLTableElement>(null);
  const colgroupRef = useRef<HTMLTableColElement>(null);
  const { startResize } = useDirectoryColumnResize({
    columnWidths,
    columns: visibleColumns,
    minWidthPx: (columnId) => DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX[columnId],
    clampUserWidth: clampUserResizedDashboardBreakdownColumnWidthPx,
    onColumnWidthCommit: handleColumnWidthCommit,
    colgroupRef,
    tableRef,
  });
  const resolveRowKey = useCallback((row: DashboardBreakdownRow) => row.id ?? row.name ?? '', []);
  const getRowAttributes = useCallback(
    (row: DashboardBreakdownRow) => {
      if (!onRowSelect) {
        return undefined;
      }
      const rowKey = resolveRowKey(row);
      return campaignListRowDataAttributes(selectedRowId != null && selectedRowId === rowKey);
    },
    [onRowSelect, resolveRowKey, selectedRowId]
  );

  return (
    <DirectoryDataTableEngine<DashboardBreakdownRow, DashboardBreakdownColumnId>
      bodyCellClassName={dashboardTableTdClass}
      colgroupRef={colgroupRef}
      columnWidths={columnWidths}
      columns={visibleColumns}
      fillContainer={fillContainer}
      footerCellClassName={cn(dashboardTableTdClass, dashboardTableTfootTdClass)}
      footerRow={totals as DashboardBreakdownRow | undefined}
      getRowAttributes={getRowAttributes}
      headerCellClassName={dashboardTableThClass}
      isResizableColumn={(columnId) =>
        isDashboardBreakdownColumnResizable(columnId, visibleColumns)
      }
      renderBodyCell={(columnId, row) => {
        if (columnId === 'name') {
          return (
            <div className={campaignListNameRowCellClass}>
              <div
                className={campaignListNameRowTextClass}
                onClick={nameLink ? (event) => event.stopPropagation() : undefined}
              >
                {nameLink ? (
                  nameLink(row)
                ) : (
                  <span className={campaignListNameTextClass} title={row.name}>
                    {row.name}
                  </span>
                )}
              </div>
            </div>
          );
        }
        return (
          <div
            className={cn(
              campaignListCellContentNumClass,
              dashboardBreakdownMetricToneClass(columnId, row)
            )}
          >
            {renderBreakdownCell(columnId, { row })}
          </div>
        );
      }}
      renderFooterCell={(columnId, row) => {
        if (columnId === 'name') {
          return <div className={campaignListCellContentClass}>Total</div>;
        }
        return (
          <div className={campaignListCellContentNumClass}>
            {renderBreakdownCell(columnId, { row, isTotal: true })}
          </div>
        );
      }}
      renderHeaderLabel={(columnId) => {
        const label = BREAKDOWN_COLUMN_LABELS[columnId];
        if (sortState && onSortChange && isDashboardBreakdownSortableColumn(columnId)) {
          return (
            <SortableTableHead
              activeOrder={sortState.order}
              activeSort={sortState.column}
              className="h-auto p-0"
              label={label}
              numeric={columnId !== 'name'}
              sortField={columnId}
              onSort={(field) =>
                onSortChange(
                  toggleDashboardBreakdownSort(sortState, field as DashboardBreakdownColumnId)
                )
              }
            />
          );
        }
        return (
          <div className={campaignListHeaderShellClass}>
            <div className={campaignListHeaderLabelClass}>
              <span className="whitespace-nowrap" title={label}>
                {label}
              </span>
            </div>
          </div>
        );
      }}
      renderResizeHandle={(columnId) => {
        const label = BREAKDOWN_COLUMN_LABELS[columnId];
        return (
          <CampaignListColumnResizeHandle
            label={`Resize ${label} column`}
            onPointerDown={(event) => startResize(columnId, event)}
          />
        );
      }}
      onRowClick={onRowSelect}
      resolveColumnMaxWidths={resolveDashboardBreakdownLayoutMaxWidths}
      rowClassName={onRowSelect ? 'cursor-pointer' : undefined}
      rowKey={resolveRowKey}
      rows={rows}
      surfaceClassName={dashboardTableSurfaceClass}
      tableClassName={dashboardTableClass}
      tableRef={tableRef}
    />
  );
}

export {
  campaignReportBreakdownLink,
  campaignBreakdownLink,
} from '@/domains/dashboards/dashboard_breakdown_links';
