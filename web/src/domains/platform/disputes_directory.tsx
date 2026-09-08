import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import type { DisputeRow } from '@/api/types';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type DisputesDirectoryProps = {
  disputes: DisputeRow[];
  total: number;
  limit: number;
  offset: number;
  appliedCustomerId: string;
  draftCustomerId: string;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onPageChange: (nextOffset: number) => void;
};

export function DisputesDirectory({
  disputes,
  total,
  limit,
  offset,
  appliedCustomerId,
  draftCustomerId,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  onPageChange,
}: DisputesDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Payment disputes">
        <ErrorBlock title="Could not load disputes" message={error.message} />
      </PageChrome>
    );
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <PageChrome title="Payment disputes">
      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      {disputes.length === 0 ? (
        <EmptyState title="No disputes" description="No payment disputes for the selected scope." />
      ) : (
        <DirectoryTable className={directoryTableRevalidatingClass(listRevalidating)}>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Intent</DirectoryTableHead>
              <DirectoryTableHead>Provider dispute</DirectoryTableHead>
              <DirectoryTableHead>Amount (micro)</DirectoryTableHead>
              <DirectoryTableHead>Currency</DirectoryTableHead>
              <DirectoryTableHead>Updated</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {disputes.map((row) => (
              <TableRow key={row.provider_dispute_id ?? row.intent_id ?? row.updated_at}>
                <TableCell className="text-xs">{row.intent_id ?? ''}</TableCell>
                <TableCell className="text-xs">{row.provider_dispute_id ?? ''}</TableCell>
                <TableCell>{displayMicro(row.amount_micro)}</TableCell>
                <TableCell>{row.currency ?? ''}</TableCell>
                <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      )}

      {total > 0 ? (
        <DirectoryPaginationFooter
          canGoNext={canGoNext}
          canGoPrev={canGoPrev}
          disabled={fetching || listRevalidating}
          onNext={() => onPageChange(offset + limit)}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
          rangeLabel={`${offset + 1}-${Math.min(offset + disputes.length, total)} of ${total}`}
        />
      ) : null}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
