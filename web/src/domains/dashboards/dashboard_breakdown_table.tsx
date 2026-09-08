import type { ReactNode } from 'react';
import { useMemo } from 'react';

import {
  dashboardCardTitleClass,
  dashboardTableSectionHeaderClass,
} from '@/domains/dashboards/dashboard_classes';
import { EmptyState } from '@/shell/empty_state';
import type { DashboardBreakdownTable } from '@/domains/dashboards/buyer_dashboard_types';
import { DashboardCard } from '@/domains/dashboards/dashboard_card';
import { DashboardBreakdownListTable } from '@/domains/dashboards/dashboard_breakdown_list_table';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import type { DashboardBreakdownScope } from '@/domains/dashboards/dashboard_table_column_prefs';
import { DirectoryTablePagination } from '@/shell/directory_data_table/directory_table_pagination';
import {
  sliceDirectoryTableRows,
  useDirectoryTablePagination,
} from '@/shell/directory_data_table/use_directory_table_pagination';
import { cn } from '@/lib/utils';

export type DashboardBreakdownTableProps = {
  title: string;
  scope: DashboardBreakdownScope;
  table: DashboardBreakdownTable | undefined;
  columns: DashboardBreakdownColumnId[];
  nameLink?: (row: { id?: string; name?: string }) => ReactNode;
  emptyDescription?: string;
  embedded?: boolean;
  sectionClassName?: string;
  pageSize?: number;
  meta?: ReactNode;
  fillContainer?: boolean;
};

export function DashboardBreakdownTableSection({
  title,
  scope,
  table,
  columns,
  nameLink,
  emptyDescription,
  embedded = false,
  sectionClassName,
  pageSize = 10,
  meta,
  fillContainer = true,
}: DashboardBreakdownTableProps) {
  const rows = table?.rows ?? [];
  const pagination = useDirectoryTablePagination(rows.length, pageSize);
  const pageRows = useMemo(
    () => sliceDirectoryTableRows(rows, pagination),
    [pagination.end, pagination.start, rows]
  );

  const content =
    rows.length === 0 ? (
      <EmptyState
        className="border-0 bg-transparent py-8 shadow-none"
        description={emptyDescription}
        variant="no-results"
      />
    ) : (
      <>
        <DashboardBreakdownListTable
          columns={columns}
          fillContainer={fillContainer}
          nameLink={nameLink}
          scope={scope}
          table={{ ...table, rows: pageRows }}
          widthProbeTable={table}
        />
        <DirectoryTablePagination
          end={pagination.end}
          page={pagination.page}
          pageCount={pagination.pageCount}
          pageSize={pagination.pageSize}
          start={pagination.start}
          totalRows={rows.length}
          truncatedTotal={table?.truncated ? table.total : undefined}
          onPageChange={pagination.setPage}
        />
      </>
    );

  if (embedded) {
    return (
      <section className={cn('min-w-0', sectionClassName)}>
        <div className={dashboardTableSectionHeaderClass}>
          <h2 className={dashboardCardTitleClass}>{title}</h2>
        </div>
        {content}
      </section>
    );
  }

  return (
    <DashboardCard bodyClassName="p-0" className="min-w-0" meta={meta} title={title}>
      {content}
    </DashboardCard>
  );
}

export { campaignReportBreakdownLink, campaignBreakdownLink } from '@/domains/dashboards/dashboard_breakdown_links';
