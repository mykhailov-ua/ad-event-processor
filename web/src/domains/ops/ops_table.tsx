import type { ReactNode } from 'react';

import { Table, TableBody, TableFooter, TableHeader } from '@/components/ui/table';
import { cn } from '@/lib/utils';

/** Campaigns-standard table shell (admin-table-wrap + admin-table--campaigns). */
export function OpsTable({
  head,
  children,
  foot,
  className,
}: {
  head: ReactNode;
  children: ReactNode;
  foot?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('admin-table-wrap admin-table-wrap--static', className)}>
      <Table bare className="admin-table admin-table--campaigns">
        <TableHeader>{head}</TableHeader>
        <TableBody>{children}</TableBody>
        {foot ? <TableFooter>{foot}</TableFooter> : null}
      </Table>
    </div>
  );
}

/** Section title above a table or block; no extra panel border. */
export function OpsBlock({
  title,
  meta,
  children,
  className,
}: {
  title?: string;
  meta?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  if (!title && !meta) {
    return <>{children}</>;
  }

  return (
    <section className={cn('admin-ops-block', className)}>
      <header className="admin-ops-block__head">
        {title ? <h2 className="admin-ops-block__title">{title}</h2> : null}
        {meta}
      </header>
      {children}
    </section>
  );
}

/** @deprecated Use OpsBlock */
export const OpsTableSection = OpsBlock;
