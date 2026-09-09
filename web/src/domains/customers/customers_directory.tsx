import { useCallback, useEffect, useMemo } from 'react';

import type { Customer } from '@/api/types';
import { CustomersSelectionPanel } from '@/domains/customers/customers_selection_panel';
import { listPageRange } from '@/lib/list_page_stats';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { DirectoryPageShell } from '@/shell/directory_page_shell';
import { ControlPlaneSelectTable } from '@/shell/control_plane_select_table';

export type CustomersDirectoryProps = {
  items?: Customer[];
  total: number;
  limit: number;
  offset: number;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  freshnessLabel?: string;
  selectedCustomerId: string | null;
  onSelectedCustomerIdChange: (id: string | null) => void;
  onPageChange: (nextOffset: number) => void;
  onLimitChange: (limit: number) => void;
};

export function CustomersDirectory({
  items,
  total,
  limit,
  offset,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  freshnessLabel,
  selectedCustomerId,
  onSelectedCustomerIdChange,
  onPageChange,
  onLimitChange,
}: CustomersDirectoryProps) {
  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const pageRange = listPageRange(total, limit, offset, (items ?? []).length);
  const rangeLabel =
    pageRange.rangeStart > 0
      ? `${pageRange.rangeStart} - ${pageRange.rangeEnd} of ${total}`
      : '0 of 0';

  const operateRows = useMemo(
    () =>
      (items ?? [])
        .filter((customer): customer is Customer & { id: string } => Boolean(customer.id))
        .map((customer) => ({
          id: customer.id,
          label: customer.name ?? customer.id,
        })),
    [items]
  );

  const selectedCustomer = useMemo(
    () => (items ?? []).find((customer) => customer.id === selectedCustomerId),
    [items, selectedCustomerId]
  );

  const handleClearSelection = useCallback(() => {
    onSelectedCustomerIdChange(null);
  }, [onSelectedCustomerIdChange]);

  useEffect(() => {
    if (selectedCustomerId && !selectedCustomer) {
      onSelectedCustomerIdChange(null);
    }
  }, [onSelectedCustomerIdChange, selectedCustomer, selectedCustomerId]);

  return (
    <DirectoryPageShell
      aside={
        <CustomersSelectionPanel
          selectedCustomer={selectedCustomer}
          onClearSelection={handleClearSelection}
        />
      }
      badge={
        freshnessLabel ? (
          <span >
            {freshnessLabel}
          </span>
        ) : null
      }
      blockingErrorTitle="Could not load customers"
      fetchState={{ fetching, error, hasSnapshot }}
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
      skeletonColumns={2}
      title="Customers"
    >
      <ControlPlaneSelectTable
        disabled={fetching}
        emptyMessage="Customers are provisioned through billing and platform setup."
        nameColumnLabel="Customer"
        revalidating={listRevalidating}
        rows={operateRows}
        selectedId={selectedCustomerId}
        onSelectedIdChange={onSelectedCustomerIdChange}
      />
    </DirectoryPageShell>
  );
}
