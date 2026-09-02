import { Link } from 'react-router-dom';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { PaginationPrevNext } from '@/shell/pagination_prev_next';
import type {
  BalanceLedgerEntry,
  BillingForecast,
  BillingStatement,
  Customer,
  CustomerBalance,
  PaymentHistoryRow,
  TaxProfile,
  Wallet,
} from '@/api/types';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type CustomerDetailTab =
  | 'profile'
  | 'balance'
  | 'ledger'
  | 'statement'
  | 'forecast'
  | 'wallet'
  | 'payments'
  | 'tax';

export const CUSTOMER_DETAIL_TABS: { id: CustomerDetailTab; label: string }[] = [
  { id: 'profile', label: 'Profile' },
  { id: 'balance', label: 'Balance' },
  { id: 'ledger', label: 'Ledger' },
  { id: 'statement', label: 'Statement' },
  { id: 'forecast', label: 'Forecast' },
  { id: 'wallet', label: 'Wallet' },
  { id: 'payments', label: 'Payments' },
  { id: 'tax', label: 'Tax profile' },
];

export type CustomerDetailProps = {
  customer: Customer | undefined;
  customerFetching: boolean;
  customerError: Error | undefined;
  hasCustomerSnapshot: boolean;
  tab: CustomerDetailTab;
  onTabChange: (tab: CustomerDetailTab) => void;
  balance: CustomerBalance | undefined;
  balanceFetching: boolean;
  balanceError: Error | undefined;
  hasBalanceSnapshot: boolean;
  ledgerItems: BalanceLedgerEntry[];
  ledgerTotal: number;
  ledgerLimit: number;
  ledgerOffset: number;
  ledgerFetching: boolean;
  ledgerError: Error | undefined;
  hasLedgerSnapshot: boolean;
  ledgerExporting: boolean;
  ledgerExportError: Error | undefined;
  onLedgerPageChange: (nextOffset: number) => void;
  onLedgerExportCsv: () => void;
  statementMonth: string;
  onStatementMonthChange: (month: string) => void;
  onStatementLoad: () => void;
  statement: BillingStatement | undefined;
  statementFetching: boolean;
  statementError: Error | undefined;
  hasStatementSnapshot: boolean;
  forecast: BillingForecast | undefined;
  forecastFetching: boolean;
  forecastError: Error | undefined;
  hasForecastSnapshot: boolean;
  wallet: Wallet | undefined;
  walletFetching: boolean;
  walletError: Error | undefined;
  hasWalletSnapshot: boolean;
  paymentItems: PaymentHistoryRow[];
  paymentTotal: number;
  paymentLimit: number;
  paymentOffset: number;
  paymentsFetching: boolean;
  paymentsError: Error | undefined;
  hasPaymentsSnapshot: boolean;
  onPaymentsPageChange: (nextOffset: number) => void;
  taxProfile: TaxProfile | undefined;
  taxFetching: boolean;
  taxError: Error | undefined;
  hasTaxSnapshot: boolean;
  draftCountryCode: string;
  draftTaxRegion: string;
  draftTaxScheme: string;
  draftTaxRateBps: string;
  onDraftCountryCodeChange: (value: string) => void;
  onDraftTaxRegionChange: (value: string) => void;
  onDraftTaxSchemeChange: (value: string) => void;
  onDraftTaxRateBpsChange: (value: string) => void;
  savingTax: boolean;
  saveError: Error | undefined;
  saveSuccess: boolean;
  canSaveTax: boolean;
  onSaveTaxProfile: () => void;
  draftCostCenter: string;
  onDraftCostCenterChange: (value: string) => void;
  savingCostCenter: boolean;
  costCenterSaveError: Error | undefined;
  costCenterSaveSuccess: boolean;
  canSaveCostCenter: boolean;
  onSaveCostCenter: () => void;
};

function DetailRow({ label, value }: { label: string; value: string | number | undefined }) {
  const display = value == null || value === '' ? '-' : String(value);
  return (
    <div className="grid gap-1 border-b py-3 last:border-b-0 sm:grid-cols-[10rem_1fr]">
      <dt className="text-sm font-medium text-muted-foreground">{label}</dt>
      <dd className="text-sm">{display}</dd>
    </div>
  );
}

