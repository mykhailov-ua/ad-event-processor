import { Link } from 'react-router-dom';

import { CustomerDetailBalanceTab } from '@/domains/customers/customer_detail_balance_tab';
import { CustomerDetailForecastTab } from '@/domains/customers/customer_detail_forecast_tab';
import { CustomerDetailLedgerTab } from '@/domains/customers/customer_detail_ledger_tab';
import { CustomerDetailPaymentsTab } from '@/domains/customers/customer_detail_payments_tab';
import { CustomerDetailProfileTab } from '@/domains/customers/customer_detail_profile_tab';
import { CustomerDetailStatementTab } from '@/domains/customers/customer_detail_statement_tab';
import { CustomerDetailTabBar } from '@/domains/customers/customer_detail_tab_bar';
import { CustomerDetailTaxTab } from '@/domains/customers/customer_detail_tax_tab';
import type { CustomerDetailProps } from '@/domains/customers/customer_detail_types';
import { CustomerDetailWalletTab } from '@/domains/customers/customer_detail_wallet_tab';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';

export type {
  CustomerDetailTab,
  CustomerDetailProps,
} from '@/domains/customers/customer_detail_types';

export function CustomerDetail({
  customer,
  customerFetching,
  customerError,
  hasCustomerSnapshot,
  paymentEnabled,
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
  draftName,
  onDraftNameChange,
  draftCostCenter,
  onDraftCostCenterChange,
  savingProfile,
  profileSaveError,
  profileSaveSuccess,
  canSaveProfile,
  onSaveProfile,
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
      <header className="grid gap-1">
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

      <CustomerDetailTabBar paymentEnabled={paymentEnabled} tab={tab} onTabChange={onTabChange} />

      {tab === 'profile' ? (
        <CustomerDetailProfileTab
          canSaveProfile={canSaveProfile}
          customer={customer}
          draftCostCenter={draftCostCenter}
          draftName={draftName}
          profileSaveError={profileSaveError}
          profileSaveSuccess={profileSaveSuccess}
          savingProfile={savingProfile}
          onDraftCostCenterChange={onDraftCostCenterChange}
          onDraftNameChange={onDraftNameChange}
          onSaveProfile={onSaveProfile}
        />
      ) : null}
      {tab === 'balance' ? (
        <CustomerDetailBalanceTab
          balance={balance}
          error={balanceError}
          fetching={balanceFetching}
          hasSnapshot={hasBalanceSnapshot}
        />
      ) : null}
      {tab === 'ledger' ? (
        <CustomerDetailLedgerTab
          error={ledgerError}
          exportError={ledgerExportError}
          exporting={ledgerExporting}
          fetching={ledgerFetching}
          hasSnapshot={hasLedgerSnapshot}
          items={ledgerItems}
          limit={ledgerLimit}
          offset={ledgerOffset}
          total={ledgerTotal}
          onExportCsv={onLedgerExportCsv}
          onPageChange={onLedgerPageChange}
        />
      ) : null}
      {tab === 'statement' ? (
        <CustomerDetailStatementTab
          error={statementError}
          fetching={statementFetching}
          hasSnapshot={hasStatementSnapshot}
          statement={statement}
          statementMonth={statementMonth}
          onStatementLoad={onStatementLoad}
          onStatementMonthChange={onStatementMonthChange}
        />
      ) : null}
      {tab === 'forecast' ? (
        <CustomerDetailForecastTab
          error={forecastError}
          fetching={forecastFetching}
          forecast={forecast}
          hasSnapshot={hasForecastSnapshot}
        />
      ) : null}
      {tab === 'wallet' ? (
        <CustomerDetailWalletTab
          error={walletError}
          fetching={walletFetching}
          hasSnapshot={hasWalletSnapshot}
          wallet={wallet}
        />
      ) : null}
      {tab === 'payments' ? (
        <CustomerDetailPaymentsTab
          error={paymentsError}
          fetching={paymentsFetching}
          hasSnapshot={hasPaymentsSnapshot}
          items={paymentItems}
          limit={paymentLimit}
          offset={paymentOffset}
          total={paymentTotal}
          onPageChange={onPaymentsPageChange}
        />
      ) : null}
      {tab === 'tax' ? (
        <CustomerDetailTaxTab
          canSave={canSaveTax}
          draftCountryCode={draftCountryCode}
          draftTaxRateBps={draftTaxRateBps}
          draftTaxRegion={draftTaxRegion}
          draftTaxScheme={draftTaxScheme}
          error={taxError}
          fetching={taxFetching}
          hasSnapshot={hasTaxSnapshot}
          saveError={saveError}
          saveSuccess={saveSuccess}
          saving={savingTax}
          taxProfile={taxProfile}
          onDraftCountryCodeChange={onDraftCountryCodeChange}
          onDraftTaxRateBpsChange={onDraftTaxRateBpsChange}
          onDraftTaxRegionChange={onDraftTaxRegionChange}
          onDraftTaxSchemeChange={onDraftTaxSchemeChange}
          onSave={onSaveTaxProfile}
        />
      ) : null}
    </section>
  );
}
