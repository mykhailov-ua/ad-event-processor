// owner: customer detail tabs, draft tax/cost-center forms, tab-gated useResource lanes.
// Each tab uses a separate useResource; inactive tabs call skipCustomerDetailTabFetch().
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';
import { useParams } from 'react-router-dom';

import {
  exportCustomerBalanceCsv,
  getCustomerBalance,
  getCustomerBillingForecast,
  getCustomerBillingStatement,
  getCustomerTaxProfile,
  getCustomerWallet,
  listCustomerLedger,
  listCustomerPayments,
  putCustomerTaxProfile,
} from '@/api/billing_api';
import { getCustomer, patchCustomer } from '@/api/customers_api';
import type { CustomerDetailTab } from '@/domains/customers/customer_detail_types';
import {
  CUSTOMER_DETAIL_LEDGER_PAGE_LIMIT,
  CUSTOMER_DETAIL_PAYMENTS_PAGE_LIMIT,
  currentCustomerDetailMonthValue,
  skipCustomerDetailTabFetch,
  taxProfileToDraft,
} from '@/domains/customers/customer_detail_page_utils';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';
import { useMeta } from '@/hooks/use_meta';
import { useSession } from '@/hooks/use_session';
import { triggerBlobDownload } from '@/lib/trigger_blob_download';

