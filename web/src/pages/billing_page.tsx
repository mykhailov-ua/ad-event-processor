import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import type {
  InvoiceDTO,
  InvoiceListResponse,
  LedgerEntryDTO,
  LedgerListResponse,
  WalletBalanceDTO,
} from '../types/api/index.js';
import * as auth from '../helpers/auth.js';
import * as storage from '../helpers/storage.js';
import { isBuyer, can } from '../helpers/permissions.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { formatAmountMicro, formatDecimalDisplay } from '../helpers/money.js';
import {
  isPageBlockingError,
  mapServiceError,
} from '../helpers/service_error.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { touchCustomerContext, isCustomerUuid, shortCustomerId } from '../helpers/customer_context.js';
import { validateCustomerIdField } from '../helpers/validators.js';
import { displayLabel } from '../helpers/display_labels.js';
import { createSortState, sortRows, toggleSort } from '../lib/table_sort.js';
import { useResource } from '../hooks/use_resource.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { CopyableUuid } from '../components/copyable_uuid.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { Icon } from '../components/icon.js';
import { PaginationBar } from '../components/pagination_bar.js';
import { RecentCustomers } from '../components/recent_customers.js';
import { StatusBadge } from '../components/status_badge.js';
import { TabBar } from '../components/tab_bar.js';
import { BillingExportsSection } from '../components/billing_exports_section.js';
import { BillingSelfServeSection } from '../components/billing_selfserve_section.js';

const LEDGER_PAGE = 50;
const INVOICE_PAGE = 50;

type BillingTab = 'wallet' | 'ledger' | 'invoices' | 'exports';

function parseTab(raw: string | null): BillingTab {
  if (raw === 'ledger' || raw === 'invoices' || raw === 'exports') return raw;
  return 'wallet';
}

function buildInvoiceUrl(customerId: string, page: number, adminWide: boolean): string | null {
  const offset = page * INVOICE_PAGE;
  const params = new URLSearchParams({
    limit: String(INVOICE_PAGE),
    offset: String(offset),
  });
  if (customerId) params.set('customer_id', customerId);
  if (!customerId && !adminWide) return null;
  return `/api/v1/billing/invoices?${params.toString()}`;
}

function buildLedgerUrl(customerId: string, page: number): string | null {
  if (!customerId) return null;
  const offset = page * LEDGER_PAGE;
  const params = new URLSearchParams({
    limit: String(LEDGER_PAGE),
    offset: String(offset),
  });
  return `/api/v1/customers/${customerId}/ledger?${params.toString()}`;
}

