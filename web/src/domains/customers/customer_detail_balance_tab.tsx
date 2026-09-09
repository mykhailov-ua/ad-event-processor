import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { adminSpacing, adminTypography, customerDetailSectionClass } from '@/lib/admin_spacing';
import type { CustomerBalance } from '@/api/types';
import { CustomerDetailLedgerEntryTable } from '@/domains/customers/customer_detail_ledger_entry_table';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { CustomerTabShell } from '@/shell/customer_tab_shell';
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
  const recent = balance?.ledger ?? [];

  return (
    <CustomerTabShell
      blockingErrorTitle="Could not load balance"
      fetchState={{ fetching, error, hasSnapshot }}
    >
      {balance ? (
        <section className={customerDetailSectionClass}>
          <Card>
            <CardHeader>
              <CardTitle>Ledger balance</CardTitle>
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
              <CardTitle>Recent ledger entries</CardTitle>
            </CardHeader>
            <CardContent className="overflow-x-auto">
              {recent.length === 0 ? (
                <p className={adminTypography.bodyMuted}>No recent ledger entries.</p>
              ) : (
                <CustomerDetailLedgerEntryTable items={recent} />
              )}
            </CardContent>
          </Card>
        </section>
      ) : null}
    </CustomerTabShell>
  );
}
