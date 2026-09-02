import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
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
    <button
      className={cn(
        'inline-flex w-full items-center gap-1 justify-start text-left',
        active && 'text-[var(--campaigns-ws-text)]',
      )}
      disabled={disabled}
      type="button"
      onClick={onSort}
    >
      <span className="truncate">{label}</span>
      <span aria-hidden className="text-[0.625rem] leading-none">
        {active ? (activeOrder === 'asc' ? '▲' : '▼') : '↕'}
      </span>
    </button>
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
    <div className="campaigns-list-workspace flex min-h-full flex-col">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--campaigns-ws-border)] bg-white px-3 py-2">
        <div className="flex min-w-0 items-center gap-2">
          <h1 className="text-lg font-semibold text-[var(--campaigns-ws-text)]">Customers</h1>
          {freshnessLabel ? (
            <span className="rounded border border-[var(--campaigns-ws-border)] bg-[#f7f7f7] px-2 py-0.5 text-xs text-[var(--campaigns-ws-muted)]">
              {freshnessLabel}
            </span>
          ) : null}
        </div>

        <div className="flex flex-wrap items-center gap-2 text-sm text-[var(--campaigns-ws-muted)]">
          <label className="flex items-center gap-1.5">
            <span>Per page</span>
            <input
              className="campaigns-list-workspace-page-size tabular-nums"
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
          <button
            className="campaigns-list-workspace-btn-secondary"
            disabled={fetching || !canGoPrev}
            type="button"
            onClick={() => onPageChange(Math.max(0, offset - limit))}
          >
            Previous
          </button>
          <button
            className="campaigns-list-workspace-btn-secondary"
            disabled={fetching || !canGoNext}
            type="button"
            onClick={() => onPageChange(offset + limit)}
          >
            Next
          </button>
          <span className="tabular-nums">{rangeLabel}</span>
        </div>
      </div>

      <div aria-atomic="true" aria-live="polite" className="flex min-h-0 flex-1 flex-col bg-white">
        {items.length === 0 ? (
          <EmptyState
            variant="blank-slate"
            title="No customers"
            description="Customers are provisioned through billing and platform setup."
            actionLabel="View documentation"
            actionHref="/docs"
          />
        ) : (
          <div className="campaigns-list-workspace-table-wrap ui-scrollbar">
            <table className="campaigns-list-workspace-table campaigns-list-workspace-table--no-row-accent">
              <thead>
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
                  <th className="campaigns-list-workspace-num w-[11%]">
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
                  <th className="campaigns-list-workspace-num w-[10%]">
                    <SortableHeader
                      active={appliedSort === 'active_campaigns'}
                      activeOrder={appliedOrder}
                      disabled={fetching}
                      label="Active"
                      onSort={() => onColumnSort('active_campaigns')}
                    />
                  </th>
                  <th className="campaigns-list-workspace-num w-[12%]">Total spend</th>
                  <th className="campaigns-list-workspace-num w-[18%]">
                    <SortableHeader
                      active={appliedSort === 'created_at'}
                      activeOrder={appliedOrder}
                      disabled={fetching}
                      label="Created"
                      onSort={() => onColumnSort('created_at')}
                    />
                  </th>
                </tr>
              </thead>
              <tbody>
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
                            className="campaigns-list-workspace-link block truncate"
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
                      <td className="campaigns-list-workspace-num">{customer.balance ?? ''}</td>
                      <td className="truncate">{customer.currency ?? ''}</td>
                      <td className="max-w-0 truncate" title={customer.cost_center ?? undefined}>
                        {customer.cost_center ?? ''}
                      </td>
                      <td className="campaigns-list-workspace-num">
                        {customer.active_campaigns ?? ''}
                      </td>
                      <td className="campaigns-list-workspace-num">{customer.total_spend ?? ''}</td>
                      <td className="campaigns-list-workspace-num max-w-0 truncate" title={createdLabel}>
                        {createdLabel}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
