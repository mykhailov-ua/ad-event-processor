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
  fillHeight = false,
  headerClassName,
  meta,
  title,
}: {
  bodyClassName?: string;
  children: ReactNode;
  className?: string;
  fillHeight?: boolean;
  headerClassName?: string;
  meta?: ReactNode;
  title: ReactNode;
}) {
  return (
    <section
      className={cn(dashboardCardClass, fillHeight && 'flex h-full min-h-0 flex-col', className)}
    >
      <div className={cn(dashboardCardHeaderClass, headerClassName)}>
        <h2 className={dashboardCardTitleClass}>{title}</h2>
        {meta}
      </div>
      <div
        className={cn(
          dashboardCardBodyClass,
          bodyClassName,
          bodyClassName === 'p-0' && 'min-w-0',
          fillHeight && 'flex min-h-0 flex-1 flex-col'
        )}
      >
        {children}
      </div>
    </section>
  );
}
