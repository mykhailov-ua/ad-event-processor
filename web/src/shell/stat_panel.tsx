import type { ReactNode } from 'react';

import { shellChrome } from '@/shell/shell_chrome';
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
      <div className={cn(SECTION_TABLE_SURFACE_CLASS, 'grid gap-2 text-[13px] leading-[18px]')}>
        {children}
      </div>
    </div>
  );
}

export function StatRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="grid grid-cols-[1fr_auto] gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-numeric text-foreground">{value}</span>
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
      <div className={cn(shellChrome.sectionHeaderBandClass, 'border-border/40')}>
        <h3 className="text-base font-medium tracking-tight">{title}</h3>
        {meta}
      </div>
      <div className={cn(SECTION_TABLE_SURFACE_CLASS, 'px-5 py-4')}>{children}</div>
    </section>
  );
}
