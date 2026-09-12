import type { ReactNode } from 'react';

import { OpsStatusChip } from '@/domains/ops/ops_status';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';
export function OpsStatGrid({ children }: { children: ReactNode }) {
  return (
    <div className={`grid ${adminSpacing.gap.lg} sm:grid-cols-2 lg:grid-cols-3`}>{children}</div>
  );
}

export function OpsStatPanel({
  title,
  status,
  children,
}: {
  title: ReactNode;
  status?: string;
  children: ReactNode;
}) {
  return (
    <section className={shellChrome.sectionPanelClass}>
      <header className={cn('flex flex-wrap items-center justify-between', adminSpacing.gap.md)}>
        <h2 className={adminTypography.sectionTitle}>{title}</h2>
        <OpsStatusChip status={status} />
      </header>
      <div className={`grid ${adminSpacing.gap.md}`}>{children}</div>
    </section>
  );
}

export function OpsStatRow({ label, value }: { label: ReactNode; value: ReactNode }) {
  return (
    <div
      className={cn(
        'grid grid-cols-[1fr_auto] items-baseline',
        adminSpacing.gap.md,
        adminTypography.body
      )}
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="font-semibold tabular-nums num">{value}</span>
    </div>
  );
}

/** @deprecated Use OpsStatRow */
export const OpsKvRow = OpsStatRow;
