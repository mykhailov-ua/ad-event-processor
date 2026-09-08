import { Link } from 'react-router-dom';
import { useMemo } from 'react';

import { EmptyState } from '@/shell/empty_state';
import type { ClickLogEvent } from '@/domains/dashboards/buyer_dashboard_types';
import { DashboardCard } from '@/domains/dashboards/dashboard_card';
import {
  dashboardTableScrollHostBoundedClass,
  dashboardTableScrollHostClass,
} from '@/domains/dashboards/dashboard_classes';
import { DashboardRecentClicksListTable } from '@/domains/dashboards/dashboard_recent_clicks_list_table';
import type { DashboardRecentClickColumnId } from '@/domains/dashboards/dashboard_preferences';
import { DirectoryTablePagination } from '@/shell/directory_data_table/directory_table_pagination';
import {
  sliceDirectoryTableRows,
  useDirectoryTablePagination,
} from '@/shell/directory_data_table/use_directory_table_pagination';
import { cn } from '@/lib/utils';

export type DashboardRecentClicksProps = {
  events: ClickLogEvent[];
  columns: DashboardRecentClickColumnId[];
  viewAllHref?: string;
  pageSize?: number;
  fillContainer?: boolean;
  fillHeight?: boolean;
};

export function DashboardRecentClicks({
  events,
  columns,
  viewAllHref,
  pageSize = 10,
  fillContainer = true,
  fillHeight = false,
}: DashboardRecentClicksProps) {
  const pagination = useDirectoryTablePagination(events.length, pageSize);
  const pageEvents = useMemo(
    () => sliceDirectoryTableRows(events, pagination),
    [events, pagination.end, pagination.start]
  );

  const tableBody =
    events.length === 0 ? (
      <EmptyState
        className={cn('border-0 bg-transparent py-8 shadow-none', fillHeight && 'min-h-0 flex-1')}
        description="Clicks will appear here when traffic is recorded for the selected period."
        variant="no-results"
      />
    ) : (
      <>
        <div
          className={
            fillHeight ? dashboardTableScrollHostClass : dashboardTableScrollHostBoundedClass
          }
        >
          <DashboardRecentClicksListTable
            columns={columns}
            events={pageEvents}
            fillContainer={fillContainer}
          />
        </div>
        <DirectoryTablePagination
          end={pagination.end}
          page={pagination.page}
          pageCount={pagination.pageCount}
          pageSize={pagination.pageSize}
          start={pagination.start}
          totalRows={events.length}
          onPageChange={pagination.setPage}
        />
      </>
    );

  const content = fillHeight ? (
    <div className="flex min-h-0 flex-1 flex-col">{tableBody}</div>
  ) : (
    tableBody
  );

  return (
    <DashboardCard
      bodyClassName="p-0"
      className="min-w-0"
      fillHeight={fillHeight}
      meta={
        viewAllHref ? (
          <Link className="text-sm text-primary hover:underline" to={viewAllHref}>
            View all
          </Link>
        ) : undefined
      }
      title="Recent clicks"
    >
      {content}
    </DashboardCard>
  );
}
