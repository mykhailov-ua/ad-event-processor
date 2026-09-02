import { useCallback, useEffect, useState } from 'react';
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
import { getCustomer, patchCustomerCostCenter } from '@/api/customers_api';
import {
  CustomerDetail,
  type CustomerDetailTab,
} from '@/domains/customers/customer_detail';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';

function currentMonthValue(): string {
  const now = new Date();
  const year = now.getUTCFullYear();
  const month = String(now.getUTCMonth() + 1).padStart(2, '0');
  return `${year}-${month}`;
}

function skipFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

function taxProfileToDraft(profile: {
  country_code?: string;
  tax_region?: string;
  tax_scheme?: string;
  tax_rate_bps?: number;
}) {
  return {
    countryCode: profile.country_code ?? '',
    taxRegion: profile.tax_region ?? '',
    taxScheme: profile.tax_scheme ?? '',
    taxRateBps:
      profile.tax_rate_bps != null && !Number.isNaN(profile.tax_rate_bps)
        ? String(profile.tax_rate_bps)
        : '',
  };
}

const PAYMENTS_PAGE_LIMIT = 50;
const LEDGER_PAGE_LIMIT = 50;

function triggerBlobDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function CustomerDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [tab, setTab] = useState<CustomerDetailTab>('profile');
  const [statementMonth, setStatementMonth] = useState(currentMonthValue);
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
  const [savingCostCenter, setSavingCostCenter] = useState(false);
  const [costCenterSaveError, setCostCenterSaveError] = useState<Error | undefined>();
  const [costCenterSaveSuccess, setCostCenterSaveSuccess] = useState(false);
  const [customerRefreshToken, setCustomerRefreshToken] = useState(0);

  useEffect(() => {
    setPaymentsOffset(0);
    setLedgerOffset(0);
  }, [id]);

  const { user } = useSession();
  const canSaveTax = user?.permissions?.includes('customers:write') ?? false;
  const canSaveCostCenter = canSaveTax;

  const customerResource = useResource(
    (signal) => {
      if (!id) {
        return Promise.reject(new Error('Customer id is required'));
      }
      return getCustomer(id, signal);
    },
    [id, customerRefreshToken],
  );

  useEffect(() => {
    if (!customerResource.data) {
      return;
    }
    setDraftCostCenter(customerResource.data.cost_center ?? '');
  }, [customerResource.data]);

  const balanceResource = useResource(
    (signal) => {
      if (!id || tab !== 'balance') {
        return skipFetch();
      }
      return getCustomerBalance(id, signal);
    },
    [id, tab],
  );

  const ledgerResource = useResource(
    (signal) => {
      if (!id || tab !== 'ledger') {
        return skipFetch();
      }
      return listCustomerLedger(id, { limit: LEDGER_PAGE_LIMIT, offset: ledgerOffset }, signal);
    },
    [id, tab, ledgerOffset],
  );

  const statementResource = useResource(
    (signal) => {
      if (!id || tab !== 'statement' || !statementMonthApplied) {
        return skipFetch();
      }
      return getCustomerBillingStatement(id, statementMonthApplied, signal);
    },
    [id, tab, statementMonthApplied],
  );

  const forecastResource = useResource(
    (signal) => {
      if (!id || tab !== 'forecast') {
        return skipFetch();
      }
      return getCustomerBillingForecast(id, signal);
    },
    [id, tab],
  );

  const walletResource = useResource(
    (signal) => {
      if (!id || tab !== 'wallet') {
        return skipFetch();
      }
      return getCustomerWallet(id, signal);
    },
    [id, tab],
  );

  const paymentsResource = useResource(
    (signal) => {
      if (!id || tab !== 'payments') {
        return skipFetch();
      }
      return listCustomerPayments(
        id,
        { limit: PAYMENTS_PAGE_LIMIT, offset: paymentsOffset },
        signal,
      );
    },
    [id, tab, paymentsOffset],
  );

  const taxResource = useResource(
    (signal) => {
      if (!id || tab !== 'tax') {
        return skipFetch();
      }
      return getCustomerTaxProfile(id, signal);
    },
    [id, tab, taxRefreshToken],
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
    } catch (err) {
      setLedgerExportError(err instanceof Error ? err : new Error(String(err)));
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
      setTaxRefreshToken((value) => value + 1);
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSavingTax(false);
    }
  }, [
    canSaveTax,
    draftCountryCode,
    draftTaxRateBps,
    draftTaxRegion,
    draftTaxScheme,
    id,
  ]);

  const onSaveCostCenter = useCallback(async () => {
    if (!id || !canSaveCostCenter) {
      return;
    }

    setSavingCostCenter(true);
    setCostCenterSaveError(undefined);
    setCostCenterSaveSuccess(false);
    try {
      await patchCustomerCostCenter(id, { cost_center: draftCostCenter.trim() });
      setCostCenterSaveSuccess(true);
      setCustomerRefreshToken((value) => value + 1);
    } catch (err) {
      setCostCenterSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSavingCostCenter(false);
    }
  }, [canSaveCostCenter, draftCostCenter, id]);

  useBreadcrumbSegmentLabel(id, customerResource.data?.name);

  return (
    <CustomerDetail
      customer={customerResource.data}
      customerFetching={customerResource.fetching}
      customerError={customerResource.error}
      hasCustomerSnapshot={customerResource.data != null}
      tab={tab}
      onTabChange={setTab}
      balance={balanceResource.data}
      balanceFetching={balanceResource.fetching}
      balanceError={balanceResource.error}
      hasBalanceSnapshot={balanceResource.data != null}
      ledgerItems={ledgerResource.data?.items ?? []}
      ledgerTotal={ledgerResource.data?.total ?? 0}
      ledgerLimit={ledgerResource.data?.limit ?? LEDGER_PAGE_LIMIT}
      ledgerOffset={ledgerResource.data?.offset ?? ledgerOffset}
      ledgerFetching={ledgerResource.fetching}
      ledgerError={ledgerResource.error}
      hasLedgerSnapshot={ledgerResource.data != null}
      ledgerExporting={ledgerExporting}
      ledgerExportError={ledgerExportError}
      onLedgerPageChange={onLedgerPageChange}
      onLedgerExportCsv={onLedgerExportCsv}
      statementMonth={statementMonth}
      onStatementMonthChange={setStatementMonth}
      onStatementLoad={onStatementLoad}
      statement={statementResource.data}
      statementFetching={statementResource.fetching}
      statementError={statementResource.error}
      hasStatementSnapshot={statementResource.data != null}
      forecast={forecastResource.data}
      forecastFetching={forecastResource.fetching}
      forecastError={forecastResource.error}
      hasForecastSnapshot={forecastResource.data != null}
      wallet={walletResource.data}
      walletFetching={walletResource.fetching}
      walletError={walletResource.error}
      hasWalletSnapshot={walletResource.data != null}
      paymentItems={paymentsResource.data?.items ?? []}
      paymentTotal={paymentsResource.data?.total ?? 0}
      paymentLimit={paymentsResource.data?.limit ?? PAYMENTS_PAGE_LIMIT}
      paymentOffset={paymentsResource.data?.offset ?? paymentsOffset}
      paymentsFetching={paymentsResource.fetching}
      paymentsError={paymentsResource.error}
      hasPaymentsSnapshot={paymentsResource.data != null}
      onPaymentsPageChange={onPaymentsPageChange}
      taxProfile={taxResource.data}
      taxFetching={taxResource.fetching}
      taxError={taxResource.error}
      hasTaxSnapshot={taxResource.data != null}
      draftCountryCode={draftCountryCode}
      draftTaxRegion={draftTaxRegion}
      draftTaxScheme={draftTaxScheme}
      draftTaxRateBps={draftTaxRateBps}
      onDraftCountryCodeChange={setDraftCountryCode}
      onDraftTaxRegionChange={setDraftTaxRegion}
      onDraftTaxSchemeChange={setDraftTaxScheme}
      onDraftTaxRateBpsChange={setDraftTaxRateBps}
      savingTax={savingTax}
      saveError={saveError}
      saveSuccess={saveSuccess}
      canSaveTax={canSaveTax}
      onSaveTaxProfile={onSaveTaxProfile}
      draftCostCenter={draftCostCenter}
      onDraftCostCenterChange={setDraftCostCenter}
      savingCostCenter={savingCostCenter}
      costCenterSaveError={costCenterSaveError}
      costCenterSaveSuccess={costCenterSaveSuccess}
      canSaveCostCenter={canSaveCostCenter}
      onSaveCostCenter={onSaveCostCenter}
    />
  );
}
