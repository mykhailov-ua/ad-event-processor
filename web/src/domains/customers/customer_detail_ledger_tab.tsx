import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import type { BalanceLedgerEntry } from '@/api/types';
import { CustomerDetailLedgerEntryTable } from '@/domains/customers/customer_detail_ledger_entry_table';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { COMPACT_TOOLBAR_ROW_CLASS } from '@/shell/filter_panel';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { CustomerTabShell } from '@/shell/customer_tab_shell';
import { adminTypography, customerDetailSectionClass } from '@/lib/admin_kit';
import { buildExportHubHref } from '@/lib/export_hub_paths';
import { cn } from '@/lib/utils';

export type CustomerDetailLedgerTabProps = {
  customerId?: string;
  items?: BalanceLedgerEntry[];
  total: number;
  limit: number;
  offset: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  exporting: boolean;
  exportError: Error | undefined;
  onPageChange: (nextOffset: number) => void;
  onExportCsv: () => void;
};

export function CustomerDetailLedgerTab({
  customerId,
  items,
  total,
  limit,
  offset,
  fetching,
  error,
  hasSnapshot,
  exporting,
  exportError,
  onPageChange,
  onExportCsv,
}: CustomerDetailLedgerTabProps) {
  const billingLedgerExportHref = customerId?.trim()
    ? buildExportHubHref({
        entry: 'billing-ledger',
        kind: 'billing',
        customerId: customerId.trim(),
        format: 'csv',
        returnTo: `/customers/${customerId.trim()}?tab=ledger`,
      })
    : buildExportHubHref({ entry: 'billing-ledger', kind: 'billing', format: 'csv' });

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <CustomerTabShell
      blockingErrorTitle="Could not load ledger"
      fetchState={{ fetching, error, hasSnapshot }}
    >
      <section className={customerDetailSectionClass}>
        <form className={COMPACT_TOOLBAR_ROW_CLASS} onSubmit={(event) => event.preventDefault()}>
          <Button
            className={cn('shrink-0', adminTypography.body)}
            disabled={exporting}
            type="button"
            variant="outline"
            onClick={onExportCsv}
          >
            {exporting ? 'Exporting...' : 'Export CSV'}
          </Button>
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            variant="outline"
            onNext={() => onPageChange(offset + limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
        </form>

        {exportError ? <ErrorBlock title="Export failed" message={exportError.message} /> : null}

        <p className={adminTypography.bodyMuted}>
          Export CSV downloads a balance snapshot for this customer. For a date-range ledger export
          job, use{' '}
          <Link className="text-primary underline" to={billingLedgerExportHref}>
            Export Hub (Billing ledger)
          </Link>
          .
        </p>

        {(items ?? []).length === 0 ? (
          <EmptyState
            title="No ledger entries"
            description="This customer has no ledger rows yet."
          />
        ) : (
          <Card>
            <CardContent className="overflow-x-auto">
              <CustomerDetailLedgerEntryTable items={items ?? []} />
            </CardContent>
          </Card>
        )}
      </section>
    </CustomerTabShell>
  );
}
