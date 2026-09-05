import { Link } from 'react-router-dom';

import {
  dashboardCardTitleClass,
  dashboardTableSectionHeaderClass,
} from '@/domains/dashboards/dashboard_classes';
import { EmptyState } from '@/shell/empty_state';
import type { ClickLogEvent } from '@/domains/dashboards/buyer_dashboard_types';
import { DashboardCard } from '@/domains/dashboards/dashboard_card';
import { DashboardRecentClicksListTable } from '@/domains/dashboards/dashboard_recent_clicks_list_table';
import type { DashboardRecentClickColumnId } from '@/domains/dashboards/dashboard_preferences';
import { cn } from '@/lib/utils';

export type DashboardRecentClicksProps = {
  events: ClickLogEvent[];
  columns: DashboardRecentClickColumnId[];
  viewAllHref?: string;
  embedded?: boolean;
  sectionClassName?: string;
};

export function DashboardRecentClicks({
  events,
  columns,
  viewAllHref,
  embedded = false,
  sectionClassName,
}: DashboardRecentClicksProps) {
  const content =
    events.length === 0 ? (
      <EmptyState
        className="border-0 bg-transparent py-8 shadow-none"
        description="Clicks will appear here when traffic is recorded for the selected period."
        variant="no-results"
      />
    ) : (
      <DashboardRecentClicksListTable columns={columns} events={events} />
    );

  if (embedded) {
    return (
      <section className={cn('min-w-0', sectionClassName)}>
        <div className={dashboardTableSectionHeaderClass}>
          <h2 className={dashboardCardTitleClass}>Recent clicks</h2>
          {viewAllHref ? (
            <Link className="text-sm text-primary hover:underline" to={viewAllHref}>
              View all
            </Link>
          ) : null}
        </div>
        {content}
      </section>
    );
  }

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
