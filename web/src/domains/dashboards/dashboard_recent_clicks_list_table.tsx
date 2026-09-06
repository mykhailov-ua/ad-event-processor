import { useRef } from 'react';

import {
  campaignListBodyToolsGutterClass,
  campaignListCellContentClass,
  campaignListCellContentNumClass,
  campaignListCellToolsClass,
  campaignListHeaderCellClass,
  campaignListHeaderLabelClass,
  campaignListHeaderLabelNumClass,
  campaignListHeaderToolsClass,
  campaignListNumClass,
  campaignListTableClass,
  campaignListTableSurfaceClass,
  campaignListTdClass,
  campaignListThClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { CampaignListColumnResizeHandle } from '@/domains/campaigns/list/campaign_list_table_header_cell';
import type { ClickLogEvent } from '@/domains/dashboards/buyer_dashboard_types';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';
import type { DashboardRecentClickColumnId } from '@/domains/dashboards/dashboard_preferences';
import { RECENT_CLICK_COLUMN_LABELS } from '@/domains/dashboards/dashboard_preferences';
import {
  clampUserResizedDashboardRecentClickColumnWidthPx,
  dashboardRecentClickTableWidthPx,
  DASHBOARD_RECENT_CLICK_COLUMN_WIDTH_PX,
  isDashboardRecentClickColumnResizable,
  isDashboardRecentClickCopyColumn,
  isDashboardRecentClickNumericColumn,
} from '@/domains/dashboards/dashboard_table_column_widths';
import { useDashboardRecentClickColumnWidths } from '@/domains/dashboards/use_dashboard_table_column_widths';
import { CopyButton } from '@/shell/copy_button';
import { DirectoryTable, TableBody, TableHeader } from '@/shell/directory_table';
import { useDirectoryColumnResize } from '@/shell/directory_column_resize';
import { displayTimestamp } from '@/lib/display';
import { cn } from '@/lib/utils';

export type DashboardRecentClicksListTableProps = {
  events: ClickLogEvent[];
  columns: DashboardRecentClickColumnId[];
};

function renderRecentClickCell(columnId: DashboardRecentClickColumnId, event: ClickLogEvent) {
  switch (columnId) {
    case 'click_id':
      return event.click_id ?? '-';
    case 'created_at':
      return displayTimestamp(event.created_at);
    case 'campaign_id':
      return event.campaign_id ?? '-';
    case 'country':
      return event.country ?? '-';
    case 'sub1':
      return event.sub1 ?? '-';
    case 'placement_id':
      return event.placement_id ?? '-';
    case 'goal_name':
      return event.goal_name ?? '-';
    case 'cost':
      return formatDashboardUsdFromMicro(event.attributed_cost_micro);
    case 'revenue':
      return formatDashboardUsdFromMicro(event.revenue_micro);
    default:
      return '-';
  }
}

export function DashboardRecentClicksListTable({
  events,
  columns,
}: DashboardRecentClicksListTableProps) {
  const visibleColumns =
    columns.length > 0
      ? columns
      : (['click_id', 'created_at', 'campaign_id'] as DashboardRecentClickColumnId[]);
  const { columnWidths, handleColumnWidthCommit } =
    useDashboardRecentClickColumnWidths(visibleColumns);
  const tableRef = useRef<HTMLTableElement>(null);
  const colgroupRef = useRef<HTMLTableColElement>(null);
  const { startResize } = useDirectoryColumnResize({
    columnWidths,
    columns: visibleColumns,
    minWidthPx: (columnId) => DASHBOARD_RECENT_CLICK_COLUMN_WIDTH_PX[columnId],
    clampUserWidth: clampUserResizedDashboardRecentClickColumnWidthPx,
    onColumnWidthCommit: handleColumnWidthCommit,
    colgroupRef,
    tableRef,
  });
  const tableWidthPx = dashboardRecentClickTableWidthPx(visibleColumns, columnWidths);

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
            const numeric = isDashboardRecentClickNumericColumn(columnId);
            const resizable = isDashboardRecentClickColumnResizable(columnId, visibleColumns);
            const label = RECENT_CLICK_COLUMN_LABELS[columnId];
            return (
              <th
                key={columnId}
                className={cn(
                  campaignListThClass,
                  campaignListCellToolsClass,
                  resizable && 'relative'
                )}
              >
                <div className={campaignListHeaderCellClass}>
                  <div
                    className={
                      numeric ? campaignListHeaderLabelNumClass : campaignListHeaderLabelClass
                    }
                  >
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
        {events.map((event) => (
          <tr key={`${event.click_id}-${event.created_at}`}>
            {visibleColumns.map((columnId) => {
              if (columnId === 'click_id' && event.click_id) {
                return (
                  <td
                    key={columnId}
                    className={cn(
                      campaignListTdClass,
                      campaignListCellToolsClass,
                      campaignListNumClass,
                      'text-muted-foreground'
                    )}
                  >
                    <div className={campaignListHeaderCellClass}>
                      <div className={campaignListCellContentClass}>
                        <span
                          className="select-text whitespace-nowrap font-mono text-xs tabular-nums"
                          title={event.click_id}
                        >
                          {event.click_id}
                        </span>
                      </div>
                      <CopyButton
                        flashOnCopy
                        label="Event id"
                        showToast={false}
                        value={event.click_id}
                      />
                    </div>
                  </td>
                );
              }
              if (columnId === 'campaign_id' && event.campaign_id) {
                return (
                  <td
                    key={columnId}
                    className={cn(
                      campaignListTdClass,
                      campaignListCellToolsClass,
                      campaignListNumClass,
                      'text-muted-foreground'
                    )}
                  >
                    <div className={campaignListHeaderCellClass}>
                      <div className={campaignListCellContentClass}>
                        <span
                          className="select-text whitespace-nowrap font-mono text-xs tabular-nums"
                          title={event.campaign_id}
                        >
                          {event.campaign_id}
                        </span>
                      </div>
                      <CopyButton
                        flashOnCopy
                        label="Campaign ID"
                        showToast={false}
                        value={event.campaign_id}
                      />
                    </div>
                  </td>
                );
              }
              if (isDashboardRecentClickCopyColumn(columnId)) {
                return (
                  <td
                    key={columnId}
                    className={cn(
                      campaignListTdClass,
                      campaignListCellToolsClass,
                      campaignListNumClass,
                      'text-muted-foreground'
                    )}
                  >
                    <div className={campaignListHeaderCellClass}>
                      <div className={campaignListCellContentNumClass}>-</div>
                      <div aria-hidden className={campaignListBodyToolsGutterClass} />
                    </div>
                  </td>
                );
              }
              const numeric = isDashboardRecentClickNumericColumn(columnId);
              return (
                <td
                  key={columnId}
                  className={cn(
                    campaignListTdClass,
                    campaignListCellToolsClass,
                    numeric && campaignListNumClass
                  )}
                >
                  <div className={campaignListHeaderCellClass}>
                    <div
                      className={
                        numeric ? campaignListCellContentNumClass : campaignListCellContentClass
                      }
                    >
                      <span
                        className="block whitespace-nowrap"
                        title={columnId === 'sub1' ? event.sub1 : undefined}
                      >
                        {renderRecentClickCell(columnId, event)}
                      </span>
                    </div>
                    <div aria-hidden className={campaignListBodyToolsGutterClass} />
                  </div>
                </td>
              );
            })}
          </tr>
        ))}
      </TableBody>
    </DirectoryTable>
  );
}