export function useCustomerDetailPageWorkspace() {
  const { id } = useParams<{ id: string }>();
  const [tab, setTab] = useState<CustomerDetailTab>('profile');
  const [statementMonth, setStatementMonth] = useState(currentCustomerDetailMonthValue);
  const [statementMonthApplied, setStatementMonthApplied] = useState<string | null>(null);
  const [paymentsOffset, setPaymentsOffset] = useState(0);
  const [ledgerOffset, setLedgerOffset] = useState(0);
  const [ledgerExporting, setLedgerExporting] = useState(false);
  const [ledgerExportError, setLedgerExportError] = useState<Error | undefined>();
  const [taxRefreshToken, setTaxRefreshToken] = useState(0);
  const [draftCountryCode, setDraftCountryCode] = useState('');
  const [draftTaxRegion, setDraftTaxRegion] = useState('');
  const [draftTaxScheme, setDraftTaxScheme] = useState('');
  const [draftTaxRateBps, setDraftTaxRateBps] = useState('');
  const [savingTax, setSavingTax] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [draftCostCenter, setDraftCostCenter] = useState('');
  const [draftName, setDraftName] = useState('');
  const [savingProfile, setSavingProfile] = useState(false);
  const [profileSaveError, setProfileSaveError] = useState<Error | undefined>();
  const [profileSaveSuccess, setProfileSaveSuccess] = useState(false);
  const [customerRefreshToken, setCustomerRefreshToken] = useState(0);

  useEffect(() => {
    setPaymentsOffset(0);
    setLedgerOffset(0);
  }, [id]);

  const { user } = useSession();
  const { meta } = useMeta();
  const paymentEnabled = meta?.payment_enabled === true;
  const canSaveTax = user?.permissions?.includes('customers:write') ?? false;
  const canSaveProfile = canSaveTax;

  const customerResource = useResource(
    (signal) => {
      if (!id) {
        return Promise.reject(new Error('Customer id is required'));
      }
      return getCustomer(id, signal);
    },
    [id, customerRefreshToken]
  );

  useEffect(() => {
    if (!customerResource.data) {
      return;
    }
    setDraftCostCenter(customerResource.data.cost_center ?? '');
    setDraftName(customerResource.data.name ?? '');
  }, [customerResource.data]);

  const balanceResource = useResource(
    (signal) => {
      if (!id || tab !== 'balance') {
        return skipCustomerDetailTabFetch();
      }
      return getCustomerBalance(id, signal);
    },
    [id, tab]
  );

  const ledgerResource = useResource(
    (signal) => {
      if (!id || tab !== 'ledger') {
        return skipCustomerDetailTabFetch();
      }
      return listCustomerLedger(
        id,
        { limit: CUSTOMER_DETAIL_LEDGER_PAGE_LIMIT, offset: ledgerOffset },
        signal
      );
    },
    [id, tab, ledgerOffset]
  );

  const statementResource = useResource(
    (signal) => {
      if (!id || tab !== 'statement' || !statementMonthApplied) {
        return skipCustomerDetailTabFetch();
      }
      return getCustomerBillingStatement(id, statementMonthApplied, signal);
    },
    [id, tab, statementMonthApplied]
  );

  const forecastResource = useResource(
    (signal) => {
      if (!id || tab !== 'forecast') {
        return skipCustomerDetailTabFetch();
      }
      return getCustomerBillingForecast(id, signal);
    },
    [id, tab]
  );

  const walletResource = useResource(
    (signal) => {
      if (!id || tab !== 'wallet') {
        return skipCustomerDetailTabFetch();
      }
      return getCustomerWallet(id, signal);
    },
    [id, tab]
  );

  const paymentsResource = useResource(
    (signal) => {
      if (!id || tab !== 'payments') {
        return skipCustomerDetailTabFetch();
      }
      return listCustomerPayments(
        id,
        { limit: CUSTOMER_DETAIL_PAYMENTS_PAGE_LIMIT, offset: paymentsOffset },
        signal
      );
    },
    [id, tab, paymentsOffset]
  );

  const taxResource = useResource(
    (signal) => {
      if (!id || tab !== 'tax') {
        return skipCustomerDetailTabFetch();
      }
      return getCustomerTaxProfile(id, signal);
    },
    [id, tab, taxRefreshToken]
  );

  useEffect(() => {
    if (!taxResource.data) {
      return;
    }
    const draft = taxProfileToDraft(taxResource.data);
    setDraftCountryCode(draft.countryCode);
    setDraftTaxRegion(draft.taxRegion);
    setDraftTaxScheme(draft.taxScheme);
    setDraftTaxRateBps(draft.taxRateBps);
  }, [taxResource.data]);

  useEffect(() => {
    if (!paymentEnabled && tab === 'payments') {
      setTab('profile');
    }
  }, [paymentEnabled, tab]);

  const onPaymentsPageChange = useCallback((nextOffset: number) => {
    setPaymentsOffset(Math.max(0, nextOffset));
  }, []);

  const onLedgerPageChange = useCallback((nextOffset: number) => {
    setLedgerOffset(Math.max(0, nextOffset));
  }, []);

  const onLedgerExportCsv = useCallback(async () => {
    if (!id) {
      return;
    }
    setLedgerExporting(true);
    setLedgerExportError(undefined);
    try {
      const result = await exportCustomerBalanceCsv(id);
      triggerBlobDownload(result.blob, `customer-${id}-ledger.csv`);
      toast.success('Ledger CSV exported');
    } catch (err: unknown) {
      const nextError = err instanceof Error ? err : new Error(String(err));
      setLedgerExportError(nextError);
      toast.error(nextError.message);
    } finally {
      setLedgerExporting(false);
    }
  }, [id]);

  const onStatementLoad = useCallback(() => {
    if (!statementMonth) {
      return;
    }
    setStatementMonthApplied(statementMonth);
  }, [statementMonth]);

  const onSaveTaxProfile = useCallback(async () => {
    if (!id || !canSaveTax) {
      return;
    }
    const trimmedRate = draftTaxRateBps.trim();
    const parsedRate = trimmedRate === '' ? undefined : Number.parseInt(trimmedRate, 10);
    if (trimmedRate !== '' && (parsedRate == null || Number.isNaN(parsedRate))) {
      setSaveError(new Error('Tax rate (bps) must be an integer'));
      setSaveSuccess(false);
      return;
    }

    setSavingTax(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    try {
      await putCustomerTaxProfile(id, {
        country_code: draftCountryCode.trim(),
        tax_region: draftTaxRegion.trim(),
        tax_scheme: draftTaxScheme.trim(),
        tax_rate_bps: parsedRate,
      });
      setSaveSuccess(true);
      toast.success('Tax profile saved');
      setTaxRefreshToken((value) => value + 1);
    } catch (err: unknown) {
      const nextError = err instanceof Error ? err : new Error(String(err));
      setSaveError(nextError);
      toast.error(nextError.message);
    } finally {
      setSavingTax(false);
    }
  }, [canSaveTax, draftCountryCode, draftTaxRateBps, draftTaxRegion, draftTaxScheme, id]);

  const onSaveProfile = useCallback(async () => {
    if (!id || !canSaveProfile) {
      return;
    }
    const name = draftName.trim();
    if (!name) {
      setProfileSaveError(new Error('Customer name is required'));
      setProfileSaveSuccess(false);
      return;
    }

    setSavingProfile(true);
    setProfileSaveError(undefined);
    setProfileSaveSuccess(false);
    try {
      await patchCustomer(id, {
        name,
        cost_center: draftCostCenter.trim(),
      });
      setProfileSaveSuccess(true);
      toast.success('Customer profile saved');
      setCustomerRefreshToken((value) => value + 1);
    } catch (err: unknown) {
      const nextError = err instanceof Error ? err : new Error(String(err));
      setProfileSaveError(nextError);
      toast.error(nextError.message);
    } finally {
      setSavingProfile(false);
    }
  }, [canSaveProfile, draftCostCenter, draftName, id]);

  useBreadcrumbSegmentLabel(id, customerResource.data?.name);

  return {
    customer: customerResource.data,
    customerFetching: customerResource.fetching,
    customerError: customerResource.error,
    hasCustomerSnapshot: customerResource.data != null,
    paymentEnabled,
    tab,
    onTabChange: setTab,
    balance: balanceResource.data,
    balanceFetching: balanceResource.fetching,
    balanceError: balanceResource.error,
    hasBalanceSnapshot: balanceResource.data != null,
    ledgerItems: ledgerResource.data?.items,
    ledgerTotal: ledgerResource.data?.total ?? 0,
    ledgerLimit: ledgerResource.data?.limit ?? CUSTOMER_DETAIL_LEDGER_PAGE_LIMIT,
    ledgerOffset: ledgerResource.data?.offset ?? ledgerOffset,
    ledgerFetching: ledgerResource.fetching,
    ledgerError: ledgerResource.error,
    hasLedgerSnapshot: ledgerResource.data != null,
    ledgerExporting,
    ledgerExportError,
    onLedgerPageChange,
    onLedgerExportCsv,
    statementMonth,
    onStatementMonthChange: setStatementMonth,
    onStatementLoad,
    statement: statementResource.data,
    statementFetching: statementResource.fetching,
    statementError: statementResource.error,
    hasStatementSnapshot: statementResource.data != null,
    forecast: forecastResource.data,
    forecastFetching: forecastResource.fetching,
    forecastError: forecastResource.error,
    hasForecastSnapshot: forecastResource.data != null,
    wallet: walletResource.data,
    walletFetching: walletResource.fetching,
    walletError: walletResource.error,
    hasWalletSnapshot: walletResource.data != null,
    paymentItems: paymentsResource.data?.items,
    paymentTotal: paymentsResource.data?.total ?? 0,
    paymentLimit: paymentsResource.data?.limit ?? CUSTOMER_DETAIL_PAYMENTS_PAGE_LIMIT,
    paymentOffset: paymentsResource.data?.offset ?? paymentsOffset,
    paymentsFetching: paymentsResource.fetching,
    paymentsError: paymentsResource.error,
    hasPaymentsSnapshot: paymentsResource.data != null,
    onPaymentsPageChange,
    taxProfile: taxResource.data,
    taxFetching: taxResource.fetching,
    taxError: taxResource.error,
    hasTaxSnapshot: taxResource.data != null,
    draftCountryCode,
    draftTaxRegion,
    draftTaxScheme,
    draftTaxRateBps,
    onDraftCountryCodeChange: setDraftCountryCode,
    onDraftTaxRegionChange: setDraftTaxRegion,
    onDraftTaxSchemeChange: setDraftTaxScheme,
    onDraftTaxRateBpsChange: setDraftTaxRateBps,
    savingTax,
    saveError,
    saveSuccess,
    canSaveTax,
    onSaveTaxProfile,
    draftName,
    onDraftNameChange: setDraftName,
    draftCostCenter,
    onDraftCostCenterChange: setDraftCostCenter,
    savingProfile,
    profileSaveError,
    profileSaveSuccess,
    canSaveProfile,
    onSaveProfile,
  };
}
