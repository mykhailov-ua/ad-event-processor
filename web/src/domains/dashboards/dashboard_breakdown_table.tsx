import type { ReactNode } from 'react';
import { useMemo, useState } from 'react';

import {
  dashboardCardTitleClass,
  dashboardTableScrollHostBoundedClass,
  dashboardTableScrollHostClass,
  dashboardTableSectionHeaderClass,
} from '@/domains/dashboards/dashboard_classes';
import { EmptyState } from '@/shell/empty_state';
import type { DashboardBreakdownTable } from '@/domains/dashboards/buyer_dashboard_types';
import { DashboardCard } from '@/domains/dashboards/dashboard_card';
import { DashboardBreakdownListTable } from '@/domains/dashboards/dashboard_breakdown_list_table';
import {
  DEFAULT_CAMPAIGNS_BREAKDOWN_SORT,
  sortDashboardBreakdownRows,
  type DashboardBreakdownSortState,
} from '@/domains/dashboards/dashboard_breakdown_sort';
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
  fillHeight?: boolean;
  selectedRowId?: string | null;
  onRowSelect?: (rowId: string) => void;
  titleSuffix?: ReactNode;
  enableCampaignSort?: boolean;
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
  fillHeight = false,
  selectedRowId,
  onRowSelect,
  titleSuffix,
  enableCampaignSort = false,
}: DashboardBreakdownTableProps) {
  const [sortState, setSortState] = useState<DashboardBreakdownSortState>(
    DEFAULT_CAMPAIGNS_BREAKDOWN_SORT
  );
  const sortedRows = useMemo(() => {
    if (!enableCampaignSort) {
      return table?.rows ?? [];
    }
    return sortDashboardBreakdownRows(table?.rows ?? [], sortState);
  }, [enableCampaignSort, sortState, table?.rows]);
  const pagination = useDirectoryTablePagination(sortedRows.length, pageSize);
  const pageRows = useMemo(
    () => sliceDirectoryTableRows(sortedRows, pagination),
    [pagination.end, pagination.start, sortedRows]
  );
  const displayTable = useMemo(
    () => (table ? { ...table, rows: pageRows } : table),
    [pageRows, table]
  );
  const titleContent = titleSuffix ? (
    <span className="inline-flex min-w-0 items-center gap-2">
      <span>{title}</span>
      {titleSuffix}
    </span>
  ) : (
    title
  );

  const tableBody =
    sortedRows.length === 0 ? (
      <EmptyState
        className={cn('border-0 bg-transparent py-8 shadow-none', fillHeight && 'min-h-0 flex-1')}
        description={emptyDescription}
        variant="no-results"
      />
    ) : (
      <>
        <div
          className={
            fillHeight ? dashboardTableScrollHostClass : dashboardTableScrollHostBoundedClass
          }
        >
          <DashboardBreakdownListTable
            columns={columns}
            fillContainer={fillContainer}
            nameLink={nameLink}
            scope={scope}
            selectedRowId={selectedRowId}
            sortState={enableCampaignSort ? sortState : undefined}
            table={displayTable}
            widthProbeTable={table}
            onRowSelect={
              onRowSelect
                ? (row) => {
                    const rowId = row.id ?? row.name;
                    if (rowId) {
                      onRowSelect(rowId);
                    }
                  }
                : undefined
            }
            onSortChange={enableCampaignSort ? setSortState : undefined}
          />
        </div>
        <DirectoryTablePagination
          end={pagination.end}
          page={pagination.page}
          pageCount={pagination.pageCount}
          pageSize={pagination.pageSize}
          start={pagination.start}
          totalRows={sortedRows.length}
          truncatedTotal={table?.truncated ? table.total : undefined}
          onPageChange={pagination.setPage}
        />
      </>
    );

  const content = fillHeight ? (
    <div className="flex min-h-0 flex-1 flex-col">{tableBody}</div>
  ) : (
    tableBody
  );

  if (embedded) {
    return (
      <section className={cn('min-w-0', sectionClassName)}>
        <div className={dashboardTableSectionHeaderClass}>
          <h2 className={dashboardCardTitleClass}>{titleContent}</h2>
        </div>
        {content}
      </section>
    );
  }

  return (
    <DashboardCard
      bodyClassName="p-0"
      className="min-w-0"
      fillHeight={fillHeight}
      meta={meta}
      title={titleContent}
    >
      {content}
    </DashboardCard>
  );
}

export {
  campaignReportBreakdownLink,
  campaignBreakdownLink,
} from '@/domains/dashboards/dashboard_breakdown_links';
