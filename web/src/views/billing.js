import { el, replaceChildren } from '../lib/dom.js';
import { createResource } from '../lib/fetch_resource.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderTabBar } from '../ui/tab_bar.js';
import * as auth from '../helpers/auth.js';
import * as storage from '../helpers/storage.js';
import { isTenantUser } from '../helpers/permissions.js';
import { formatAmountMicro, formatDecimalDisplay } from '../helpers/money.js';
import {
  isPageBlockingError,
  mapServiceError,
} from '../helpers/service_error.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { touchCustomerContext, isCustomerUuid, shortCustomerId } from '../helpers/customer_context.js';
import { renderRecentCustomers } from '../ui/recent_customers.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { validateCustomerIdField } from '../helpers/validators.js';
import { renderAlertBanner } from '../ui/alert_banner.js';
import { renderIcon } from '../ui/icon.js';
import {
  createSortState,
  toggleSort,
  sortRows,
  sortableTh,
  tableSkeletonRows,
  renderEmptyState,
} from '../ui/data_table.js';

const LEDGER_PAGE = 50;
const INVOICE_PAGE = 50;

/**
 * @param {string} customerId
 * @param {number} page
 * @param {boolean} adminWide
 */
function buildInvoiceUrl(customerId, page, adminWide) {
  const offset = page * INVOICE_PAGE;
  const params = new URLSearchParams({
    limit: String(INVOICE_PAGE),
    offset: String(offset),
  });
  if (customerId) params.set('customer_id', customerId);
  if (!customerId && !adminWide) return null;
  return `/api/v1/billing/invoices?${params.toString()}`;
}

/**
 * @param {string} uuid
 * @returns {HTMLElement|string}
 */
function renderMiddleTruncateUuid(uuid) {
  if (!uuid) return '—';
  const start = uuid.slice(0, 8);
  const end = uuid.slice(-8);
  return el('span', { className: 'font-mono text-hint', title: uuid }, `${start}…${end}`);
}

/**
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams, navigate: (path: string) => void }} ctx
 */
