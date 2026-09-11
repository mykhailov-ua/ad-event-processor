import type { ReactNode } from 'react';

import { Table, TableCaption } from '@/components/ui/table';

export type UnstyledTableProps = {
  caption?: string;
  'aria-label'?: string;
  children: ReactNode;
  'data-testid'?: string;
};

export function UnstyledTable({
  caption,
  'aria-label': ariaLabel,
  children,
  'data-testid': testId,
}: UnstyledTableProps) {
  return (
    <Table aria-label={ariaLabel} bare data-role="table" data-testid={testId}>
      {caption ? <TableCaption data-role="table-caption">{caption}</TableCaption> : null}
      {children}
    </Table>
  );
}