function TabBar({
  tab,
  onTabChange,
}: {
  tab: CustomerDetailTab;
  onTabChange: (tab: CustomerDetailTab) => void;
}) {
  return (
    <div className="flex flex-wrap gap-2 border-b pb-2">
      {CUSTOMER_DETAIL_TABS.map((item) => (
        <Button
          key={item.id}
          type="button"
          variant={tab === item.id ? 'secondary' : 'ghost'}
         
          aria-pressed={tab === item.id}
          onClick={() => onTabChange(item.id)}
        >
          {item.label}
        </Button>
      ))}
    </div>
  );
}

function ProfileTab({
  customer,
  draftCostCenter,
  onDraftCostCenterChange,
  savingCostCenter,
  costCenterSaveError,
  costCenterSaveSuccess,
  canSaveCostCenter,
  onSaveCostCenter,
}: {
  customer: Customer;
  draftCostCenter: string;
  onDraftCostCenterChange: (value: string) => void;
  savingCostCenter: boolean;
  costCenterSaveError: Error | undefined;
  costCenterSaveSuccess: boolean;
  canSaveCostCenter: boolean;
  onSaveCostCenter: () => void;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Customer profile</CardTitle>
      </CardHeader>
      <CardContent>
        <dl>
          <DetailRow label="ID" value={customer.id} />
          <DetailRow label="Name" value={customer.name} />
          <DetailRow label="Balance" value={customer.balance} />
          <DetailRow label="Currency" value={customer.currency} />
          <DetailRow label="Active campaigns" value={customer.active_campaigns} />
          <DetailRow label="Total spend" value={customer.total_spend} />
          <DetailRow
            label="Created"
            value={displayTimestamp(customer.created_at, customer.created_at_display)}
          />
          <DetailRow
            label="Updated"
            value={displayTimestamp(customer.updated_at, customer.updated_at_display)}
          />
        </dl>

        <div className="mt-4 border-t pt-4">
          <form
            className="grid max-w-md gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              onSaveCostCenter();
            }}
          >
            <div className="grid gap-2">
              <Label htmlFor="customer-cost-center">Cost center</Label>
              {canSaveCostCenter ? (
                <Input
                  id="customer-cost-center"
                  value={draftCostCenter}
                  onChange={(event) => onDraftCostCenterChange(event.target.value)}
                />
              ) : (
                <p className="text-sm">{customer.cost_center ?? '-'}</p>
              )}
            </div>
            {canSaveCostCenter ? (
              <PrimaryActionButton disabled={!canSaveCostCenter} loading={savingCostCenter} type="submit">
                Save
              </PrimaryActionButton>
            ) : null}
          </form>
          {costCenterSaveError ? (
            <div className="mt-4">
              <ErrorBlock title="Save failed" message={costCenterSaveError.message} />
            </div>
          ) : null}
          {costCenterSaveSuccess ? (
            <p className="mt-4 text-sm text-muted-foreground" role="status">
              Cost center saved.
            </p>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}

function StatementTab({
  statementMonth,
  onStatementMonthChange,
  onStatementLoad,
  statement,
  fetching,
  error,
  hasSnapshot,
}: {
  statementMonth: string;
  onStatementMonthChange: (month: string) => void;
  onStatementLoad: () => void;
  statement: BillingStatement | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
}) {
  const lines = statement?.lines ?? [];

  return (
    <section className="grid gap-4">
      <div className="grid gap-4 sm:grid-cols-[minmax(0,12rem)_auto] sm:items-end">
        <div className="grid gap-2">
          <Label htmlFor="statement-month">Billing month</Label>
          <Input
            id="statement-month"
            type="month"
            value={statementMonth}
            onChange={(event) => onStatementMonthChange(event.target.value)}
          />
        </div>
        <Button type="button" onClick={onStatementLoad} disabled={fetching || !statementMonth}>
          Load
        </Button>
      </div>

      {fetching && !hasSnapshot && !error ? <PageSkeleton /> : null}

      {error && !hasSnapshot ? (
        <ErrorBlock title="Could not load statement" message={error.message} />
      ) : null}

      {hasSnapshot && statement ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Statement summary</CardTitle>
            </CardHeader>
            <CardContent>
              <dl>
                <DetailRow label="Currency" value={statement.currency} />
                <DetailRow
                  label="Period from"
                  value={statement.period?.from}
                />
                <DetailRow
                  label="Period to"
                  value={statement.period?.to}
                />
                <DetailRow
                  label="Opening balance (micro)"
                  value={displayMicro(statement.opening_balance_micro)}
                />
                <DetailRow
                  label="Closing balance (micro)"
                  value={displayMicro(statement.closing_balance_micro)}
                />
                <DetailRow
                  label="Tax (micro)"
                  value={displayMicro(statement.tax_breakdown?.tax_micro)}
                />
                <DetailRow
                  label="Invoice total (micro)"
                  value={displayMicro(statement.reconciliation?.invoice_total_micro)}
                />
                <DetailRow
                  label="Ledger sum (micro)"
                  value={displayMicro(statement.reconciliation?.ledger_sum_micro)}
                />
                <DetailRow
                  label="Reconciliation delta (micro)"
                  value={displayMicro(statement.reconciliation?.delta_micro)}
                />
              </dl>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Statement lines</CardTitle>
            </CardHeader>
            <CardContent className="overflow-x-auto">
              {lines.length === 0 ? (
                <p className="text-sm text-muted-foreground">No statement lines for this month.</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Ledger type</TableHead>
                      <TableHead className="text-right">Amount (micro)</TableHead>
                      <TableHead className="text-right">Entry count</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {lines.map((line, index) => (
                      <TableRow key={`${line.ledger_type ?? 'line'}-${index}`}>
                        <TableCell>{line.ledger_type ?? ''}</TableCell>
                        <TableCell className="text-right tabular-nums">
                          {displayMicro(line.amount_micro)}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {line.entry_count ?? ''}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </>
      ) : null}

      {!hasSnapshot && !fetching && !error ? (
        <p className="text-sm text-muted-foreground">Choose a month and click Load.</p>
      ) : null}
    </section>
  );
}

function ForecastTab({
  forecast,
  fetching,
  error,
  hasSnapshot,
}: {
  forecast: BillingForecast | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
}) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load forecast" message={error.message} />;
  }

  if (!forecast) {
    return null;
  }

  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 space-y-0">
        <CardTitle className="text-base">Billing forecast</CardTitle>
        <div className="flex flex-wrap gap-2">
          {forecast.low_confidence ? (
            <Badge variant="secondary">Low confidence</Badge>
          ) : null}
          {forecast.ch_unavailable ? (
            <Badge variant="outline">ClickHouse unavailable</Badge>
          ) : null}
        </div>
      </CardHeader>
      <CardContent>
        <dl>
          <DetailRow label="Month" value={forecast.month} />
          <DetailRow
            label="Ledger MTD (micro)"
            value={displayMicro(forecast.ledger_mtd_micro)}
          />
          <DetailRow
            label="Run rate (micro/day)"
            value={displayMicro(forecast.ledger_run_rate_micro_per_day)}
          />
          <DetailRow
            label="Projected month end (micro)"
            value={displayMicro(forecast.projected_month_end_micro)}
          />
          <DetailRow label="Days remaining" value={forecast.days_remaining} />
        </dl>
      </CardContent>
    </Card>
  );
}

function PaymentsTab({
  items,
  total,
  limit,
  offset,
  fetching,
  error,
  hasSnapshot,
  onPageChange,
}: {
  items: PaymentHistoryRow[];
  total: number;
  limit: number;
  offset: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onPageChange: (nextOffset: number) => void;
}) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load payments" message={error.message} />;
  }

  if (!hasSnapshot) {
    return null;
  }

  return (
    <section className="grid gap-6">
      {items.length === 0 ? (
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
          <form
            className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
            onSubmit={(event) => event.preventDefault()}
          >
            <PaginationPrevNext
              canGoPrev={offset > 0}
              canGoNext={offset + items.length < total}
              disabled={fetching}
              onPrev={() => onPageChange(Math.max(0, offset - limit))}
              onNext={() => onPageChange(offset + limit)}
            />
          </form>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Payments</CardTitle>
            </CardHeader>
            <CardContent className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Intent ID</TableHead>
                    <TableHead className="text-right">Amount (micro)</TableHead>
                    <TableHead>Currency</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Provider</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((row) => (
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
              </Table>
            </CardContent>
          </Card>
        </>
      )}

      {error && hasSnapshot ? (
        <ErrorBlock title="Refresh failed" message={error.message} />
      ) : null}
    </section>
  );
}

function BalanceTab({
  balance,
  fetching,
  error,
  hasSnapshot,
}: {
  balance: CustomerBalance | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
}) {
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
          <dl>
            <DetailRow label="Balance" value={balance.balance} />
            <DetailRow label="Currency" value={balance.currency} />
          </dl>
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
            <LedgerEntryTable items={recent} />
          )}
        </CardContent>
      </Card>
    </section>
  );
}

