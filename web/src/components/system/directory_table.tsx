import type { ComponentProps, ReactNode } from 'react';
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react';

import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn } from '@/lib/utils';

const SORT_ICON_CLASS = 'h-3.5 w-3.5 shrink-0';

export type DirectoryTableProps = {
  children: ReactNode;
  className?: string;
  scrollable?: boolean;
  fixedLayout?: boolean;
};

export function DirectoryTable({
  children,
  className,
  scrollable = false,
  fixedLayout = false,
}: DirectoryTableProps) {
  return (
    <div className={cn('overflow-hidden rounded-2xl border border-border/40 bg-card/40', className)}>
      <div
        className={cn(
          'relative w-full min-w-0 max-w-full',
          scrollable
            ? 'ui-scrollbar max-h-[min(70vh,48rem)] overflow-auto'
            : 'overflow-x-auto',
        )}
      >
        <table
          className={cn(
            'w-full caption-bottom text-sm',
            fixedLayout && 'table-fixed [&_td]:whitespace-nowrap [&_th]:whitespace-nowrap',
          )}
        >
          {children}
        </table>
      </div>
    </div>
  );
}

export { TableBody, TableCell, TableHeader, TableRow };

type HeadAlign = 'start' | 'end';

function DirectoryTableHeadShell({
  className,
  ...props
}: ComponentProps<typeof TableHead>) {
  return (
    <TableHead
      className={cn('h-10 bg-card/90 p-0 backdrop-blur-sm', className)}
      {...props}
    />
  );
}

function DirectoryTableHeadContent({
  align = 'start',
  children,
  reserveSortIcon = false,
}: {
  align?: HeadAlign;
  children: ReactNode;
  reserveSortIcon?: boolean;
}) {
  return (
    <div
      className={cn(
        'flex h-10 w-full items-center gap-1.5 px-2 font-medium text-muted-foreground',
        align === 'end' ? 'justify-end text-right' : 'justify-start text-left',
      )}
    >
      {children}
      {reserveSortIcon ? <span aria-hidden className={SORT_ICON_CLASS} /> : null}
    </div>
  );
}

export function DirectoryTableHead({
  align = 'start',
  className,
  children,
  ...props
}: Omit<ComponentProps<typeof TableHead>, 'align'> & { align?: HeadAlign }) {
  return (
    <DirectoryTableHeadShell className={className} {...props}>
      <DirectoryTableHeadContent align={align} reserveSortIcon={align === 'end'}>
        <span className="truncate">{children}</span>
      </DirectoryTableHeadContent>
    </DirectoryTableHeadShell>
  );
}

export type SortableTableHeadProps = {
  label: string;
  sortField: string;
  activeSort: string;
  activeOrder: 'asc' | 'desc';
  onSort: (field: string) => void;
  className?: string;
  numeric?: boolean;
};

export function SortableTableHead({
  label,
  sortField,
  activeSort,
  activeOrder,
  onSort,
  className,
  numeric = false,
}: SortableTableHeadProps) {
  const active = activeSort === sortField;
  const Icon = active ? (activeOrder === 'asc' ? ArrowUp : ArrowDown) : ArrowUpDown;
  const align: HeadAlign = numeric ? 'end' : 'start';

  return (
    <DirectoryTableHeadShell className={className}>
      <button
        aria-label={`Sort by ${label}`}
        aria-sort={
          active ? (activeOrder === 'asc' ? 'ascending' : 'descending') : 'none'
        }
        className={cn(
          'flex h-10 w-full items-center gap-1.5 px-2 font-medium text-muted-foreground transition-colors hover:text-foreground',
          align === 'end' ? 'justify-end text-right' : 'justify-start text-left',
          active && 'text-foreground',
        )}
        onClick={() => onSort(sortField)}
        type="button"
      >
        <span className="truncate">{label}</span>
        <Icon
          aria-hidden
          className={cn(SORT_ICON_CLASS, active ? 'opacity-90' : 'opacity-45')}
        />
      </button>
    </DirectoryTableHeadShell>
  );
}
