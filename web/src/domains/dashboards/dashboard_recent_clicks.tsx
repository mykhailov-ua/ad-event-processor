import { Link } from 'react-router-dom';

import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { CopyableText } from '@/shell/copyable_text';
import { EmptyState } from '@/shell/empty_state';
import { PanelSection } from '@/shell/stat_panel';
import type { ClickLogEvent } from '@/domains/dashboards/buyer_dashboard_types';
import { formatDashboardUsdFromMicro } from '@/domains/dashboards/dashboard_format';
import type { DashboardRecentClickColumnId } from '@/domains/dashboards/dashboard_preferences';
import { RECENT_CLICK_COLUMN_LABELS } from '@/domains/dashboards/dashboard_preferences';
import { displayTimestamp } from '@/lib/display';

export type DashboardRecentClicksProps = {
  events: ClickLogEvent[];
  columns: DashboardRecentClickColumnId[];
  viewAllHref?: string;
};

function renderRecentClickCell(columnId: DashboardRecentClickColumnId, event: ClickLogEvent) {
  switch (columnId) {
    case 'click_id':
      return event.click_id ? (
        <CopyableText label="Click ID" mono value={event.click_id} />
      ) : (
        '-'
      );
    case 'created_at':
      return displayTimestamp(event.created_at);
    case 'campaign_id':
      return event.campaign_id ? (
        <CopyableText label="Campaign ID" mono value={event.campaign_id} />
      ) : (
        '-'
      );
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

export function DashboardRecentClicks({ events, columns, viewAllHref }: DashboardRecentClicksProps) {
  const visibleColumns =
    columns.length > 0 ? columns : (['click_id', 'created_at', 'campaign_id'] as DashboardRecentClickColumnId[]);

  return (
    <PanelSection
      className="min-w-0"
      title="Recent clicks"
      meta={
        viewAllHref ? (
          <Link className="text-sm text-primary hover:underline" to={viewAllHref}>
            View all
          </Link>
        ) : undefined
      }
    >
      {events.length === 0 ? (
        <EmptyState
          className="border-0 bg-transparent py-8 shadow-none"
          description="Clicks will appear here when traffic is recorded for the selected period."
          variant="no-results"
        />
      ) : (
        <DirectoryTable className="border-0 bg-transparent shadow-none" scrollable>
          <TableHeader>
            <TableRow>
              {visibleColumns.map((columnId) => (
                <DirectoryTableHead key={columnId} className="min-w-[8rem]">
                  {RECENT_CLICK_COLUMN_LABELS[columnId]}
                </DirectoryTableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.map((event) => (
              <TableRow key={`${event.click_id}-${event.created_at}`}>
                {visibleColumns.map((columnId) => (
                  <TableCell
                    key={columnId}
                    className={
                      columnId === 'click_id'
                        ? 'max-w-0 truncate font-mono text-xs'
                        : 'max-w-0 truncate text-sm'
                    }
                    title={
                      columnId === 'sub1' ? event.sub1 : columnId === 'click_id' ? event.click_id : undefined
                    }
                  >
                    {renderRecentClickCell(columnId, event)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      )}
    </PanelSection>
  );
}
