import { Link } from 'react-router-dom';
import { useMemo } from 'react';

import { EmptyState } from '@/shell/empty_state';
import type { ClickLogEvent } from '@/domains/dashboards/buyer_dashboard_types';
import { DashboardCard } from '@/domains/dashboards/dashboard_card';
import { DashboardRecentClicksListTable } from '@/domains/dashboards/dashboard_recent_clicks_list_table';
import type { DashboardRecentClickColumnId } from '@/domains/dashboards/dashboard_preferences';
import { DirectoryTablePagination } from '@/shell/directory_data_table/directory_table_pagination';
import {
  sliceDirectoryTableRows,
  useDirectoryTablePagination,
} from '@/shell/directory_data_table/use_directory_table_pagination';

export type DashboardRecentClicksProps = {
  events: ClickLogEvent[];
  columns: DashboardRecentClickColumnId[];
  viewAllHref?: string;
  pageSize?: number;
  fillContainer?: boolean;
};

export function DashboardRecentClicks({
  events,
  columns,
  viewAllHref,
  pageSize = 10,
  fillContainer = false,
}: DashboardRecentClicksProps) {
  const pagination = useDirectoryTablePagination(events.length, pageSize);
  const pageEvents = useMemo(
    () => sliceDirectoryTableRows(events, pagination),
    [events, pagination.end, pagination.start]
  );

  const content =
    events.length === 0 ? (
      <EmptyState
        className="border-0 bg-transparent py-8 shadow-none"
        description="Clicks will appear here when traffic is recorded for the selected period."
        variant="no-results"
      />
    ) : (
      <>
        <DashboardRecentClicksListTable
          columns={columns}
          events={pageEvents}
          fillContainer={fillContainer}
        />
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

  return (
    <DashboardCard
      bodyClassName="p-0"
      className="min-w-0"
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
