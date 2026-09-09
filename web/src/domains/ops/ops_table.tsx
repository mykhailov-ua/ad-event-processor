import type { ComponentProps, ReactNode } from 'react';

import { TableCell, TableHead, TableRow } from '@/components/ui/table';
import { adminSpacing } from '@/lib/admin_spacing';
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
export const OPS_DIRECTORY_TABLE_CLASS = cn(adminSpacing.opsDirectoryTable);

/** Ops directory table shell; delegates border/scroll to DirectoryTable. */
export function OpsTable({
  head,
  children,
  foot,
  horizontalScroll = false,
}: {
  head: ReactNode;
  children: ReactNode;
  foot?: ReactNode;
  horizontalScroll?: boolean;
}) {
  return (
    <DirectoryTable
     
      fixedLayout
      horizontalScroll={horizontalScroll}
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
  ...props
}: ComponentProps<typeof TableHead> & { numeric?: boolean }) {
  return <TableHead  {...props} />;
}

export function OpsTableCell({
  numeric,
  ...props
}: ComponentProps<typeof TableCell> & { numeric?: boolean }) {
  return <TableCell  {...props} />;
}

/** Section title above a table or block; no extra panel border. */
export function OpsBlock({
  title,
  meta,
  children,
}: {
  title?: ReactNode;
  meta?: ReactNode;
  children: ReactNode;
}) {
  if (!title && !meta) {
    return <>{children}</>;
  }

  return (
    <section >
      <header >
        {title ? <h2 >{title}</h2> : null}
        {meta}
      </header>
      <div >{children}</div>
    </section>
  );
}
