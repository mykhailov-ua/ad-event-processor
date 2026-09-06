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

export function customerDetailTabs(paymentEnabled: boolean) {
  if (paymentEnabled) {
    return CUSTOMER_DETAIL_TABS;
  }
  return CUSTOMER_DETAIL_TABS.filter((item) => item.id !== 'payments');
}

export type CustomerDetailProps = {
  customer: Customer | undefined;
  customerFetching: boolean;
  customerError: Error | undefined;
  hasCustomerSnapshot: boolean;
  paymentEnabled: boolean;
  tab: CustomerDetailTab;
  onTabChange: (tab: CustomerDetailTab) => void;
  balance: CustomerBalance | undefined;
  balanceFetching: boolean;
  balanceError: Error | undefined;
  hasBalanceSnapshot: boolean;
  ledgerItems?: BalanceLedgerEntry[];
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
  paymentItems?: PaymentHistoryRow[];
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
