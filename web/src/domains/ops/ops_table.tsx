import type { ComponentProps, ReactNode } from 'react';

import { TableCell, TableHead, TableRow } from '@/components/ui/table';
import { cn } from '@/lib/utils';
import {
  DirectoryTable,
  SECTION_TABLE_HOST_CLASS,
  TableBody,
  TableFooter,
  TableHeader,
} from '@/shell/directory_table';
import { shellChrome } from '@/shell/shell_chrome';

/** Ops matrix chrome: sticky headers, zebra rows, numeric column alignment. */
export const OPS_DIRECTORY_TABLE_CLASS = cn(
  'w-full border-collapse text-[13px] leading-[18px]',
  '[&_th]:sticky [&_th]:top-0 [&_th]:z-[2] [&_th]:h-[34px] [&_th]:max-h-[34px] [&_th]:bg-card [&_th]:px-4 [&_th]:py-0 [&_th]:text-[11px] [&_th]:font-semibold [&_th]:uppercase [&_th]:leading-[14px] [&_th]:tracking-normal [&_th]:text-muted-foreground [&_th]:shadow-sm',
  '[&_td]:h-[34px] [&_td]:max-h-[34px] [&_td]:px-4 [&_td]:py-0',
  '[&_tbody_tr:nth-child(even)_td]:bg-muted/30',
  '[&_tbody_tr:last-child_td]:border-b-0'
);

/** Ops directory table shell; delegates border/scroll to DirectoryTable. */
export function OpsTable({
  head,
  children,
  foot,
  className,
  horizontalScroll = false,
}: {
  head: ReactNode;
  children: ReactNode;
  foot?: ReactNode;
  className?: string;
  horizontalScroll?: boolean;
}) {
  return (
    <DirectoryTable
      className={className}
      fixedLayout
      horizontalScroll={horizontalScroll}
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
  return <TableHead className={cn(numeric && 'text-right', className)} {...props} />;
}

export function OpsTableCell({
  numeric,
  className,
  ...props
}: ComponentProps<typeof TableCell> & { numeric?: boolean }) {
  return <TableCell className={cn(numeric && 'text-right', className)} {...props} />;
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
    <section className={cn(shellChrome.sectionPanelClass, className)}>
      <header className="grid grid-cols-[1fr_auto] items-center gap-2">
        {title ? <h2 className="text-[13px] font-semibold leading-[18px]">{title}</h2> : null}
        {meta}
      </header>
      <div className={SECTION_TABLE_HOST_CLASS}>{children}</div>
    </section>
  );
}