function LedgerTab({
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
}: {
  items: BalanceLedgerEntry[];
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
}) {
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
      <form
        className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
        onSubmit={(event) => event.preventDefault()}
      >
        <Button
          className="text-sm"
          disabled={exporting}
          onClick={onExportCsv}
          type="button"
          variant="outline"
        >
          {exporting ? 'Exporting...' : 'Export CSV'}
        </Button>
        <PaginationPrevNext
          canGoPrev={canGoPrev}
          canGoNext={canGoNext}
          disabled={fetching}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
          onNext={() => onPageChange(offset + limit)}
        />
      </form>

      {exportError ? <ErrorBlock title="Export failed" message={exportError.message} /> : null}

      {items.length === 0 ? (
        <EmptyState title="No ledger entries" description="This customer has no ledger rows yet." />
      ) : (
        <Card>
          <CardContent className="overflow-x-auto pt-6">
            <LedgerEntryTable items={items} />
          </CardContent>
        </Card>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </section>
  );
}

function LedgerEntryTable({ items }: { items: BalanceLedgerEntry[] }) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>ID</TableHead>
          <TableHead>Type</TableHead>
          <TableHead className="text-right">Amount</TableHead>
          <TableHead>Campaign</TableHead>
          <TableHead>Created</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((row) => (
          <TableRow key={row.id ?? `${row.created_at}-${row.type}`}>
            <TableCell className="tabular-nums">{row.id ?? ''}</TableCell>
            <TableCell>{row.type ?? ''}</TableCell>
            <TableCell className="text-right tabular-nums">{row.amount ?? ''}</TableCell>
            <TableCell className="font-mono text-xs">{row.campaign_id ?? ''}</TableCell>
            <TableCell>{displayTimestamp(row.created_at)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function WalletTab({
  wallet,
  fetching,
  error,
  hasSnapshot,
}: {
  wallet: Wallet | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
}) {
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
        <dl>
          <DetailRow label="Balance (micro)" value={displayMicro(wallet.balance_micro)} />
          <DetailRow label="Currency" value={wallet.currency} />
          <DetailRow
            label="Allowed overdraft (micro)"
            value={displayMicro(wallet.allowed_overdraft_micro)}
          />
          <DetailRow
            label="Low balance threshold (micro)"
            value={displayMicro(wallet.low_balance_threshold_micro)}
          />
          <DetailRow label="Burn days estimate" value={wallet.burn_days_estimate} />
          <DetailRow
            label="Last invoice"
            value={displayTimestamp(wallet.last_invoice_at)}
          />
          <DetailRow label="Payment provider" value={wallet.payment_provider} />
          <DetailRow
            label="Provider configured"
            value={wallet.payment_provider_configured ? 'yes' : 'no'}
          />
        </dl>
      </CardContent>
    </Card>
  );
}

function TaxProfileTab({
  taxProfile,
  fetching,
  error,
  hasSnapshot,
  draftCountryCode,
  draftTaxRegion,
  draftTaxScheme,
  draftTaxRateBps,
  onDraftCountryCodeChange,
  onDraftTaxRegionChange,
  onDraftTaxSchemeChange,
  onDraftTaxRateBpsChange,
  saving,
  saveError,
  saveSuccess,
  canSave,
  onSave,
}: {
  taxProfile: TaxProfile | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftCountryCode: string;
  draftTaxRegion: string;
  draftTaxScheme: string;
  draftTaxRateBps: string;
  onDraftCountryCodeChange: (value: string) => void;
  onDraftTaxRegionChange: (value: string) => void;
  onDraftTaxSchemeChange: (value: string) => void;
  onDraftTaxRateBpsChange: (value: string) => void;
  saving: boolean;
  saveError: Error | undefined;
  saveSuccess: boolean;
  canSave: boolean;
  onSave: () => void;
}) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load tax profile" message={error.message} />;
  }

  if (!hasSnapshot) {
    return null;
  }

  return (
    <section className="grid gap-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Tax profile</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            className="grid max-w-md gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              onSave();
            }}
          >
            <div className="grid gap-2">
              <Label htmlFor="tax-country-code">Country code</Label>
              <Input
                id="tax-country-code"
                value={draftCountryCode}
                onChange={(event) => onDraftCountryCodeChange(event.target.value)}
                disabled={!canSave}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="tax-region">Tax region</Label>
              <Input
                id="tax-region"
                value={draftTaxRegion}
                onChange={(event) => onDraftTaxRegionChange(event.target.value)}
                disabled={!canSave}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="tax-scheme">Tax scheme</Label>
              <Input
                id="tax-scheme"
                value={draftTaxScheme}
                onChange={(event) => onDraftTaxSchemeChange(event.target.value)}
                disabled={!canSave}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="tax-rate-bps">Tax rate (bps)</Label>
              <Input
                id="tax-rate-bps"
                type="number"
                inputMode="numeric"
                value={draftTaxRateBps}
                onChange={(event) => onDraftTaxRateBpsChange(event.target.value)}
                disabled={!canSave}
              />
            </div>
            <PrimaryActionButton disabled={!canSave} loading={saving} type="submit">
              Save
            </PrimaryActionButton>
          </form>
          {taxProfile?.customer_id ? (
            <p className="mt-4 text-xs text-muted-foreground">
              Customer ID: {taxProfile.customer_id}
            </p>
          ) : null}
        </CardContent>
      </Card>

      {saveError ? <ErrorBlock title="Save failed" message={saveError.message} /> : null}
      {saveSuccess ? (
        <p className="text-sm text-muted-foreground" role="status">
          Tax profile saved.
        </p>
      ) : null}
      {error && hasSnapshot ? (
        <ErrorBlock title="Refresh failed" message={error.message} />
      ) : null}
    </section>
  );
}

