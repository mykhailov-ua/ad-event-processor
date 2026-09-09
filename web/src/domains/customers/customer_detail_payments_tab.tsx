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
import { CustomerTabShell } from '@/shell/customer_tab_shell';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { COMPACT_TOOLBAR_ROW_CLASS } from '@/shell/filter_panel';
import { displayMicro, displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';

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
  return (
    <CustomerTabShell
      blockingErrorOptions={{ unavailableTitle: 'Payments unavailable' }}
      blockingErrorTitle="Could not load payments"
      fetchState={{ fetching, error, hasSnapshot }}
    >
      {hasSnapshot ? (
        <section>
          {(items ?? []).length === 0 ? (
            <Card>
              <CardHeader>
                <CardTitle>Payments</CardTitle>
              </CardHeader>
              <CardContent>
                <p>No payments for this customer.</p>
              </CardContent>
            </Card>
          ) : (
            <>
              <form
               
                onSubmit={(event) => event.preventDefault()}
              >
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
                  <CardTitle>Payments</CardTitle>
                </CardHeader>
                <CardContent>
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
                          <TableCell>{row.intent_id ?? ''}</TableCell>
                          <TableCell>
                            {displayMicro(row.amount_micro)}
                          </TableCell>
                          <TableCell className={adminTypography.monoData} >{row.currency ?? ''}</TableCell>
                          <TableCell className="text-right tabular-nums" >{row.status ?? ''}</TableCell>
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
        </section>
      ) : null}
    </CustomerTabShell>
  );
}