function TableSkeleton({ cols, rows = 5 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, rowIndex) => (
        <tr key={`skel-${rowIndex}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, colIndex) => (
            <td key={`skel-${rowIndex}-${colIndex}`}><span className="skeleton-bar" /></td>
          ))}
        </tr>
      ))}
    </>
  );
}

function BlockingError({ error }: { error: unknown }) {
  if (!error) return null;
  const view = mapServiceError(error);
  if (!isPageBlockingError(view) && view.kind !== 'empty') return null;
  return <ErrorBlock error={error} />;
}

/**
 * Billing wallet / ledger / invoices / exports tabs.
 */
export function BillingPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const buyerView = isBuyer(user?.role);
  const canReadCustomers = can(user?.permissions ?? [], 'customers:read');

  const tab = parseTab(searchParams.get('tab'));
  const [ledgerPage, setLedgerPage] = useState(0);
  const [invoicePage, setInvoicePage] = useState(0);
  const [customerInput, setCustomerInput] = useState(() => (
    sessionScoped
      ? boundCustomerId(user)
      : (searchParams.get('customer_id') ?? storage.getLastCustomerId() ?? '')
  ));
  const [customerInputError, setCustomerInputError] = useState<string | null>(null);
  const [ledgerSortState, setLedgerSortState] = useState(() => createSortState('created_at', 'desc'));
  const [invoiceSortState, setInvoiceSortState] = useState(() => createSortState('billing_month', 'desc'));
  const customerInputRef = useRef<HTMLInputElement>(null);

  const customerId = sessionScoped ? boundCustomerId(user) : (customerInput.trim() || '');

  useEffect(() => {
    if (customerId && isCustomerUuid(customerId)) {
      touchCustomerContext(customerId);
    }
  }, [customerId]);

  const walletUrl = customerId ? `/api/v1/customers/${customerId}/wallet` : null;
  const balanceUrl = customerId ? `/api/v1/customers/${customerId}/balance` : null;
  const ledgerUrl = buildLedgerUrl(customerId, ledgerPage);
  const invoiceUrl = buildInvoiceUrl(customerId, invoicePage, !sessionScoped);

  const wallet = useResource<WalletBalanceDTO>(walletUrl, { skip: tab !== 'wallet' || !customerId });
  const balance = useResource<WalletBalanceDTO>(balanceUrl, { skip: tab !== 'ledger' || !customerId });
  const ledger = useResource<LedgerListResponse>(ledgerUrl, { skip: tab !== 'ledger' || !ledgerUrl });
  const invoices = useResource<InvoiceListResponse>(invoiceUrl, { skip: tab !== 'invoices' || !invoiceUrl });

  useEffect(() => {
    if (wallet.error) surfaceServiceErrorToast(wallet.error);
  }, [wallet.error]);
  useEffect(() => {
    if (balance.error) surfaceServiceErrorToast(balance.error);
  }, [balance.error]);
  useEffect(() => {
    if (ledger.error) surfaceServiceErrorToast(ledger.error);
  }, [ledger.error]);
  useEffect(() => {
    if (invoices.error) surfaceServiceErrorToast(invoices.error);
  }, [invoices.error]);

  const applyCustomerFilter = () => {
    const id = customerInput.trim();
    const err = sessionScoped ? null : validateCustomerIdField(id);
    setCustomerInputError(err);
    if (err) return;
    if (isCustomerUuid(id)) touchCustomerContext(id);
    setLedgerPage(0);
    setInvoicePage(0);
    if (id) navigate(`/billing?customer_id=${encodeURIComponent(id)}${tab !== 'wallet' ? `&tab=${tab}` : ''}`);
    else navigate(tab !== 'wallet' ? `/billing?tab=${tab}` : '/billing');
  };

  const setTab = (next: BillingTab) => {
    const params = new URLSearchParams(searchParams);
    if (next === 'wallet') params.delete('tab');
    else params.set('tab', next);
    navigate(`/billing?${params.toString()}`);
  };

  const invoiceItems = useMemo(() => {
    const raw = invoices.data?.items ?? invoices.data?.invoices ?? [];
    return sortRows(raw, invoiceSortState, {
      id: (inv: InvoiceDTO) => inv.id ?? '',
      billing_month: (inv: InvoiceDTO) => inv.billing_month ?? '',
      status: (inv: InvoiceDTO) => inv.status ?? '',
      total_micro: (inv: InvoiceDTO) => Number(inv.total_micro ?? 0),
      customer_id: (inv: InvoiceDTO) => inv.customer_id ?? '',
    });
  }, [invoices.data, invoiceSortState]);

  const ledgerRows = useMemo(() => {
    const raw = ledger.data?.items ?? [];
    return sortRows(raw, ledgerSortState, {
      id: (row: LedgerEntryDTO) => row.id ?? '',
      type: (row: LedgerEntryDTO) => row.type ?? '',
      amount: (row: LedgerEntryDTO) => Number(row.amount ?? 0),
      campaign_id: (row: LedgerEntryDTO) => row.campaign_id ?? '',
      created_at: (row: LedgerEntryDTO) => row.created_at ?? '',
    });
  }, [ledger.data, ledgerSortState]);

  const invoiceTotal = invoices.data?.total ?? 0;
  const invoicePages = Math.max(1, Math.ceil(invoiceTotal / INVOICE_PAGE));
  const ledgerTotal = ledger.data?.total ?? 0;

  const onLedgerSort = useCallback((key: string) => {
    setLedgerSortState((prev) => {
      const next = { ...prev };
      toggleSort(next, key);
      return next;
    });
  }, []);

  const onInvoiceSort = useCallback((key: string) => {
    setInvoiceSortState((prev) => {
      const next = { ...prev };
      toggleSort(next, key);
      return next;
    });
  }, []);

  const sortHeader = (
    label: string,
    key: string,
    state: ReturnType<typeof createSortState>,
    onSort: (key: string) => void,
  ) => {
    const active = state.key === key;
    const dir = active ? state.dir : '';
    return (
      <th scope="col">
        <button
          type="button"
          className={`sortable-th${active ? ' sortable-th--active' : ''}`}
          onClick={() => onSort(key)}
        >
          {label}
          {active ? (dir === 'asc' ? ' ↑' : ' ↓') : ''}
        </button>
      </th>
    );
  };

  const tabs = [
    { id: 'wallet', label: 'Wallet' },
    ...(canReadCustomers
      ? [
        { id: 'ledger', label: 'Ledger' },
        { id: 'invoices', label: 'Invoices' },
        { id: 'exports', label: 'Exports' },
      ]
      : []),
  ];

  const focusBillingCustomer = () => {
    customerInputRef.current?.focus();
  };

  return (
    <>
      <div className="page-header">
        {customerId && !sessionScoped ? (
          <Breadcrumbs items={[
            { label: 'Billing', href: '/billing' },
            { label: shortCustomerId(customerId, 12) },
          ]}
        />
        ) : null}
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="credit-card" size={20} className="text-muted" />
            <h1 className="page-header__title">Billing</h1>
          </div>
        </div>
        <RecentCustomers tenant={sessionScoped && !buyerView} />
        {!sessionScoped ? (
          <div className="input-group mt-4">
            <input
              ref={customerInputRef}
              id="billing-customer-input"
              type="text"
              className={`form-input${customerInputError ? ' form-input--error' : ''}`}
              placeholder="customer_id (UUID)"
              value={customerInput}
              onChange={(e) => {
                setCustomerInput(e.target.value);
                setCustomerInputError(
                  e.target.value.trim() ? validateCustomerIdField(e.target.value) : null,
                );
              }}
            />
            <Button label="Apply" variant="secondary" onClick={applyCustomerFilter} />
          </div>
        ) : null}
        {customerInputError ? (
          <AlertBanner variant="error" message={customerInputError} />
        ) : null}
        {sessionScoped && customerId ? (
          <p className="text-muted text-sm mt-2">
            Customer:{' '}
            <a href={`/customers/${customerId}`} className="font-mono">{customerId}</a>
          </p>
        ) : null}
      </div>

      <TabBar tabs={tabs} active={tab} onChange={(id) => setTab(id as BillingTab)} />

      {tab === 'wallet' ? (
        <div className="section-block stack">
          {!customerId && !sessionScoped ? (
            <AlertBanner variant="info" message="Enter customer_id for wallet and balance." />
          ) : null}
          {wallet.loading ? <span className="text-muted">Loading…</span> : null}
          <BlockingError error={wallet.error} />
          {wallet.data ? (
            <div className="grid-stats">
              <div className="metric-card">
                <div className="metric-card__label">Balance</div>
                <div className="metric-card__value font-mono">
                  {formatAmountMicro(wallet.data.balance_micro ?? 0, wallet.data.currency)}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-card__label">Overdraft</div>
                <div className="metric-card__value font-mono">
                  {formatAmountMicro(wallet.data.allowed_overdraft_micro ?? 0, wallet.data.currency)}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-card__label">Low balance</div>
                <div className="metric-card__value font-mono">
                  {formatAmountMicro(wallet.data.low_balance_threshold_micro ?? 0, wallet.data.currency)}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-card__label">Provider</div>
                <div className="metric-card__value">
                  {wallet.data.payment_provider ?? '—'}
                  {wallet.data.payment_provider_configured ? '' : ' (not configured)'}
                </div>
              </div>
            </div>
          ) : null}
          {sessionScoped ? (
            <BillingSelfServeSection customerId={customerId} buyerMode={buyerView} />
          ) : null}
        </div>
      ) : null}

      {tab === 'ledger' ? (
        <div className="section-block stack">
          {!customerId ? (
            <AlertBanner variant="info" message="Enter customer_id for ledger." />
          ) : null}
          {balance.loading ? <span className="text-muted">Loading…</span> : null}
          <BlockingError error={balance.error} />
          <BlockingError error={ledger.error} />
          {balance.data ? (
            <div>
              <div className="metric-row mb-4">
                <div className="metric-card">
                  <div className="metric-card__label">Balance (PG)</div>
                  <div className="metric-card__value font-mono">
                    {`${formatDecimalDisplay(String(balance.data.balance ?? ''))} ${balance.data.currency ?? ''}`}
                  </div>
                </div>
              </div>
              {ledgerTotal > LEDGER_PAGE ? (
                <div className="mb-4">
                  <FilterToolbar
                    pagination={(
                      <PaginationBar
                        label={`${ledgerPage + 1} / ${Math.ceil(ledgerTotal / LEDGER_PAGE)}`}
                        prevDisabled={ledgerPage === 0}
                        nextDisabled={(ledgerPage + 1) * LEDGER_PAGE >= ledgerTotal}
                        onPrev={() => setLedgerPage((p) => Math.max(0, p - 1))}
                        onNext={() => setLedgerPage((p) => p + 1)}
                      />
                    )}
                  />
                </div>
              ) : null}
              <div className="table-wrapper table-wrapper--scroll elevation-raised">
                <table className="data-table">
                  <thead>
                    <tr>
                      {sortHeader('ID', 'id', ledgerSortState, onLedgerSort)}
                      {sortHeader('Type', 'type', ledgerSortState, onLedgerSort)}
                      {sortHeader('Amount', 'amount', ledgerSortState, onLedgerSort)}
                      {sortHeader('Campaign', 'campaign_id', ledgerSortState, onLedgerSort)}
                      {sortHeader('Created', 'created_at', ledgerSortState, onLedgerSort)}
                    </tr>
                  </thead>
                  <tbody>
                    {ledger.loading && ledgerRows.length === 0 ? <TableSkeleton cols={5} /> : null}
                    {!ledger.loading && ledgerRows.length === 0 ? (
                      <tr>
                        <td colSpan={5}>
                          <div className="empty-state">
                            <div className="empty-state__title">No ledger entries</div>
                            <div className="empty-state__desc text-muted text-sm">
                              Transactions appear when spend or top-ups are recorded.
                            </div>
                          </div>
                        </td>
                      </tr>
                    ) : null}
                    {ledgerRows.map((row) => (
                      <tr key={row.id}>
                        <td><CopyableUuid uuid={row.id ?? ''} /></td>
                        <td>{displayLabel(row.type)}</td>
                        <td className="font-mono">{formatDecimalDisplay(String(row.amount ?? ''))}</td>
                        <td>
                          {row.campaign_id ? <CopyableUuid uuid={row.campaign_id} /> : '—'}
                        </td>
                        <td className="text-muted">
                          {row.created_at ? new Date(row.created_at).toLocaleString() : '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : null}
        </div>
      ) : null}

      {tab === 'invoices' ? (
        <div className="section-block stack">
          {invoices.loading ? <span className="text-muted">Loading…</span> : null}
          <BlockingError error={invoices.error} />
          {invoices.data ? (
            <div>
              {invoicePages > 1 ? (
                <div className="mb-4">
                  <FilterToolbar
                    pagination={(
                      <PaginationBar
                        label={`${invoicePage + 1} / ${invoicePages}`}
                        prevDisabled={invoicePage === 0}
                        nextDisabled={invoicePage >= invoicePages - 1}
                        onPrev={() => setInvoicePage((p) => Math.max(0, p - 1))}
                        onNext={() => setInvoicePage((p) => p + 1)}
                      />
                    )}
                  />
                </div>
              ) : null}
              <div className="table-wrapper table-wrapper--scroll elevation-raised">
                <table className="data-table">
                  <thead>
                    <tr>
                      {sortHeader('ID', 'id', invoiceSortState, onInvoiceSort)}
                      {sortHeader('Month', 'billing_month', invoiceSortState, onInvoiceSort)}
                      {sortHeader('Status', 'status', invoiceSortState, onInvoiceSort)}
                      {sortHeader('Amount', 'total_micro', invoiceSortState, onInvoiceSort)}
                      {sortHeader('Customer', 'customer_id', invoiceSortState, onInvoiceSort)}
                    </tr>
                  </thead>
                  <tbody>
                    {invoices.loading && invoiceItems.length === 0 ? <TableSkeleton cols={5} /> : null}
                    {!invoices.loading && invoiceItems.length === 0 ? (
                      <tr>
                        <td colSpan={5}>
                          <div className="empty-state">
                            <div className="empty-state__title">No invoices</div>
                            <div className="empty-state__desc text-muted text-sm">
                              {customerId
                                ? 'No invoices for this customer yet.'
                                : 'Enter a customer_id to load invoices.'}
                            </div>
                            {!customerId ? (
                              <button
                                type="button"
                                className="empty-state__action"
                                onClick={focusBillingCustomer}
                              >
                                Focus customer field
                              </button>
                            ) : null}
                          </div>
                        </td>
                      </tr>
                    ) : null}
                    {invoiceItems.map((inv) => (
                      <tr key={inv.id}>
                        <td>
                          <a href={`/billing/invoices/${inv.id}`}>
                            <CopyableUuid uuid={inv.id ?? ''} />
                          </a>
                        </td>
                        <td>{inv.billing_month ?? '—'}</td>
                        <td>
                          {inv.status ? <StatusBadge status={inv.status} kind="invoice" /> : '—'}
                        </td>
                        <td className="font-mono">
                          {formatAmountMicro(inv.total_micro ?? 0, inv.currency)}
                        </td>
                        <td>
                          {inv.customer_id ? (
                            <a href={`/customers/${inv.customer_id}`}>
                              <CopyableUuid uuid={inv.customer_id} />
                            </a>
                          ) : '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : null}
        </div>
      ) : null}

      {tab === 'exports' ? (
        <div className="section-block">
          <BillingExportsSection customerId={customerId} tenant={sessionScoped} />
        </div>
      ) : null}
    </>
  );
}
