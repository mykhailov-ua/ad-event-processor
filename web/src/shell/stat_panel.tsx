import type { ReactNode } from 'react';

import { cn } from '@/lib/utils';

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
    <div className={cn('ui-surface-raised grid gap-3 rounded-2xl border border-border/40 p-5', className)}>
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-base font-medium tracking-tight">{title}</h3>
        {meta}
      </div>
      <div className="grid gap-2 text-sm">{children}</div>
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
    <section className={cn('ui-surface-raised overflow-hidden rounded-2xl border border-border/40', className)}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border/40 px-5 py-4">
        <h3 className="text-base font-medium tracking-tight">{title}</h3>
        {meta}
      </div>
      {children}
    </section>
  );
}
