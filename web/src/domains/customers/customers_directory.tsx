import { useCallback, useEffect, useMemo } from 'react';
import { Link, useLocation } from 'react-router-dom';

import type { Customer } from '@/api/types';
import { InAppLink } from '@/shell/in_app_link';
import { buildExportHubHref } from '@/lib/export_hub_paths';
import { rememberExportHubReturnPath } from '@/lib/export_hub_return';
import { CustomersSelectionPanel } from '@/domains/customers/customers_selection_panel';
import { listPageRange } from '@/lib/list_page_stats';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { DirectoryPageShell } from '@/shell/directory_page_shell';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { DirectoryRowActionsMenu } from '@/shell/directory_row_actions_menu';
import { PrimaryActionButton } from '@/shell/action_buttons';

function buildCustomerOverviewFields(customer: Customer): DirectoryOverviewField[] {
  return [
    { label: 'Name', value: customer.name ?? customer.id },
    { label: 'ID', value: customer.id ?? '-' },
    { label: 'Balance', value: customer.balance ?? '-' },
    { label: 'Currency', value: customer.currency ?? '-' },
    { label: 'Active campaigns', value: customer.active_campaigns ?? '-' },
    { label: 'Total spend', value: customer.total_spend ?? '-' },
  ];
}

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
  const location = useLocation();
  const exportHubHref = useMemo(
    () =>
      buildExportHubHref({
        entry: 'customers-directory',
        reportKey: 'customer-portfolio',
        kind: 'report',
        returnTo: `${location.pathname}${location.search}`,
      }),
    [location.pathname, location.search]
  );

  useEffect(() => {
    rememberExportHubReturnPath(`${location.pathname}${location.search}`);
  }, [location.pathname, location.search]);

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const pageRange = listPageRange(total, limit, offset, (items ?? []).length);
  const rangeLabel =
    pageRange.rangeStart > 0
      ? `${pageRange.rangeStart} - ${pageRange.rangeEnd} of ${total}`
      : '0 of 0';

  const operateRows = useMemo(
    () =>
      directoryOperateRows(
        items,
        (customer) => customer.id,
        (customer) => customer.name ?? customer.id ?? ''
      ),
    [items]
  );

  const selectedCustomer = useMemo(
    () => (items ?? []).find((customer) => customer.id === selectedCustomerId),
    [items, selectedCustomerId]
  );
  const customerById = useMemo(
    () => directoryRecordMap(items, (customer) => customer.id),
    [items]
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
          <span>
            {freshnessLabel}
          </span>
        ) : null
      }
      blockingErrorTitle="Could not load customers"
      description={
        <>
          Customer directory for operational billing context. Portfolio KPI exports run on{' '}
          <InAppLink className="text-primary hover:underline" to={exportHubHref}>
            Export hub
          </InAppLink>{' '}
          (Operations nav).
        </>
      }
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
      skeletonColumns={3}
      title="Customers"
    >
      <DirectorySelectOverviewTable
        buildOverviewFields={buildCustomerOverviewFields}
        disabled={fetching}
        emptyMessage="Customers are provisioned through billing and platform setup."
        nameColumnLabel="Customer"
        overviewFooter={(customer) =>
          customer.id ? (
            <>
              <PrimaryActionButton asChild>
                <Link to={`/customers/${customer.id}`}>Open detail</Link>
              </PrimaryActionButton>
              <InAppLink to={exportHubHref}>Export hub</InAppLink>
            </>
          ) : null
        }
        overviewTitle={(customer) => customer.name ?? customer.id ?? ''}
        recordById={customerById}
        revalidating={listRevalidating}
        renderActions={(row, _customer, openOverview) => (
          <DirectoryRowActionsMenu
            ariaLabel={`Customer actions ${row.id}`}
            disabled={fetching}
            onOverview={openOverview}
          />
        )}
        rows={operateRows}
        selectedId={selectedCustomerId}
        onSelectedIdChange={onSelectedCustomerIdChange}
      />
    </DirectoryPageShell>
  );
}
