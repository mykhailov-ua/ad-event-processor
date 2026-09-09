import type { ReactNode } from 'react';

import { Button } from '@/components/ui/button';
import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import { EmptyState } from '@/shell/empty_state';

export type ControlPlaneSelectRow = {
  id: string;
  label: ReactNode;
};

export type ControlPlaneSelectTableProps = {
  rows: ControlPlaneSelectRow[];
  selectedId: string | null;
  onSelectedIdChange: (id: string | null) => void;
  emptyMessage?: string;
  selectLabel?: string;
  selectedLabel?: string;
  selectColumnLabel?: string;
  nameColumnLabel?: string;
  revalidating?: boolean;
  disabled?: boolean;
  nameCellClassName?: string;
};

export function ControlPlaneSelectTable({
  rows,
  selectedId,
  onSelectedIdChange,
  emptyMessage = 'No items.',
  selectLabel = 'Select',
  selectedLabel = 'Selected',
  selectColumnLabel = 'Select',
  nameColumnLabel = 'Name',
  revalidating = false,
  disabled = false,
  nameCellClassName,
}: ControlPlaneSelectTableProps) {
  if (rows.length === 0) {
    return (
      <EmptyState
        description={emptyMessage}
        title="No items"
        variant="blank-slate"
      />
    );
  }

  return (
    <DirectoryTable
      fixedLayout
      horizontalScroll={false}
      nested
      scrollable={false}
      tableClassName="w-full table-fixed"
    >
      <TableHeader>
        <TableRow className={adminKit.tableRowHeight}>
          <DirectoryTableHead className="w-[7.5rem]">{selectColumnLabel}</DirectoryTableHead>
          <DirectoryTableHead>{nameColumnLabel}</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody className={directoryTableRevalidatingClass(revalidating)}>
        {rows.map((row) => {
          const isSelected = selectedId === row.id;
          return (
            <TableRow
              key={row.id}
              aria-selected={isSelected}
              className={cn(
                adminKit.tableRowHeight,
                isSelected && 'bg-admin-selection'
              )}
              data-selected={isSelected ? 'true' : undefined}
            >
              <TableCell className={adminChrome.tableCell}>
                <Button
                  aria-pressed={isSelected}
                  disabled={disabled}
                  type="button"
                  variant={isSelected ? 'accent' : 'outline'}
                  onClick={() => onSelectedIdChange(isSelected ? null : row.id)}
                >
                  {isSelected ? selectedLabel : selectLabel}
                </Button>
              </TableCell>
              <TableCell
                className={cn(
                  adminChrome.tableCell,
                  'whitespace-normal',
                  nameCellClassName
                )}
              >
                {row.label}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </DirectoryTable>
  );
}
