import { Link } from 'react-router-dom';

import type { Customer } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { listPageRange } from '@/lib/list_page_stats';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import {
  DirectoryTable,
  DirectoryTableHead,
  SortableTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageLayout } from '@/shell/page_layout';
import { PageSkeleton } from '@/shell/page_skeleton';

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
      ? `${pageRange.rangeStart} - ${pageRange.rangeEnd} of ${total}`
      : '0 of 0';

  return (
    <PageLayout
      badge={freshnessLabel ? <span className="inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-xs dark:border-zinc-700 dark:bg-zinc-900">{freshnessLabel}</span> : null}
      footer={
        <DirectoryPaginationFooter
          canGoNext={canGoNext}
          canGoPrev={canGoPrev}
          disabled={fetching}
          limit={limit}
          pageSizeId="customers-page-size"
          rangeLabel={rangeLabel}
          onLimitChange={onLimitChange}
          onNext={() => onPageChange(offset + limit)}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
        />
      }
      title="Customers"
    >
      {items.length === 0 ? (
        <EmptyState
          actionHref="/docs"
          actionLabel="View documentation"
          description="Customers are provisioned through billing and platform setup."
          title="No customers"
          variant="blank-slate"
        />
      ) : (
        <DirectoryTable fixedLayout>
          <TableHeader>
            <TableRow>
              <SortableTableHead
                activeOrder={appliedOrder}
                activeSort={appliedSort}
                className="w-[28%]"
                label="Name"
                sortField="name"
                onSort={(field) => onColumnSort(field as CustomerSortField)}
              />
              <SortableTableHead
                activeOrder={appliedOrder}
                activeSort={appliedSort}
                className="w-[11%]"
                label="Balance"
                numeric
                sortField="balance"
                onSort={(field) => onColumnSort(field as CustomerSortField)}
              />
              <DirectoryTableHead className="w-[7%]">Currency</DirectoryTableHead>
              <DirectoryTableHead className="w-[14%]">Cost center</DirectoryTableHead>
              <SortableTableHead
                activeOrder={appliedOrder}
                activeSort={appliedSort}
                className="w-[10%]"
                label="Active"
                numeric
                sortField="active_campaigns"
                onSort={(field) => onColumnSort(field as CustomerSortField)}
              />
              <DirectoryTableHead align="end" className="w-[12%]">
                Total spend
              </DirectoryTableHead>
              <SortableTableHead
                activeOrder={appliedOrder}
                activeSort={appliedSort}
                className="w-[18%]"
                label="Created"
                sortField="created_at"
                onSort={(field) => onColumnSort(field as CustomerSortField)}
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
                        className="block truncate"
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
                    className="max-w-0 truncate text-muted-foreground"
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
    </PageLayout>
  );
}
