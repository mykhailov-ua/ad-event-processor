import type { ReactNode } from 'react';

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
}: DashboardBreakdownTableProps) {
  const rows = table?.rows ?? [];

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
          nameLink={nameLink}
          scope={scope}
          table={table}
        />
        {table?.truncated ? (
          <p className="border-t border-border px-5 py-3 text-xs text-muted-foreground">
            Showing top {rows.length} of {table.total ?? rows.length} rows.
          </p>
        ) : null}
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
    <DashboardCard bodyClassName="p-0" className="min-w-0" title={title}>
      {content}
    </DashboardCard>
  );
}

export { campaignBreakdownLink } from '@/domains/dashboards/dashboard_breakdown_list_table';
