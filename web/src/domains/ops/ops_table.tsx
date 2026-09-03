import type { ComponentProps, ReactNode } from 'react';

import { TableCell, TableHead, TableRow } from '@/components/ui/table';
import { cn } from '@/lib/utils';
import { DirectoryTable, TableBody, TableFooter, TableHeader } from '@/shell/directory_table';

/** Ops matrix chrome: sticky headers, zebra rows, numeric column alignment. */
export const OPS_DIRECTORY_TABLE_CLASS =
  'w-auto table-fixed text-[13px] [&_th]:sticky [&_th]:top-0 [&_th]:z-[2] [&_th]:bg-zinc-50 [&_th]:px-3 [&_th]:py-1.5 [&_th]:text-xs [&_th]:font-semibold [&_th]:text-zinc-500 [&_td]:border-b [&_td]:border-zinc-100 [&_td]:px-3 [&_td]:py-1.5 dark:[&_th]:bg-zinc-900 dark:[&_th]:text-zinc-400 dark:[&_td]:border-zinc-800 [&_td.num]:text-right [&_tbody_tr:nth-child(even)_td]:bg-zinc-50/50 dark:[&_tbody_tr:nth-child(even)_td]:bg-zinc-900/40';

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
