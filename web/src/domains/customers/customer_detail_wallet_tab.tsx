import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { Wallet } from '@/api/types';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { CustomerTabShell } from '@/shell/customer_tab_shell';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type CustomerDetailWalletTabProps = {
  wallet: Wallet | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function CustomerDetailWalletTab({
  wallet,
  fetching,
  error,
  hasSnapshot,
}: CustomerDetailWalletTabProps) {
  return (
    <CustomerTabShell
      blockingErrorTitle="Could not load wallet"
      fetchState={{ fetching, error, hasSnapshot }}
    >
      {wallet ? (
        <Card>
          <CardHeader>
            <CardTitle>Wallet</CardTitle>
          </CardHeader>
          <CardContent>
            <CustomerDetailPanel>
              <CustomerDetailRow label="Balance (micro)" value={displayMicro(wallet.balance_micro)} />
              <CustomerDetailRow label="Currency" value={wallet.currency} />
              <CustomerDetailRow
                label="Allowed overdraft (micro)"
                value={displayMicro(wallet.allowed_overdraft_micro)}
              />
              <CustomerDetailRow
                label="Low balance threshold (micro)"
                value={displayMicro(wallet.low_balance_threshold_micro)}
              />
              <CustomerDetailRow label="Burn days estimate" value={wallet.burn_days_estimate} />
              <CustomerDetailRow
                label="Last invoice"
                value={displayTimestamp(wallet.last_invoice_at)}
              />
              <CustomerDetailRow label="Payment provider" value={wallet.payment_provider} />
              <CustomerDetailRow
                label="Provider configured"
                value={wallet.payment_provider_configured ? 'yes' : 'no'}
              />
            </CustomerDetailPanel>
          </CardContent>
        </Card>
      ) : null}
    </CustomerTabShell>
  );
}
