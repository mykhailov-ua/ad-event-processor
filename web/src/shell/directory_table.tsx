import type { ComponentProps, CSSProperties, ReactNode, Ref } from 'react';
import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react';

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

const SORT_ICON_CLASS = 'h-3.5 w-3.5 shrink-0';
const DIRECTORY_TABLE_HEAD_ROW_CLASS = 'h-[34px]';

/** Subtle opacity while a directory list refetches (sort/pagination URL update). */
export function directoryTableRevalidatingClass(revalidating?: boolean): string {
  return cn('transition-opacity duration-150', revalidating && 'opacity-80');
}

/** Wrap section body when it may contain a DirectoryTable inside bordered chrome. */
export const SECTION_TABLE_HOST_CLASS = 'ui-section-table-host mt-2 min-w-0';

export type DirectoryTableProps = {
  children: ReactNode;
  className?: string;
  scrollable?: boolean;
  horizontalScroll?: boolean;
  fixedLayout?: boolean;
  /** Drop outer frame when parent section/card already has a border. */
  nested?: boolean;
  tableClassName?: string;
  tableStyle?: CSSProperties;
  tableRef?: Ref<HTMLTableElement>;
};

/** Pixel-width tables (campaign column probe) hug content; fluid tables fill the host. */
function directoryTableUsesContentWidth(tableStyle?: CSSProperties): boolean {
  const width = tableStyle?.width;
  if (width == null) {
    return false;
  }
  if (typeof width === 'number') {
    return width > 0;
  }
  if (typeof width === 'string') {
    const trimmed = width.trim();
    if (trimmed === '' || trimmed.endsWith('%')) {
      return false;
    }
    return /^\d+(\.\d+)?px$/.test(trimmed);
  }
  return false;
}

export function DirectoryTable({
  children,
  className,
  scrollable = false,
  horizontalScroll = false,
  fixedLayout = false,
  nested = false,
  tableClassName,
  tableStyle,
  tableRef,
}: DirectoryTableProps) {
  const contentWidth = directoryTableUsesContentWidth(tableStyle);

  return (
    <div
      data-directory-table=""
      className={cn(
        'ui-scrollbar min-w-0 border border-border',
        nested && 'border-0 shadow-none',
        scrollable && 'max-h-[min(70vh,48rem)] overflow-y-auto',
        horizontalScroll && 'overflow-x-auto',
        className
      )}
    >
      <Table
        bare
        ref={tableRef}
        className={cn(
          'border-collapse text-[13px] leading-[18px]',
          '[&_thead_th]:border-b [&_thead_th]:border-border',
          '[&_tbody_td]:border-b [&_tbody_td]:border-border',
          '[&_tbody_tr:last-child_td]:border-b-0',
          contentWidth ? 'w-max' : 'w-full',
          fixedLayout && '[&_td]:whitespace-nowrap [&_th]:whitespace-nowrap',
          tableClassName
        )}
        style={tableStyle}
      >
        {children}
      </Table>
    </div>
  );
}

export { TableBody, TableCell, TableFooter, TableHeader, TableRow } from '@/components/ui/table';

type HeadAlign = 'start' | 'end';

function DirectoryTableHeadShell({ className, ...props }: ComponentProps<typeof TableHead>) {
  return (
    <TableHead
      className={cn(DIRECTORY_TABLE_HEAD_ROW_CLASS, 'bg-admin-table-header p-0 backdrop-blur-sm', className)}
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
        DIRECTORY_TABLE_HEAD_ROW_CLASS,
        adminKit.directoryTableHeadInnerClass,
        align === 'end' ? 'justify-end text-right' : 'justify-start text-left'
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
        <span className="whitespace-nowrap">{children}</span>
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
        aria-sort={active ? (activeOrder === 'asc' ? 'ascending' : 'descending') : 'none'}
        className={cn(
          DIRECTORY_TABLE_HEAD_ROW_CLASS,
          adminKit.directoryTableHeadInnerClass,
          'transition-colors hover:text-foreground',
          align === 'end' ? 'justify-end text-right' : 'justify-start text-left',
          active && 'text-foreground'
        )}
        onClick={() => onSort(sortField)}
        type="button"
      >
        <span className="whitespace-nowrap">{label}</span>
        <Icon aria-hidden className={cn(SORT_ICON_CLASS, active ? 'opacity-90' : 'opacity-45')} />
      </button>
    </DirectoryTableHeadShell>
  );
}
