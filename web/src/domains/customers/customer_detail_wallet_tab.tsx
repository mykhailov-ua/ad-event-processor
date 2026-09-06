import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { Wallet } from '@/api/types';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
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
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load wallet" message={error.message} />;
  }

  if (!wallet) {
    return null;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Wallet</CardTitle>
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
  );
}
