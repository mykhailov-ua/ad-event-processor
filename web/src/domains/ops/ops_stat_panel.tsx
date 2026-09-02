import type { ReactNode } from 'react';

import { OpsStatusChip } from '@/domains/ops/ops_status';

export function OpsStatGrid({ children }: { children: ReactNode }) {
  return <div className="admin-ops-stat-grid">{children}</div>;
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
    <section className="admin-ops-stat-card">
      <header className="admin-ops-block__head">
        <h2 className="admin-ops-block__title">{title}</h2>
        <OpsStatusChip status={status} />
      </header>
      <dl className="admin-ops-kv">{children}</dl>
    </section>
  );
}

export function OpsKvRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="admin-ops-kv__row">
      <dt className="admin-ops-kv__label">{label}</dt>
      <dd className="admin-ops-kv__value num">{value}</dd>
    </div>
  );
}
