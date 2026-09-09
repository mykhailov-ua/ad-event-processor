import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import type { BalanceLedgerEntry } from '@/api/types';
import { CustomerDetailLedgerEntryTable } from '@/domains/customers/customer_detail_ledger_entry_table';
import { CustomerTabShell } from '@/shell/customer_tab_shell';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { COMPACT_TOOLBAR_ROW_CLASS } from '@/shell/filter_panel';
import { EmptyState } from '@/shell/empty_state';
import { AdminMutationError } from '@/shell/admin_error';

export type CustomerDetailLedgerTabProps = {
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
  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <CustomerTabShell
      blockingErrorTitle="Could not load ledger"
      fetchState={{ fetching, error, hasSnapshot }}
    >
      <section >
        <form  onSubmit={(event) => event.preventDefault()}>
          <Button
           
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

        {exportError ? <AdminMutationError error={exportError} title="Export failed" /> : null}

        {(items ?? []).length === 0 ? (
          <EmptyState title="No ledger entries" description="This customer has no ledger rows yet." />
        ) : (
          <Card>
            <CardContent >
              <CustomerDetailLedgerEntryTable items={items ?? []} />
            </CardContent>
          </Card>
        )}
      </section>
    </CustomerTabShell>
  );
}
