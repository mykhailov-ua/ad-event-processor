import { useRef } from 'react';

import {
  campaignListCellContentClass,
  campaignListCellContentNumClass,
  campaignListCopyRowClass,
  campaignListCopyToolsSlotClass,
  campaignListDashboardHeaderLabelClass,
  campaignListEllipsisTextClass,
  campaignListHeaderShellClass,
  campaignListTableClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import {
  dashboardTableSurfaceClass,
  dashboardTableTdClass,
  dashboardTableThClass,
} from '@/domains/dashboards/dashboard_classes';
import { CampaignListColumnResizeHandle } from '@/domains/campaigns/list/campaign_list_table_header_cell';
import type { ClickLogEvent } from '@/domains/dashboards/buyer_dashboard_types';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';
import type { DashboardRecentClickColumnId } from '@/domains/dashboards/dashboard_preferences';
import { RECENT_CLICK_COLUMN_LABELS } from '@/domains/dashboards/dashboard_preferences';
import {
  clampUserResizedDashboardRecentClickColumnWidthPx,
  DASHBOARD_RECENT_CLICK_COLUMN_WIDTH_PX,
  isDashboardRecentClickColumnResizable,
  isDashboardRecentClickCopyColumn,
  isDashboardRecentClickNumericColumn,
} from '@/domains/dashboards/dashboard_table_column_widths';
import { useDashboardRecentClickColumnWidths } from '@/domains/dashboards/use_dashboard_table_column_widths';
import { CopyButton } from '@/shell/copy_button';
import { DirectoryDataTableEngine } from '@/shell/directory_data_table/directory_data_table_engine';
import { resolveDashboardRecentClickLayoutMaxWidths } from '@/shell/directory_data_table/layout';
import { useDirectoryColumnResize } from '@/shell/directory_column_resize';
import { displayTimestamp } from '@/lib/display';
import { cn } from '@/lib/utils';

export type DashboardRecentClicksListTableProps = {
  events: ClickLogEvent[];
  columns: DashboardRecentClickColumnId[];
  fillContainer?: boolean;
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
  fillContainer = false,
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

  return (
    <DirectoryDataTableEngine
      bodyCellClassName={dashboardTableTdClass}
      colgroupRef={colgroupRef}
      columnWidths={columnWidths}
      columns={visibleColumns}
      fillContainer={fillContainer}
      headerCellClassName={dashboardTableThClass}
      isResizableColumn={(columnId) =>
        isDashboardRecentClickColumnResizable(columnId, visibleColumns)
      }
      renderBodyCell={(columnId, event) => {
        if (columnId === 'click_id' && event.click_id) {
          return (
            <div className={campaignListCopyRowClass}>
              <span
                className={cn(
                  campaignListEllipsisTextClass,
                  'min-w-0 flex-1 font-numeric text-xs text-muted-foreground'
                )}
                title={event.click_id}
              >
                {event.click_id}
              </span>
              <div className={campaignListCopyToolsSlotClass}>
                <CopyButton
                  flashOnCopy
                  label="Event id"
                  showToast={false}
                  value={event.click_id}
                />
              </div>
            </div>
          );
        }
        if (columnId === 'campaign_id' && event.campaign_id) {
          return (
            <div className={campaignListCopyRowClass}>
              <span
                className={cn(
                  campaignListEllipsisTextClass,
                  'min-w-0 flex-1 font-numeric text-xs text-muted-foreground'
                )}
                title={event.campaign_id}
              >
                {event.campaign_id}
              </span>
              <div className={campaignListCopyToolsSlotClass}>
                <CopyButton
                  flashOnCopy
                  label="Campaign ID"
                  showToast={false}
                  value={event.campaign_id}
                />
              </div>
            </div>
          );
        }
        if (isDashboardRecentClickCopyColumn(columnId)) {
          return <div className={campaignListCellContentNumClass}>-</div>;
        }
        const numeric = isDashboardRecentClickNumericColumn(columnId);
        return (
          <div className={numeric ? campaignListCellContentNumClass : campaignListCellContentClass}>
            <span
              className="block whitespace-nowrap"
              title={columnId === 'sub1' ? event.sub1 : undefined}
            >
              {renderRecentClickCell(columnId, event)}
            </span>
          </div>
        );
      }}
      renderHeaderLabel={(columnId) => {
        const label = RECENT_CLICK_COLUMN_LABELS[columnId];
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
        const label = RECENT_CLICK_COLUMN_LABELS[columnId];
        return (
          <CampaignListColumnResizeHandle
            label={`Resize ${label} column`}
            onPointerDown={(event) => startResize(columnId, event)}
          />
        );
      }}
      resolveColumnMaxWidths={resolveDashboardRecentClickLayoutMaxWidths}
      rowKey={(event) => `${event.click_id}-${event.created_at}`}
      rows={events}
      surfaceClassName={dashboardTableSurfaceClass}
      tableClassName={campaignListTableClass}
      tableRef={tableRef}
    />
  );
}
