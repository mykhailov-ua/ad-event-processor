import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { CustomerBalance } from '@/api/types';
import { CustomerDetailLedgerEntryTable } from '@/domains/customers/customer_detail_ledger_entry_table';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';

export type CustomerDetailBalanceTabProps = {
  balance: CustomerBalance | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function CustomerDetailBalanceTab({
  balance,
  fetching,
  error,
  hasSnapshot,
}: CustomerDetailBalanceTabProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load balance" message={error.message} />;
  }

  if (!balance) {
    return null;
  }

  const recent = balance.ledger ?? [];

  return (
    <section className="grid gap-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Ledger balance</CardTitle>
        </CardHeader>
        <CardContent>
          <CustomerDetailPanel>
            <CustomerDetailRow label="Balance" value={balance.balance} />
            <CustomerDetailRow label="Currency" value={balance.currency} />
          </CustomerDetailPanel>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Recent ledger entries</CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          {recent.length === 0 ? (
            <p className="text-sm text-muted-foreground">No recent ledger entries.</p>
          ) : (
            <CustomerDetailLedgerEntryTable items={recent} />
          )}
        </CardContent>
      </Card>
    </section>
  );
}
