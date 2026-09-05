import { useMemo, useRef, type ReactNode } from 'react';
import { Link } from 'react-router-dom';

import {
  campaignListBodyToolsGutterClass,
  campaignListCellContentClass,
  campaignListCellContentNumClass,
  campaignListCellToolsClass,
  campaignListHeaderCellClass,
  campaignListHeaderLabelClass,
  campaignListHeaderLabelNumClass,
  campaignListHeaderToolsClass,
  campaignListNameRowCellClass,
  campaignListNameRowMenuSlotClass,
  campaignListNameRowTextClass,
  campaignListNameTextClass,
  campaignListNumClass,
  campaignListTableClass,
  campaignListTableSurfaceClass,
  campaignListTdClass,
  campaignListTfootTdClass,
  campaignListThClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { CampaignListColumnResizeHandle } from '@/domains/campaigns/list/campaign_list_table_header_cell';
import type { DashboardBreakdownRow, DashboardBreakdownTable } from '@/domains/dashboards/buyer_dashboard_types';
import {
  formatDashboardCrPct,
  formatDashboardUsdFromMicro,
} from '@/domains/dashboards/dashboard_format';
import { formatRoi } from '@/domains/dashboards/dashboard_metrics';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import { BREAKDOWN_COLUMN_LABELS } from '@/domains/dashboards/dashboard_preferences';
import type { DashboardBreakdownScope } from '@/domains/dashboards/dashboard_table_column_prefs';
import {
  clampUserResizedDashboardBreakdownColumnWidthPx,
  dashboardBreakdownTableWidthPx,
  DASHBOARD_BREAKDOWN_COLUMN_WIDTH_PX,
  isDashboardBreakdownColumnResizable,
  isDashboardBreakdownNumericColumn,
} from '@/domains/dashboards/dashboard_table_column_widths';
import { useDashboardBreakdownColumnWidths } from '@/domains/dashboards/use_dashboard_table_column_widths';
import { DirectoryTable, TableBody, TableFooter, TableHeader } from '@/shell/directory_table';
import { useDirectoryColumnResize } from '@/shell/directory_column_resize';
import { displayCount } from '@/lib/display';
import { cn } from '@/lib/utils';

export type DashboardBreakdownListTableProps = {
  scope: DashboardBreakdownScope;
  table: DashboardBreakdownTable | undefined;
  columns: DashboardBreakdownColumnId[];
  nameLink?: (row: { id?: string; name?: string }) => ReactNode;
};

type BreakdownCellContext = {
  row: DashboardBreakdownRow;
  isTotal?: boolean;
};

function renderBreakdownCell(columnId: DashboardBreakdownColumnId, ctx: BreakdownCellContext): ReactNode {
  const { row } = ctx;
  switch (columnId) {
    case 'name':
      return ctx.isTotal ? 'Total' : row.name;
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
      return formatDashboardUsdFromMicro(row.profit_micro);
    case 'cpc':
      return formatDashboardUsdFromMicro(row.cpc_micro);
    case 'cpa':
      return formatDashboardUsdFromMicro(row.cpa_micro);
    case 'cr':
      return formatDashboardCrPct(row.cr_pct);
    case 'epc':
      return formatDashboardUsdFromMicro(row.epc_micro);
    case 'roi':
      return formatRoi(row.roi_pct);
    default:
      return '';
  }
}

export function DashboardBreakdownListTable({
  scope,
  table,
  columns,
  nameLink,
}: DashboardBreakdownListTableProps) {
  const rows = table?.rows ?? [];
  const totals = table?.totals;
  const visibleColumns = columns.length > 0 ? columns : (['name', 'clicks', 'conversions'] as DashboardBreakdownColumnId[]);
  const nameLabels = useMemo(
    () => ['Total', ...rows.map((row) => row.name ?? ''), ...(totals?.name ? [totals.name] : [])],
    [rows, totals?.name],
  );
  const { columnWidths, handleColumnWidthCommit } = useDashboardBreakdownColumnWidths(
    scope,
    visibleColumns,
    nameLabels,
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
  const tableWidthPx = dashboardBreakdownTableWidthPx(visibleColumns, columnWidths);

  return (
    <DirectoryTable
      className={cn(campaignListTableSurfaceClass, 'rounded-none border-0 shadow-none')}
      fixedLayout
      horizontalScroll
      tableClassName={campaignListTableClass}
      tableRef={tableRef}
      tableStyle={{
        width: `${tableWidthPx}px`,
        minWidth: `${tableWidthPx}px`,
        tableLayout: 'fixed',
      }}
    >
      <colgroup ref={colgroupRef}>
        {visibleColumns.map((columnId) => (
          <col key={columnId} style={{ width: `${columnWidths[columnId]}px` }} />
        ))}
      </colgroup>
      <TableHeader>
        <tr>
          {visibleColumns.map((columnId) => {
            const numeric = isDashboardBreakdownNumericColumn(columnId);
            const resizable = isDashboardBreakdownColumnResizable(columnId, visibleColumns);
            const label = BREAKDOWN_COLUMN_LABELS[columnId];
            return (
              <th
                key={columnId}
                className={cn(
                  campaignListThClass,
                  campaignListCellToolsClass,
                  resizable && 'relative',
                )}
              >
                <div className={campaignListHeaderCellClass}>
                  <div className={numeric ? campaignListHeaderLabelNumClass : campaignListHeaderLabelClass}>
                    <span className="whitespace-nowrap" title={label}>
                      {label}
                    </span>
                  </div>
                  <div className={campaignListHeaderToolsClass} />
                </div>
                {resizable ? (
                  <CampaignListColumnResizeHandle
                    label={`Resize ${label} column`}
                    onPointerDown={(event) => startResize(columnId, event)}
                  />
                ) : null}
              </th>
            );
          })}
        </tr>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <tr key={row.id ?? row.name}>
            {visibleColumns.map((columnId) => {
              const numeric = isDashboardBreakdownNumericColumn(columnId);
              if (columnId === 'name') {
                return (
                  <td key={columnId} className={cn(campaignListTdClass, campaignListCellToolsClass)}>
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
                      <div aria-hidden className={campaignListNameRowMenuSlotClass} />
                    </div>
                  </td>
                );
              }
              return (
                <td
                  key={columnId}
                  className={cn(campaignListTdClass, campaignListCellToolsClass, campaignListNumClass)}
                >
                  <div className={campaignListHeaderCellClass}>
                    <div className={campaignListCellContentNumClass}>
                      {renderBreakdownCell(columnId, { row })}
                    </div>
                    <div aria-hidden className={campaignListBodyToolsGutterClass} />
                  </div>
                </td>
              );
            })}
          </tr>
        ))}
      </TableBody>
      {totals ? (
        <TableFooter>
          <tr>
            {visibleColumns.map((columnId) => {
              const numeric = isDashboardBreakdownNumericColumn(columnId);
              if (columnId === 'name') {
                return (
                  <td
                    key={columnId}
                    className={cn(campaignListTdClass, campaignListTfootTdClass, campaignListCellToolsClass)}
                  >
                    <div className={campaignListHeaderCellClass}>
                      <div className={campaignListCellContentClass}>Total</div>
                    </div>
                  </td>
                );
              }
              return (
                <td
                  key={columnId}
                  className={cn(
                    campaignListTdClass,
                    campaignListTfootTdClass,
                    campaignListCellToolsClass,
                    campaignListNumClass,
                  )}
                >
                  <div className={campaignListHeaderCellClass}>
                    <div className={campaignListCellContentNumClass}>
                      {renderBreakdownCell(columnId, { row: totals, isTotal: true })}
                    </div>
                    <div aria-hidden className={campaignListBodyToolsGutterClass} />
                  </div>
                </td>
              );
            })}
          </tr>
        </TableFooter>
      ) : null}
    </DirectoryTable>
  );
}

export function campaignBreakdownLink(row: { id?: string; name?: string }) {
  if (!row.id) {
    return (
      <span className={campaignListNameTextClass} title={row.name}>
        {row.name}
      </span>
    );
  }
  return (
    <Link
      className={cn(campaignListNameTextClass, 'text-primary hover:underline')}
      title={row.name}
      to={`/campaigns/${row.id}/edit`}
    >
      {row.name}
    </Link>
  );
}
