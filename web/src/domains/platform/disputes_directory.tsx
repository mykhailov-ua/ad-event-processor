import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { DisputeRow } from '@/api/types';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type DisputesDirectoryProps = {
  disputes: DisputeRow[];
  appliedCustomerId: string;
  draftCustomerId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
};

export function DisputesDirectory({
  disputes,
  appliedCustomerId,
  draftCustomerId,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
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
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Intent</TableHead>
                <TableHead>Provider dispute</TableHead>
                <TableHead>Amount (micro)</TableHead>
                <TableHead>Currency</TableHead>
                <TableHead>Updated</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {disputes.map((row) => (
                <TableRow key={row.provider_dispute_id ?? row.intent_id ?? row.updated_at}>
                  <TableCell className="font-mono text-xs">{row.intent_id ?? ''}</TableCell>
                  <TableCell className="font-mono text-xs">{row.provider_dispute_id ?? ''}</TableCell>
                  <TableCell>{displayMicro(row.amount_micro)}</TableCell>
                  <TableCell>{row.currency ?? ''}</TableCell>
                  <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
