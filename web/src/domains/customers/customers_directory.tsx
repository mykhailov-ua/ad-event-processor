import { Link } from 'react-router-dom';

import {
  DirectoryTable,
  DirectoryTableHead,
  SortableTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/components/system/directory_table';
import { PageChrome } from '@/components/system/page_chrome';
import { PaginationPageSize } from '@/components/system/pagination_page_size';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import { Badge } from '@/components/ui/badge';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import type { Customer } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { listPageRange } from '@/lib/list_page_stats';

export type CustomerSortField = 'name' | 'created_at' | 'balance' | 'active_campaigns';
export type SortOrder = 'asc' | 'desc';

export type CustomersDirectoryProps = {
  items: Customer[];
  total: number;
  limit: number;
  offset: number;
  appliedSort: CustomerSortField;
  appliedOrder: SortOrder;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  freshnessLabel?: string;
  onColumnSort: (field: CustomerSortField) => void;
  onPageChange: (nextOffset: number) => void;
  onLimitChange: (limit: number) => void;
};

export function CustomersDirectory({
  items,
  total,
  limit,
  offset,
  appliedSort,
  appliedOrder,
  fetching,
  error,
  hasSnapshot,
  freshnessLabel,
  onColumnSort,
  onPageChange,
  onLimitChange,
}: CustomersDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={4} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load customers" message={error.message} />;
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const pageRange = listPageRange(total, limit, offset, items.length);
  const rangeLabel =
    pageRange.rangeStart > 0
      ? `${pageRange.rangeStart}-${pageRange.rangeEnd} of ${total}`
      : null;

  return (
    <PageChrome
      title="Customers"
      badge={
        freshnessLabel ? (
          <Badge variant="secondary">{freshnessLabel}</Badge>
        ) : undefined
      }
    >
      <div className="flex flex-wrap items-end justify-between gap-3">
        {rangeLabel ? (
          <p className="text-sm text-muted-foreground tabular-nums">{rangeLabel}</p>
        ) : (
          <span aria-hidden className="text-sm" />
        )}
        <div className="flex flex-wrap items-end gap-2">
          <PaginationPageSize
            id="customers-limit"
            value={limit}
            disabled={fetching}
            onChange={onLimitChange}
          />
          <PaginationPrevNext
            className="w-[12rem]"
            canGoPrev={canGoPrev}
            canGoNext={canGoNext}
            disabled={fetching}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
            onNext={() => onPageChange(offset + limit)}
          />
        </div>
      </div>

      <div aria-atomic="true" aria-live="polite">
        {items.length === 0 ? (
          <EmptyState
            variant="blank-slate"
            title="No customers"
            description="Customers are provisioned through billing and platform setup."
            actionLabel="View documentation"
            actionHref="/docs"
          />
        ) : (
          <DirectoryTable fixedLayout>
            <TableHeader>
              <TableRow>
                <SortableTableHead
                  className="w-[28%]"
                  activeOrder={appliedOrder}
                  activeSort={appliedSort}
                  label="Name"
                  onSort={(field) => onColumnSort(field as CustomerSortField)}
                  sortField="name"
                />
                <SortableTableHead
                  className="w-[11%]"
                  activeOrder={appliedOrder}
                  activeSort={appliedSort}
                  label="Balance"
                  numeric
                  onSort={(field) => onColumnSort(field as CustomerSortField)}
                  sortField="balance"
                />
                <DirectoryTableHead className="w-[7%]">Currency</DirectoryTableHead>
                <DirectoryTableHead className="w-[14%]">Cost center</DirectoryTableHead>
                <SortableTableHead
                  className="w-[10%]"
                  activeOrder={appliedOrder}
                  activeSort={appliedSort}
                  label="Active"
                  numeric
                  onSort={(field) => onColumnSort(field as CustomerSortField)}
                  sortField="active_campaigns"
                />
                <DirectoryTableHead align="end" className="w-[12%]">
                  Total spend
                </DirectoryTableHead>
                <SortableTableHead
                  className="w-[18%]"
                  activeOrder={appliedOrder}
                  activeSort={appliedSort}
                  label="Created"
                  numeric
                  onSort={(field) => onColumnSort(field as CustomerSortField)}
                  sortField="created_at"
                />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((customer) => {
                const createdLabel = displayTimestamp(
                  customer.created_at,
                  customer.created_at_display,
                );
                return (
                  <TableRow key={customer.id ?? customer.name}>
                    <TableCell className="max-w-0 truncate font-medium">
                      {customer.id ? (
                        <Link
                          className="block truncate text-primary hover:underline"
                          title={customer.name ?? customer.id}
                          to={`/customers/${customer.id}`}
                        >
                          {customer.name ?? customer.id}
                        </Link>
                      ) : (
                        <span className="block truncate" title={customer.name ?? undefined}>
                          {customer.name}
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">{customer.balance ?? ''}</TableCell>
                    <TableCell className="truncate">{customer.currency ?? ''}</TableCell>
                    <TableCell className="max-w-0 truncate" title={customer.cost_center ?? undefined}>
                      {customer.cost_center ?? ''}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {customer.active_campaigns ?? ''}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">{customer.total_spend ?? ''}</TableCell>
                    <TableCell
                      className="max-w-0 truncate text-right tabular-nums"
                      title={createdLabel}
                    >
                      {createdLabel}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </DirectoryTable>
        )}
      </div>
    </PageChrome>
  );
}
