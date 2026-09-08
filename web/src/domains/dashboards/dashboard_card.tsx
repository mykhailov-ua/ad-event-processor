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
  headerClassName,
  meta,
  title,
}: {
  bodyClassName?: string;
  children: ReactNode;
  className?: string;
  headerClassName?: string;
  meta?: ReactNode;
  title: ReactNode;
}) {
  return (
    <section className={cn(dashboardCardClass, className)}>
      <div className={cn(dashboardCardHeaderClass, headerClassName)}>
        <h2 className={dashboardCardTitleClass}>{title}</h2>
        {meta}
      </div>
      <div className={cn(dashboardCardBodyClass, bodyClassName, bodyClassName === 'p-0' && 'min-w-0')}>
        {children}
      </div>
    </section>
  );
}
