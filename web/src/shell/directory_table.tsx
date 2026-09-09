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
export const SECTION_TABLE_HOST_CLASS = '';

export type DirectoryTableProps = {
  children: ReactNode;
  hostClassName?: string;
  hostRef?: Ref<HTMLDivElement>;
  hostStyle?: CSSProperties;
  pinnedEdgeWidthPx?: number;
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
  hostClassName,
  hostRef,
  hostStyle,
  pinnedEdgeWidthPx,
  scrollable = false,
  horizontalScroll = false,
  fixedLayout = false,
  nested = false,
  tableClassName,
  tableStyle,
  tableRef,
}: DirectoryTableProps) {
  const contentWidth = directoryTableUsesContentWidth(tableStyle);
  const explicitTableWidth = tableStyle?.width != null;

  return (
    <div
      ref={hostRef}
      data-directory-table=""
     
     
    >
      {pinnedEdgeWidthPx != null && pinnedEdgeWidthPx > 0 ? (
        <div
          aria-hidden
         
          data-directory-pinned-edge=""
         
        />
      ) : null}
      <Table
        bare
        ref={tableRef}
       
       
      >
        {children}
      </Table>
    </div>
  );
}

export { TableBody, TableCell, TableFooter, TableHeader, TableRow } from '@/components/ui/table';

type HeadAlign = 'start' | 'end';

function DirectoryTableHeadShell({ ...props }: ComponentProps<typeof TableHead>) {
  return (
    <TableHead
     
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
     
    >
      {children}
      {reserveSortIcon ? <span aria-hidden  /> : null}
    </div>
  );
}

export function DirectoryTableHead({
  align = 'start',
  children,
  ...props
}: Omit<ComponentProps<typeof TableHead>, 'align'> & { align?: HeadAlign }) {
  return (
    <DirectoryTableHeadShell  {...props}>
      <DirectoryTableHeadContent align={align} reserveSortIcon={align === 'end'}>
        <span >{children}</span>
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
  numeric?: boolean;
};

export function SortableTableHead({
  label,
  sortField,
  activeSort,
  activeOrder,
  onSort,
  numeric = false,
}: SortableTableHeadProps) {
  const active = activeSort === sortField;
  const Icon = active ? (activeOrder === 'asc' ? ArrowUp : ArrowDown) : ArrowUpDown;
  const align: HeadAlign = numeric ? 'end' : 'start';

  return (
    <DirectoryTableHeadShell >
      <button
        aria-label={`Sort by ${label}`}
        aria-sort={active ? (activeOrder === 'asc' ? 'ascending' : 'descending') : 'none'}
       
        onClick={() => onSort(sortField)}
        type="button"
      >
        <span >{label}</span>
        <Icon aria-hidden  />
      </button>
    </DirectoryTableHeadShell>
  );
}
