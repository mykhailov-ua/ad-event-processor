import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

const SECTION_TABLE_SURFACE_CLASS = 'ui-section-table-host min-w-0';

export function StatPanel({
  children,
  className,
  title,
  meta,
}: {
  children: ReactNode;
  className?: string;
  title: ReactNode;
  meta?: ReactNode;
}) {
  return (
    <div className={cn('ui-surface-raised grid gap-3 p-5', className)}>
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-base font-medium tracking-tight">{title}</h3>
        {meta}
      </div>
      <div className={cn(SECTION_TABLE_SURFACE_CLASS, 'grid gap-2 text-sm')}>{children}</div>
    </div>
  );
}

export function StatRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="tabular-nums text-foreground">{value}</span>
    </div>
  );
}

export function PanelSection({
  children,
  className,
  title,
  meta,
}: {
  children: ReactNode;
  className?: string;
  title: ReactNode;
  meta?: ReactNode;
}) {
  return (
    <section className={cn('ui-surface-raised min-w-0', className)}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border/40 px-5 py-4">
        <h3 className="text-base font-medium tracking-tight">{title}</h3>
        {meta}
      </div>
      <div className={cn(SECTION_TABLE_SURFACE_CLASS, 'px-5 py-4')}>{children}</div>
    </section>
  );
}
