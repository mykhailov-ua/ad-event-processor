import type { ReactNode } from 'react';

import {
  dashboardCardBodyClass,
  dashboardCardClass,
  dashboardCardHeaderClass,
  dashboardCardTitleClass,
} from '@/domains/dashboards/dashboard_classes';
import { cn } from '@/lib/utils';

export function DashboardCard({
  bodyClassName,
  children,
  className,
  meta,
  title,
}: {
  bodyClassName?: string;
  children: ReactNode;
  className?: string;
  meta?: ReactNode;
  title: ReactNode;
}) {
  return (
    <section className={cn(dashboardCardClass, className)}>
      <div className={dashboardCardHeaderClass}>
        <h2 className={dashboardCardTitleClass}>{title}</h2>
        {meta}
      </div>
      <div className={cn(dashboardCardBodyClass, bodyClassName)}>{children}</div>
    </section>
  );
}