export function CustomerDetail({
  customer,
  customerFetching,
  customerError,
  hasCustomerSnapshot,
  tab,
  onTabChange,
  balance,
  balanceFetching,
  balanceError,
  hasBalanceSnapshot,
  ledgerItems,
  ledgerTotal,
  ledgerLimit,
  ledgerOffset,
  ledgerFetching,
  ledgerError,
  hasLedgerSnapshot,
  ledgerExporting,
  ledgerExportError,
  onLedgerPageChange,
  onLedgerExportCsv,
  statementMonth,
  onStatementMonthChange,
  onStatementLoad,
  statement,
  statementFetching,
  statementError,
  hasStatementSnapshot,
  forecast,
  forecastFetching,
  forecastError,
  hasForecastSnapshot,
  wallet,
  walletFetching,
  walletError,
  hasWalletSnapshot,
  paymentItems,
  paymentTotal,
  paymentLimit,
  paymentOffset,
  paymentsFetching,
  paymentsError,
  hasPaymentsSnapshot,
  onPaymentsPageChange,
  taxProfile,
  taxFetching,
  taxError,
  hasTaxSnapshot,
  draftCountryCode,
  draftTaxRegion,
  draftTaxScheme,
  draftTaxRateBps,
  onDraftCountryCodeChange,
  onDraftTaxRegionChange,
  onDraftTaxSchemeChange,
  onDraftTaxRateBpsChange,
  savingTax,
  saveError,
  saveSuccess,
  canSaveTax,
  onSaveTaxProfile,
  draftCostCenter,
  onDraftCostCenterChange,
  savingCostCenter,
  costCenterSaveError,
  costCenterSaveSuccess,
  canSaveCostCenter,
  onSaveCostCenter,
}: CustomerDetailProps) {
  if (customerFetching && !hasCustomerSnapshot && !customerError) {
    return <PageSkeleton />;
  }

  if (customerError && !hasCustomerSnapshot) {
    return <ErrorBlock title="Could not load customer" message={customerError.message} />;
  }

  if (!customer) {
    return <ErrorBlock title="Customer not found" message="No customer data returned." />;
  }

  return (
    <section className="grid gap-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm text-muted-foreground">
            <Link className="text-primary hover:underline" to="/customers">
              Customers
            </Link>
          </p>
          <h1 className="text-lg font-semibold">{customer.name ?? customer.id}</h1>
          {customerFetching ? <p className="text-sm text-muted-foreground">Refreshing...</p> : null}
        </div>
      </header>

      <TabBar tab={tab} onTabChange={onTabChange} />

      {tab === 'profile' ? (
        <ProfileTab
          customer={customer}
          draftCostCenter={draftCostCenter}
          onDraftCostCenterChange={onDraftCostCenterChange}
          savingCostCenter={savingCostCenter}
          costCenterSaveError={costCenterSaveError}
          costCenterSaveSuccess={costCenterSaveSuccess}
          canSaveCostCenter={canSaveCostCenter}
          onSaveCostCenter={onSaveCostCenter}
        />
      ) : null}
      {tab === 'balance' ? (
        <BalanceTab
          balance={balance}
          fetching={balanceFetching}
          error={balanceError}
          hasSnapshot={hasBalanceSnapshot}
        />
      ) : null}
      {tab === 'ledger' ? (
        <LedgerTab
          items={ledgerItems}
          total={ledgerTotal}
          limit={ledgerLimit}
          offset={ledgerOffset}
          fetching={ledgerFetching}
          error={ledgerError}
          hasSnapshot={hasLedgerSnapshot}
          exporting={ledgerExporting}
          exportError={ledgerExportError}
          onPageChange={onLedgerPageChange}
          onExportCsv={onLedgerExportCsv}
        />
      ) : null}
      {tab === 'statement' ? (
        <StatementTab
          statementMonth={statementMonth}
          onStatementMonthChange={onStatementMonthChange}
          onStatementLoad={onStatementLoad}
          statement={statement}
          fetching={statementFetching}
          error={statementError}
          hasSnapshot={hasStatementSnapshot}
        />
      ) : null}
      {tab === 'forecast' ? (
        <ForecastTab
          forecast={forecast}
          fetching={forecastFetching}
          error={forecastError}
          hasSnapshot={hasForecastSnapshot}
        />
      ) : null}
      {tab === 'wallet' ? (
        <WalletTab
          wallet={wallet}
          fetching={walletFetching}
          error={walletError}
          hasSnapshot={hasWalletSnapshot}
        />
      ) : null}
      {tab === 'payments' ? (
        <PaymentsTab
          items={paymentItems}
          total={paymentTotal}
          limit={paymentLimit}
          offset={paymentOffset}
          fetching={paymentsFetching}
          error={paymentsError}
          hasSnapshot={hasPaymentsSnapshot}
          onPageChange={onPaymentsPageChange}
        />
      ) : null}
      {tab === 'tax' ? (
        <TaxProfileTab
          taxProfile={taxProfile}
          fetching={taxFetching}
          error={taxError}
          hasSnapshot={hasTaxSnapshot}
          draftCountryCode={draftCountryCode}
          draftTaxRegion={draftTaxRegion}
          draftTaxScheme={draftTaxScheme}
          draftTaxRateBps={draftTaxRateBps}
          onDraftCountryCodeChange={onDraftCountryCodeChange}
          onDraftTaxRegionChange={onDraftTaxRegionChange}
          onDraftTaxSchemeChange={onDraftTaxSchemeChange}
          onDraftTaxRateBpsChange={onDraftTaxRateBpsChange}
          saving={savingTax}
          saveError={saveError}
          saveSuccess={saveSuccess}
          canSave={canSaveTax}
          onSave={onSaveTaxProfile}
        />
      ) : null}
    </section>
  );
}
