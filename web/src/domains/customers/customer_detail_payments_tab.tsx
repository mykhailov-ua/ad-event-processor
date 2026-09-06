import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { PaymentHistoryRow } from '@/api/types';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { COMPACT_TOOLBAR_ROW_CLASS } from '@/shell/filter_panel';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { StubBanner } from '@/shell/stub_banner';
import { displayMicro, displayTimestamp } from '@/lib/display';
import { isPaymentUnavailableError, userErrorMessage } from '@/lib/admin_error';

export type CustomerDetailPaymentsTabProps = {
  items?: PaymentHistoryRow[];
  total: number;
  limit: number;
  offset: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onPageChange: (nextOffset: number) => void;
};

export function CustomerDetailPaymentsTab({
  items,
  total,
  limit,
  offset,
  fetching,
  error,
  hasSnapshot,
  onPageChange,
}: CustomerDetailPaymentsTabProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    if (isPaymentUnavailableError(error)) {
      return (
        <StubBanner
          title="Payments unavailable"
          message={userErrorMessage(error, 'Payment history is not available in this deployment.')}
        />
      );
    }
    return <ErrorBlock error={error} title="Could not load payments" />;
  }

  if (!hasSnapshot) {
    return null;
  }

  return (
    <section className="grid gap-6">
      {(items ?? []).length === 0 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Payments</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">No payments for this customer.</p>
          </CardContent>
        </Card>
      ) : (
        <>
          <form className={COMPACT_TOOLBAR_ROW_CLASS} onSubmit={(event) => event.preventDefault()}>
            <DirectoryPaginationFooter
              canGoNext={offset + (items ?? []).length < total}
              canGoPrev={offset > 0}
              disabled={fetching}
              variant="outline"
              onNext={() => onPageChange(offset + limit)}
              onPrev={() => onPageChange(Math.max(0, offset - limit))}
            />
          </form>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Payments</CardTitle>
            </CardHeader>
            <CardContent className="overflow-x-auto">
              <DirectoryTable nested>
                <TableHeader>
                  <TableRow>
                    <DirectoryTableHead>Intent ID</DirectoryTableHead>
                    <DirectoryTableHead align="end">Amount (micro)</DirectoryTableHead>
                    <DirectoryTableHead>Currency</DirectoryTableHead>
                    <DirectoryTableHead>Status</DirectoryTableHead>
                    <DirectoryTableHead>Provider</DirectoryTableHead>
                    <DirectoryTableHead>Created</DirectoryTableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(items ?? []).map((row) => (
                    <TableRow key={`${row.intent_id ?? 'payment'}-${row.created_at ?? ''}`}>
                      <TableCell className="font-mono text-xs">{row.intent_id ?? ''}</TableCell>
                      <TableCell className="text-right tabular-nums">
                        {displayMicro(row.amount_micro)}
                      </TableCell>
                      <TableCell>{row.currency ?? ''}</TableCell>
                      <TableCell>{row.status ?? ''}</TableCell>
                      <TableCell>{row.provider ?? ''}</TableCell>
                      <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </DirectoryTable>
            </CardContent>
          </Card>
        </>
      )}

      {error && hasSnapshot ? <ErrorBlock error={error} title="Refresh failed" /> : null}
    </section>
  );
}
