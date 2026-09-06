import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import type { BalanceLedgerEntry } from '@/api/types';
import { CustomerDetailLedgerEntryTable } from '@/domains/customers/customer_detail_ledger_entry_table';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { COMPACT_TOOLBAR_ROW_CLASS } from '@/shell/filter_panel';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';

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
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load ledger" message={error.message} />;
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <section className="grid gap-6">
      <form className={COMPACT_TOOLBAR_ROW_CLASS} onSubmit={(event) => event.preventDefault()}>
        <Button
          className="shrink-0 text-sm"
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

      {(items ?? []).length === 0 ? (
        <EmptyState title="No ledger entries" description="This customer has no ledger rows yet." />
      ) : (
        <Card>
          <CardContent className="overflow-x-auto">
            <CustomerDetailLedgerEntryTable items={items ?? []} />
          </CardContent>
        </Card>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </section>
  );
}
