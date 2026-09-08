import type { ReactNode } from 'react';

import { OpsStatusChip } from '@/domains/ops/ops_status';
import { shellChrome } from '@/shell/shell_chrome';

export function OpsStatGrid({ children }: { children: ReactNode }) {
  return <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">{children}</div>;
}

export function OpsStatPanel({
  title,
  status,
  children,
}: {
  title: string;
  status?: string;
  children: ReactNode;
}) {
  return (
    <section className={shellChrome.sectionPanelClass}>
      <header className="grid grid-cols-[1fr_auto] items-center gap-2">
        <h2 className="text-sm font-semibold">{title}</h2>
        <OpsStatusChip status={status} />
      </header>
      <dl className="grid gap-1">{children}</dl>
    </section>
  );
}

export function OpsKvRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="grid grid-cols-[1fr_auto] items-baseline gap-2 text-sm">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-semibold">{value}</dd>
    </div>
  );
}
