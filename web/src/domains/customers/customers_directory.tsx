import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Table, TableBody, TableHeader } from '@/components/ui/table';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import type { Customer } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { listPageRange } from '@/lib/list_page_stats';
import { clampListLimit } from '@/lib/list_query';
import { cn } from '@/lib/utils';

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

function SortableHeader({
  active,
  activeOrder,
  disabled,
  label,
  onSort,
}: {
  active: boolean;
  activeOrder: SortOrder;
  disabled?: boolean;
  label: string;
  onSort: () => void;
}) {
  return (
    <Button
      className={cn(active && 'font-semibold')}
      disabled={disabled}
      type="button"
      variant="ghost"
      onClick={onSort}
    >
      <span>{label}</span>
      <span aria-hidden className="admin-muted inline-flex">
        {active ? (
          activeOrder === 'asc' ? (
            <ArrowUp className="h-3 w-3" />
          ) : (
            <ArrowDown className="h-3 w-3" />
          )
        ) : (
          <ArrowUpDown className="h-3 w-3 opacity-50" />
        )}
      </span>
    </Button>
  );
}

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
  const [pageSizeDraft, setPageSizeDraft] = useState(String(limit));

  useEffect(() => {
    setPageSizeDraft(String(limit));
  }, [limit]);

  const handlePageSizeCommit = useCallback(
    (raw: string) => {
      const next = clampListLimit(Number.parseInt(raw, 10));
      setPageSizeDraft(String(next));
      if (next !== limit) {
        onLimitChange(next);
      }
    },
    [limit, onLimitChange],
  );

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
      badge={freshnessLabel ? <span className="admin-chip">{freshnessLabel}</span> : null}
      footer={
        <>
          <label className="admin-label">
            Per page
            <input
              className="admin-select"
              disabled={fetching}
              inputMode="numeric"
              value={pageSizeDraft}
              onBlur={() => handlePageSizeCommit(pageSizeDraft)}
              onChange={(event) => setPageSizeDraft(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  event.preventDefault();
                  handlePageSizeCommit(pageSizeDraft);
                }
              }}
            />
          </label>
          <Button
            disabled={fetching || !canGoPrev}
            type="button"
            variant="secondary"
            onClick={() => onPageChange(Math.max(0, offset - limit))}
          >
            Previous
          </Button>
          <Button
            disabled={fetching || !canGoNext}
            type="button"
            variant="secondary"
            onClick={() => onPageChange(offset + limit)}
          >
            Next
          </Button>
          <span className="admin-muted tabular-nums">{rangeLabel}</span>
        </>
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
        <div className="admin-table-wrap">
          <Table bare className="admin-table">
              <TableHeader>
                <tr>
                  <th className="w-[28%]">
                    <SortableHeader
                      active={appliedSort === 'name'}
                      activeOrder={appliedOrder}
                      disabled={fetching}
                      label="Name"
                      onSort={() => onColumnSort('name')}
                    />
                  </th>
                  <th className="num w-[11%]">
                    <SortableHeader
                      active={appliedSort === 'balance'}
                      activeOrder={appliedOrder}
                      disabled={fetching}
                      label="Balance"
                      onSort={() => onColumnSort('balance')}
                    />
                  </th>
                  <th className="w-[7%]">Currency</th>
                  <th className="w-[14%]">Cost center</th>
                  <th className="num w-[10%]">
                    <SortableHeader
                      active={appliedSort === 'active_campaigns'}
                      activeOrder={appliedOrder}
                      disabled={fetching}
                      label="Active"
                      onSort={() => onColumnSort('active_campaigns')}
                    />
                  </th>
                  <th className="num w-[12%]">Total spend</th>
                  <th className="num w-[18%]">
                    <SortableHeader
                      active={appliedSort === 'created_at'}
                      activeOrder={appliedOrder}
                      disabled={fetching}
                      label="Created"
                      onSort={() => onColumnSort('created_at')}
                    />
                  </th>
                </tr>
              </TableHeader>
              <TableBody>
                {items.map((customer) => {
                  const createdLabel = displayTimestamp(
                    customer.created_at,
                    customer.created_at_display,
                  );
                  return (
                    <tr key={customer.id ?? customer.name}>
                      <td className="max-w-0 truncate font-medium">
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
                      </td>
                      <td className="num">{customer.balance ?? ''}</td>
                      <td className="truncate">{customer.currency ?? ''}</td>
                      <td className="max-w-0 truncate" title={customer.cost_center ?? undefined}>
                        {customer.cost_center ?? ''}
                      </td>
                      <td className="num">
                        {customer.active_campaigns ?? ''}
                      </td>
                      <td className="num">{customer.total_spend ?? ''}</td>
                      <td className="admin-muted max-w-0 truncate" title={createdLabel}>
                        {createdLabel}
                      </td>
                    </tr>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        )}
    </PageLayout>
  );
}
