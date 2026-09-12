import type { ReactNode } from 'react';

import { Checkbox } from '@/components/ui/checkbox';
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
import { TableHost } from '@/shell/ui_bands';

export type ControlPlaneSelectRow = {
  id: string;
  label: ReactNode;
};

export type ControlPlaneSelectTableProps = {
  rows: ControlPlaneSelectRow[];
  selectedId: string | null;
  onSelectedIdChange: (id: string | null) => void;
  emptyMessage?: string;
  selectColumnLabel?: string;
  selectAriaLabel?: (row: ControlPlaneSelectRow, selected: boolean) => string;
  nameColumnLabel?: string;
  actionsColumnLabel?: string;
  renderActions?: (row: ControlPlaneSelectRow) => ReactNode;
  revalidating?: boolean;
  disabled?: boolean;
  nameCellClassName?: string;
  onNameClick?: (row: ControlPlaneSelectRow) => void;
};

export function ControlPlaneSelectTable({
  rows,
  selectedId,
  onSelectedIdChange,
  emptyMessage = 'No items.',
  selectColumnLabel = 'Select',
  selectAriaLabel,
  nameColumnLabel = 'Name',
  actionsColumnLabel = 'Actions',
  renderActions,
  revalidating = false,
  disabled = false,
  nameCellClassName,
  onNameClick,
}: ControlPlaneSelectTableProps) {
  if (rows.length === 0) {
    return <EmptyState description={emptyMessage} title="No items" variant="blank-slate" />;
  }

  return (
    <TableHost>
      <DirectoryTable
        fixedLayout
        horizontalScroll={false}
        nested
        scrollable={false}
        tableClassName="w-full table-fixed"
      >
        <TableHeader>
          <TableRow className={adminKit.tableRowHeight}>
            <DirectoryTableHead className="w-12">
              <span className="sr-only">{selectColumnLabel}</span>
            </DirectoryTableHead>
            <DirectoryTableHead>{nameColumnLabel}</DirectoryTableHead>
            {renderActions ? (
              <DirectoryTableHead className="w-[5.5rem] text-right">
                {actionsColumnLabel}
              </DirectoryTableHead>
            ) : null}
          </TableRow>
        </TableHeader>
        <TableBody className={directoryTableRevalidatingClass(revalidating)}>
          {rows.map((row) => {
            const isSelected = selectedId === row.id;
            return (
              <TableRow
                key={row.id}
                aria-selected={isSelected}
                className={cn(adminKit.tableRowHeight, isSelected && 'bg-admin-selection')}
                data-selected={isSelected ? 'true' : undefined}
              >
                <TableCell className={cn(adminChrome.tableCell, 'w-12')}>
                  <Checkbox
                    aria-label={
                      selectAriaLabel?.(row, isSelected) ??
                      (isSelected ? `Deselect ${String(row.label)}` : `Select ${String(row.label)}`)
                    }
                    checked={isSelected}
                    disabled={disabled}
                    onCheckedChange={(checked) => onSelectedIdChange(checked ? row.id : null)}
                  />
                </TableCell>
                <TableCell
                  className={cn(adminChrome.tableCell, 'whitespace-normal', nameCellClassName)}
                >
                  {onNameClick ? (
                    <button
                      className="w-full min-w-0 text-left text-foreground hover:underline"
                      disabled={disabled}
                      type="button"
                      onClick={() => onNameClick(row)}
                    >
                      {row.label}
                    </button>
                  ) : (
                    row.label
                  )}
                </TableCell>
                {renderActions ? (
                  <TableCell className={cn(adminChrome.tableCell, 'text-right')}>
                    {renderActions(row)}
                  </TableCell>
                ) : null}
              </TableRow>
            );
          })}
        </TableBody>
      </DirectoryTable>
    </TableHost>
  );
}
