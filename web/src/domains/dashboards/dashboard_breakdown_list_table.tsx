import { useMemo, useRef, type ReactNode } from 'react';

import {
  campaignListCellContentClass,
  campaignListCellContentNumClass,
  campaignListDashboardHeaderLabelClass,
  campaignListHeaderShellClass,
  campaignListNameRowCellClass,
  campaignListNameRowTextClass,
  campaignListNameTextClass,
  campaignListTableClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import {
  dashboardTableSurfaceClass,
  dashboardTableTdClass,
  dashboardTableTfootTdClass,
  dashboardTableThClass,
} from '@/domains/dashboards/dashboard_classes';
import { formatDashboardBreakdownCellText } from '@/domains/dashboards/dashboard_breakdown_cell_text';
import { CampaignListColumnResizeHandle } from '@/domains/campaigns/list/campaign_list_table_header_cell';
import type { DashboardBreakdownRow, DashboardBreakdownTable } from '@/domains/dashboards/buyer_dashboard_types';
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

  return (
    <DirectoryDataTableEngine<DashboardBreakdownRow, DashboardBreakdownColumnId>
      bodyCellClassName={dashboardTableTdClass}
      colgroupRef={colgroupRef}
      columnWidths={columnWidths}
      columns={visibleColumns}
      fillContainer={fillContainer}
      footerCellClassName={cn(dashboardTableTdClass, dashboardTableTfootTdClass)}
      footerRow={totals as DashboardBreakdownRow | undefined}
      headerCellClassName={dashboardTableThClass}
      isResizableColumn={(columnId) =>
        isDashboardBreakdownColumnResizable(columnId, visibleColumns)
      }
      renderBodyCell={(columnId, row) => {
        if (columnId === 'name') {
          return (
            <div className={campaignListNameRowCellClass}>
              <div className={campaignListNameRowTextClass}>
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
          <div className={campaignListCellContentNumClass}>
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
        return (
          <div className={campaignListHeaderShellClass}>
            <div className={campaignListDashboardHeaderLabelClass}>
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
      resolveColumnMaxWidths={resolveDashboardBreakdownLayoutMaxWidths}
      rowKey={(row) => row.id ?? row.name ?? ''}
      rows={rows}
      surfaceClassName={dashboardTableSurfaceClass}
      tableClassName={campaignListTableClass}
      tableRef={tableRef}
    />
  );
}

export { campaignReportBreakdownLink, campaignBreakdownLink } from '@/domains/dashboards/dashboard_breakdown_links';
