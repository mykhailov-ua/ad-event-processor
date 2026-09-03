import type { ComponentProps, ReactNode } from 'react';

import { TableCell, TableHead, TableRow } from '@/components/ui/table';
import { cn } from '@/lib/utils';
import { DirectoryTable, TableBody, TableFooter, TableHeader } from '@/shell/directory_table';

/** Ops matrix chrome: sticky headers, zebra rows, numeric column alignment. */
export const OPS_DIRECTORY_TABLE_CLASS = 'admin-table--ops';

/** Ops directory table shell; delegates border/scroll to DirectoryTable. */
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
    <DirectoryTable
      className={className}
      fixedLayout
      tableClassName={OPS_DIRECTORY_TABLE_CLASS}
    >
      <TableHeader>{head}</TableHeader>
      <TableBody>{children}</TableBody>
      {foot ? <TableFooter>{foot}</TableFooter> : null}
    </DirectoryTable>
  );
}

export function OpsTableHeaderRow(props: ComponentProps<typeof TableRow>) {
  return <TableRow {...props} />;
}

export function OpsTableRow(props: ComponentProps<typeof TableRow>) {
  return <TableRow {...props} />;
}

export function OpsTableHead({
  numeric,
  className,
  ...props
}: ComponentProps<typeof TableHead> & { numeric?: boolean }) {
  return <TableHead className={cn(numeric && 'num', className)} {...props} />;
}

export function OpsTableCell({
  numeric,
  className,
  ...props
}: ComponentProps<typeof TableCell> & { numeric?: boolean }) {
  return <TableCell className={cn(numeric && 'num', className)} {...props} />;
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
    <section className={cn('rounded-md border border-zinc-200 p-3 dark:border-zinc-800', className)}>
      <header className="flex items-center justify-between gap-2">
        {title ? <h2 className="text-sm font-semibold">{title}</h2> : null}
        {meta}
      </header>
      {children}
    </section>
  );
}
