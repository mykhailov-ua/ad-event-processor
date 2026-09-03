import type { ReactNode } from 'react';

import { OpsStatusChip } from '@/domains/ops/ops_status';

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
    <section className="rounded-md border border-zinc-200 p-3 dark:border-zinc-800">
      <header className="flex items-center justify-between gap-2">
        <h2 className="text-sm font-semibold">{title}</h2>
        <OpsStatusChip status={status} />
      </header>
      <dl className="grid gap-1">{children}</dl>
    </section>
  );
}

export function OpsKvRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-2 text-sm">
      <dt className="text-zinc-500 dark:text-zinc-400">{label}</dt>
      <dd className="font-semibold tabular-nums num">{value}</dd>
    </div>
  );
}