export function mount(container, ctx) {
  let destroyed = false;
  const user = auth.getUser();
  const tenant = isTenantUser(user?.role);

  let tab = 'wallet';
  let ledgerPage = 0;
  let invoicePage = 0;
  let customerInput = tenant
    ? (user?.customer_id ?? '')
    : (ctx.query.get('customer_id') ?? storage.getLastCustomerId() ?? '');

  const walletState = { data: null, loading: false, error: null };
  const balanceState = { data: null, loading: false, error: null };
  const invoicesState = { data: null, loading: false, error: null };

  let lastWalletError = null;
  let lastBalanceError = null;
  let lastInvoicesError = null;

  const ledgerSortState = createSortState('created_at', 'desc');
  const invoiceSortState = createSortState('billing_month', 'desc');
  let customerInputError = null;

  if (customerInput && isCustomerUuid(customerInput)) {
    touchCustomerContext(customerInput);
  }

  function customerId() {
    return tenant ? (user?.customer_id ?? '') : (customerInput.trim() || '');
  }

  function applyCustomerFilter() {
    const id = customerInput.trim();
    customerInputError = tenant ? null : validateCustomerIdField(id);
    if (customerInputError) {
      render();
      return;
    }
    if (isCustomerUuid(id)) touchCustomerContext(id);
    if (id) ctx.navigate(`/billing?customer_id=${encodeURIComponent(id)}`);
    else ctx.navigate('/billing');
    ledgerPage = 0;
    invoicePage = 0;
    walletResource.reload();
    balanceResource.reload();
    invoicesResource.reload();
  }

  function renderBlocking(err) {
    if (!err) return null;
    const view = mapServiceError(err);
    if (!isPageBlockingError(view) && view.kind !== 'empty') return null;
    return renderErrorBlock(err);
  }

  function ledgerSlice() {
    const rows = balanceState.data?.ledger ?? [];
    const start = ledgerPage * LEDGER_PAGE;
    return { rows: rows.slice(start, start + LEDGER_PAGE), total: rows.length };
  }

  function render() {
    if (destroyed) return;

    const cid = customerId();
    const invoiceItemsRaw = invoicesState.data?.items ?? invoicesState.data?.invoices ?? [];
    const invoiceItems = sortRows(invoiceItemsRaw, invoiceSortState, {
      id: (inv) => inv.id ?? '',
      billing_month: (inv) => inv.billing_month ?? '',
      status: (inv) => inv.status ?? '',
      total_micro: (inv) => Number(inv.total_micro ?? 0),
      customer_id: (inv) => inv.customer_id ?? '',
    });
    const invoiceTotal = invoicesState.data?.total ?? 0;
    const invoicePages = Math.max(1, Math.ceil(invoiceTotal / INVOICE_PAGE));
    const ledgerRaw = ledgerSlice();
    const ledgerRows = sortRows(ledgerRaw.rows, ledgerSortState, {
      id: (row) => row.id ?? '',
      type: (row) => row.type ?? '',
      amount: (row) => Number(row.amount ?? 0),
      campaign_id: (row) => row.campaign_id ?? '',
      created_at: (row) => row.created_at ?? '',
    });
    const ledgerView = { rows: ledgerRows, total: ledgerRaw.total };

    const onLedgerSort = (key) => {
      toggleSort(ledgerSortState, key);
      render();
    };
    const onInvoiceSort = (key) => {
      toggleSort(invoiceSortState, key);
      render();
    };

    function focusBillingCustomer() {
      const input = container.querySelector('#billing-customer-input');
      if (input instanceof HTMLElement) input.focus();
    }

    replaceChildren(container,
      el('div', { className: 'page-header' },
        cid && !tenant
          ? renderBreadcrumbs([
            { label: 'Billing', href: '/billing' },
            { label: shortCustomerId(cid, 12) },
          ])
          : null,
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            renderIcon('credit-card', { size: 20, className: 'text-muted' }),
            el('h1', { className: 'page-header__title' }, 'Billing'),
          ),
        ),
        renderRecentCustomers({ tenant }),
        !tenant
          ? el('div', { className: 'input-group mt-4' },
            el('input', {
              id: 'billing-customer-input',
              type: 'text',
              className: 'form-input' + (customerInputError ? ' form-input--error' : ''),
              placeholder: 'customer_id (UUID)',
              value: customerInput,
              onInput: (e) => {
                customerInput = e.target.value;
                customerInputError = customerInput.trim()
                  ? validateCustomerIdField(customerInput)
                  : null;
              },
            }),
            el('button', {
              type: 'button',
              className: 'btn btn--secondary',
              onClick: applyCustomerFilter,
            }, 'Apply'),
          )
          : null,
        customerInputError
          ? renderAlertBanner({ variant: 'error', message: customerInputError })
          : null,
        tenant && cid
          ? el('p', { className: 'text-muted', style: { fontSize: 13, marginTop: 8 } },
            'Customer: ',
            el('a', { href: `/customers/${cid}`, className: 'font-mono' }, cid),
          )
          : null,
      ),
      renderTabBar({ tabs: [
        { id: 'wallet', label: 'Wallet' },
        { id: 'ledger', label: 'Ledger' },
        { id: 'invoices', label: 'Invoices' },
      ], active: tab, onChange: (t) => {
        tab = t;
        if (t === 'wallet') walletResource.reload();
        else if (t === 'ledger') balanceResource.reload();
        else invoicesResource.reload();
        render();
      } }),
      tab === 'wallet'
        ? el('div', { style: { marginTop: 24 } },
          !cid && !tenant
            ? renderAlertBanner({
              variant: 'info',
              message: 'Enter customer_id for wallet and balance.',
            })
            : null,
          walletState.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
          renderBlocking(walletState.error),
          walletState.data
            ? el('div', { className: 'grid-stats' },
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Balance'),
                el('div', { className: 'metric-card__value font-mono' },
                  formatAmountMicro(walletState.data.balance_micro, walletState.data.currency),
                ),
              ),
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Overdraft'),
                el('div', { className: 'metric-card__value font-mono' },
                  formatAmountMicro(walletState.data.allowed_overdraft_micro, walletState.data.currency),
                ),
              ),
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Low balance'),
                el('div', { className: 'metric-card__value font-mono' },
                  formatAmountMicro(walletState.data.low_balance_threshold_micro, walletState.data.currency),
                ),
              ),
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Provider'),
                el('div', { className: 'metric-card__value' },
                  walletState.data.payment_provider ?? '—',
                  walletState.data.payment_provider_configured ? '' : ' (not configured)',
                ),
              ),
            )
            : null,
        )
        : null,
      tab === 'ledger'
        ? el('div', { style: { marginTop: 24 } },
          !cid
            ? renderAlertBanner({
              variant: 'info',
              message: 'Enter customer_id for ledger.',
            })
            : null,
          balanceState.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
          renderBlocking(balanceState.error),
          balanceState.data
            ? el('div', null,
              el('div', { className: 'metric-row mb-4' },
                el('div', { className: 'metric-card' },
                  el('div', { className: 'metric-card__label' }, 'Balance (PG)'),
                  el('div', { className: 'metric-card__value font-mono' },
                    `${formatDecimalDisplay(balanceState.data.balance)} ${balanceState.data.currency}`,
                  ),
                ),
              ),
              el('div', { className: 'table-wrapper table-wrapper--scroll' },
                el('table', { className: 'data-table' },
                  el('thead', null,
                    el('tr', null,
                      sortableTh('ID', 'id', ledgerSortState, onLedgerSort),
                      sortableTh('Type', 'type', ledgerSortState, onLedgerSort),
                      sortableTh('Amount', 'amount', ledgerSortState, onLedgerSort),
                      sortableTh('Campaign', 'campaign_id', ledgerSortState, onLedgerSort),
                      sortableTh('Created', 'created_at', ledgerSortState, onLedgerSort),
                    ),
                  ),
                  el('tbody', null,
                    balanceState.loading && ledgerView.rows.length === 0
                      ? tableSkeletonRows(5)
                      : null,
                    !balanceState.loading && ledgerView.rows.length === 0
                      ? el('tr', null,
                        el('td', { colSpan: 5 },
                          renderEmptyState({
                            title: 'No ledger entries',
                            description: 'Transactions appear when spend or top-ups are recorded.',
                          }),
                        ),
                      )
                      : null,
                    ledgerView.rows.map((row) =>
                      el('tr', null,
                        el('td', null, renderMiddleTruncateUuid(row.id)),
                        el('td', null, row.type),
                        el('td', { className: 'font-mono' }, formatDecimalDisplay(row.amount)),
                        el('td', null, row.campaign_id ? renderMiddleTruncateUuid(row.campaign_id) : '—'),
                        el('td', { className: 'text-muted' },
                          row.created_at ? new Date(row.created_at).toLocaleString() : '—',
                        ),
                      ),
                    ),
                  ),
                ),
              ),
              ledgerView.total > LEDGER_PAGE
                ? el('div', {
                  className: 'flex items-center gap-2 mt-4',
                  style: { justifyContent: 'flex-end' },
                },
                  el('button', {
                    type: 'button',
                    className: 'btn btn--secondary btn--sm',
                    disabled: ledgerPage === 0,
                    onClick: () => { ledgerPage = Math.max(0, ledgerPage - 1); render(); },
                  }, 'Prev'),
                  el('span', { className: 'text-muted', style: { fontSize: 12 } },
                    `${ledgerPage + 1} / ${Math.ceil(ledgerView.total / LEDGER_PAGE)}`,
                  ),
                  el('button', {
                    type: 'button',
                    className: 'btn btn--secondary btn--sm',
                    disabled: (ledgerPage + 1) * LEDGER_PAGE >= ledgerView.total,
                    onClick: () => { ledgerPage += 1; render(); },
                  }, 'Next'),
                )
                : null,
            )
            : null,
        )
        : null,
      tab === 'invoices'
        ? el('div', { style: { marginTop: 24 } },
          invoicesState.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
          renderBlocking(invoicesState.error),
          invoicesState.data
            ? el('div', null,
              el('div', { className: 'table-wrapper table-wrapper--scroll' },
                el('table', { className: 'data-table' },
                  el('thead', null,
                    el('tr', null,
                      sortableTh('ID', 'id', invoiceSortState, onInvoiceSort),
                      sortableTh('Month', 'billing_month', invoiceSortState, onInvoiceSort),
                      sortableTh('Status', 'status', invoiceSortState, onInvoiceSort),
                      sortableTh('Amount', 'total_micro', invoiceSortState, onInvoiceSort),
                      sortableTh('Customer', 'customer_id', invoiceSortState, onInvoiceSort),
                    ),
                  ),
                  el('tbody', null,
                    invoicesState.loading && invoiceItems.length === 0
                      ? tableSkeletonRows(5)
                      : null,
                    !invoicesState.loading && invoiceItems.length === 0
                      ? el('tr', null,
                        el('td', { colSpan: 5 },
                          renderEmptyState({
                            title: 'No invoices',
                            description: cid
                              ? 'No invoices for this customer yet.'
                              : 'Enter a customer_id to load invoices.',
                            actionLabel: cid ? undefined : 'Focus customer field',
                            onAction: cid ? undefined : focusBillingCustomer,
                          }),
                        ),
                      )
                      : null,
                    invoiceItems.map((inv) =>
                      el('tr', null,
                        el('td', null,
                          el('a', {
                            href: `/billing/invoices/${inv.id}`,
                          }, renderMiddleTruncateUuid(inv.id)),
                        ),
                        el('td', null, inv.billing_month ?? '—'),
                        el('td', null, inv.status ?? '—'),
                        el('td', { className: 'font-mono' },
                          formatAmountMicro(inv.total_micro ?? 0, inv.currency),
                        ),
                        el('td', null,
                          inv.customer_id
                            ? el('a', { href: `/customers/${inv.customer_id}` },
                              renderMiddleTruncateUuid(inv.customer_id),
                            )
                            : '—',
                        ),
                      ),
                    ),
                  ),
                ),
              ),
              invoicePages > 1
                ? el('div', {
                  className: 'flex items-center gap-2 mt-4',
                  style: { justifyContent: 'flex-end' },
                },
                  el('button', {
                    type: 'button',
                    className: 'btn btn--secondary btn--sm',
                    disabled: invoicePage === 0,
                    onClick: () => {
                      invoicePage = Math.max(0, invoicePage - 1);
                      invoicesResource.reload();
                    },
                  }, 'Prev'),
                  el('span', { className: 'text-muted', style: { fontSize: 12 } },
                    `${invoicePage + 1} / ${invoicePages}`,
                  ),
                  el('button', {
                    type: 'button',
                    className: 'btn btn--secondary btn--sm',
                    disabled: invoicePage >= invoicePages - 1,
                    onClick: () => {
                      invoicePage += 1;
                      invoicesResource.reload();
                    },
                  }, 'Next'),
                )
                : null,
            )
            : null,
        )
        : null,
    );
  }

  const walletResource = createResource(
    () => customerId() ? `/api/v1/customers/${customerId()}/wallet` : null,
    {
      skip: () => tab !== 'wallet' || !customerId(),
      onUpdate: (s) => {
        if (s.error !== lastWalletError) {
          lastWalletError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(walletState, s);
        render();
      },
    },
  );

  const balanceResource = createResource(
    () => customerId() ? `/api/v1/customers/${customerId()}/balance` : null,
    {
      skip: () => tab !== 'ledger' || !customerId(),
      onUpdate: (s) => {
        if (s.error !== lastBalanceError) {
          lastBalanceError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(balanceState, s);
        render();
      },
    },
  );

  const invoicesResource = createResource(
    () => buildInvoiceUrl(customerId(), invoicePage, !tenant),
    {
      skip: () => tab !== 'invoices' || !buildInvoiceUrl(customerId(), invoicePage, !tenant),
      onUpdate: (s) => {
        if (s.error !== lastInvoicesError) {
          lastInvoicesError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(invoicesState, s);
        render();
      },
    },
  );

  render();

  return {
    destroy() {
      destroyed = true;
      walletResource.destroy();
      balanceResource.destroy();
      invoicesResource.destroy();
    },
  };
}
