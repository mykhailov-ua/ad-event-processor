import type { ReactNode } from 'react';

import { OpsStatusChip } from '@/domains/ops/ops_status';
import { shellChrome } from '@/shell/shell_chrome';

export function OpsStatGrid({ children }: { children: ReactNode }) {
  return <div >{children}</div>;
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
    <section >
      <header >
        <h2 >{title}</h2>
        <OpsStatusChip status={status} />
      </header>
      <dl >{children}</dl>
    </section>
  );
}

export function OpsKvRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div >
      <dt >{label}</dt>
      <dd >{value}</dd>
    </div>
  );
}
