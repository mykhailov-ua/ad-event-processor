import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import type {
  InvoiceDTO,
  InvoiceListResponse,
  LedgerEntryDTO,
  LedgerListResponse,
  WalletBalanceDTO,
} from '../types/api/index.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { createResource, type ResourceState } from '../lib/fetch_resource.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderTabBar } from '../ui/tab_bar.js';
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
import { renderRecentCustomers } from '../ui/recent_customers.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { validateCustomerIdField } from '../helpers/validators.js';
import { renderAlertBanner } from '../ui/alert_banner.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderIcon } from '../ui/icon.js';
import {
  createSortState,
  toggleSort,
  sortRows,
  sortableTh,
  tableSkeletonRows,
  renderEmptyState,
  renderPaginationBar,
} from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { mountFilterToolbar } from '../ui/filter_toolbar.js';
import { displayLabel } from '../helpers/display_labels.js';
import { mountBillingExportsPanel } from './billing_exports_panel.js';
import { mountBillingSelfServePanel } from './billing_selfserve_panel.js';

import { renderCopyableUuid } from '../ui/copy_text.js';

const LEDGER_PAGE = 50;
const INVOICE_PAGE = 50;

/**
 * Build the billing invoices API URL for a customer and page.
 *
 * @param {string} customerId
 * @param {number} page
 * @param {boolean} adminWide
 * @returns {string|null}
 */
function buildInvoiceUrl(customerId: any, page: any, adminWide: any) {
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
 * Build the paginated customer ledger API URL.
 *
 * @param {string} customerId
 * @param {number} page
 * @returns {string|null}
 */
function buildLedgerUrl(customerId: any, page: any) {
  if (!customerId) return null;
  const offset = page * LEDGER_PAGE;
  const params = new URLSearchParams({
    limit: String(LEDGER_PAGE),
    offset: String(offset),
  });
  return `/api/v1/customers/${customerId}/ledger?${params.toString()}`;
}

/**
 * Render a UUID with middle truncation that is clickable and copyable.
 *
 * @param {string} uuid
 * @returns {HTMLElement|string}
 */
function renderMiddleTruncateUuid(uuid: any) {
  return renderCopyableUuid(uuid);
}

/**
 * Mount the billing view with wallet, ledger, and invoice tabs.
 *
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams, navigate: (path: string) => void }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const buyerView = isBuyer(user?.role);
  const canReadCustomers = can(user?.permissions ?? [], 'customers:read');

  let tab = 'wallet';
  let ledgerPage = 0;
  let invoicePage = 0;
  let customerInput = sessionScoped
    ? boundCustomerId(user)
    : (ctx.query.get('customer_id') ?? storage.getLastCustomerId() ?? '');

  const walletState: ResourceState<WalletBalanceDTO> = { data: null, loading: false, error: null };
  const balanceState: ResourceState<WalletBalanceDTO> = { data: null, loading: false, error: null };
  const ledgerState: ResourceState<LedgerListResponse> = { data: null, loading: false, error: null };
  const invoicesState: ResourceState<InvoiceListResponse> = { data: null, loading: false, error: null };

  let lastWalletError: any = null;
  let lastBalanceError: any = null;
  let lastLedgerError: any = null;
  let lastInvoicesError: any = null;

  const ledgerSortState = createSortState('created_at', 'desc');
  const invoiceSortState = createSortState('billing_month', 'desc');
  const ledgerSortCache = {};
  const invoiceSortCache = {};
  let customerInputError: any = null;
  const exportsSlot = el('div', { 'data-billing-exports': '' });
  const selfServeSlot = el('div', { 'data-billing-selfserve': '' });
  /** @type {import('../lib/router_types.js').ViewHandle | null} */
  let exportsPanelHandle: ViewHandle | null = null;
  /** @type {import('../lib/router_types.js').ViewHandle | null} */
  let selfServePanelHandle: ViewHandle | null = null;

  if (customerInput && isCustomerUuid(customerInput)) {
    touchCustomerContext(customerInput);
  }

  function customerId() {
    return sessionScoped ? boundCustomerId(user) : (customerInput.trim() || '');
  }

  function applyCustomerFilter() {
    const id = customerInput.trim();
    customerInputError = sessionScoped ? null : validateCustomerIdField(id);
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
    ledgerResource.reload();
    invoicesResource.reload();
    if (tab === 'exports') mountExportsPanel();
    if (tab === 'wallet' && sessionScoped) mountSelfServePanel();
  }

  function renderBlocking(err: any) {
    if (!err) return null;
    const view = mapServiceError(err);
    if (!isPageBlockingError(view) && view.kind !== 'empty') return null;
    return renderErrorBlock(err);
  }

  function render() {
    if (destroyed) return;

    const cid = customerId();
    const invoiceItemsRaw = invoicesState.data?.items ?? invoicesState.data?.invoices ?? [];
    const invoiceItems = sortRows(invoiceItemsRaw, invoiceSortState, {
      id: (inv: InvoiceDTO) => inv.id ?? '',
      billing_month: (inv: InvoiceDTO) => inv.billing_month ?? '',
      status: (inv: InvoiceDTO) => inv.status ?? '',
      total_micro: (inv: InvoiceDTO) => Number(inv.total_micro ?? 0),
      customer_id: (inv: InvoiceDTO) => inv.customer_id ?? '',
    }, invoiceSortCache);
    const invoiceTotal = invoicesState.data?.total ?? 0;
    const invoicePages = Math.max(1, Math.ceil(invoiceTotal / INVOICE_PAGE));
    const ledgerItemsRaw = ledgerState.data?.items ?? [];
    const ledgerTotal = ledgerState.data?.total ?? 0;
    const ledgerRows = sortRows(ledgerItemsRaw, ledgerSortState, {
      id: (row: LedgerEntryDTO) => row.id ?? '',
      type: (row: LedgerEntryDTO) => row.type ?? '',
      amount: (row: LedgerEntryDTO) => Number(row.amount ?? 0),
      campaign_id: (row: LedgerEntryDTO) => row.campaign_id ?? '',
      created_at: (row: LedgerEntryDTO) => row.created_at ?? '',
    }, ledgerSortCache);
    const ledgerView = { rows: ledgerRows, total: ledgerTotal };

    const onLedgerSort = (key: string) => {
      toggleSort(ledgerSortState, key);
      render();
    };
    const onInvoiceSort = (key: string) => {
      toggleSort(invoiceSortState, key);
      render();
    };

    function focusBillingCustomer() {
      const input = container.querySelector('#billing-customer-input');
      if (input instanceof HTMLElement) input.focus();
    }

    const ledgerPagination = ledgerView.total > LEDGER_PAGE
      ? renderPaginationBar({
        label: `${ledgerPage + 1} / ${Math.ceil(ledgerView.total / LEDGER_PAGE)}`,
        prevDisabled: ledgerPage === 0,
        nextDisabled: (ledgerPage + 1) * LEDGER_PAGE >= ledgerView.total,
        onPrev: () => { ledgerPage = Math.max(0, ledgerPage - 1); ledgerResource.reload(); },
        onNext: () => { ledgerPage += 1; ledgerResource.reload(); },
      })
      : null;
    const ledgerToolbar = ledgerPagination
      ? (() => {
        const wrap = el('div', { className: 'mb-4' });
        mountFilterToolbar(wrap, { pagination: ledgerPagination });
        return wrap;
      })()
      : null;

    const invoicePagination = invoicePages > 1
      ? renderPaginationBar({
        label: `${invoicePage + 1} / ${invoicePages}`,
        prevDisabled: invoicePage === 0,
        nextDisabled: invoicePage >= invoicePages - 1,
        onPrev: () => {
          invoicePage = Math.max(0, invoicePage - 1);
          invoicesResource.reload();
        },
        onNext: () => {
          invoicePage += 1;
          invoicesResource.reload();
        },
      })
      : null;
    const invoiceToolbar = invoicePagination
      ? (() => {
        const wrap = el('div', { className: 'mb-4' });
        mountFilterToolbar(wrap, { pagination: invoicePagination });
        return wrap;
      })()
      : null;

    replaceChildren(container,
      el('div', { className: 'page-header' },
        cid && !sessionScoped
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
        renderRecentCustomers({ tenant: sessionScoped && !buyerView }),
        !sessionScoped
          ? el('div', { className: 'input-group mt-4' },
            el('input', {
              id: 'billing-customer-input',
              type: 'text',
              className: 'form-input' + (customerInputError ? ' form-input--error' : ''),
              placeholder: 'customer_id (UUID)',
              value: customerInput,
              onInput: (e: Event) => {
                customerInput = eventTargetValue(e);
                customerInputError = customerInput.trim()
                  ? validateCustomerIdField(customerInput)
                  : null;
              },
            }),
            renderButton({
              label: 'Apply',
              variant: 'secondary',
              onClick: applyCustomerFilter,
            }),
          )
          : null,
        customerInputError
          ? renderAlertBanner({ variant: 'error', message: customerInputError })
          : null,
        sessionScoped && cid
          ? el('p', { className: 'text-muted text-sm mt-2' },
            'Customer: ',
            el('a', { href: `/customers/${cid}`, className: 'font-mono' }, cid),
          )
          : null,
      ),
      renderTabBar({ tabs: [
        { id: 'wallet', label: 'Wallet' },
        ...(canReadCustomers
          ? [
            { id: 'ledger', label: 'Ledger' },
            { id: 'invoices', label: 'Invoices' },
            { id: 'exports', label: 'Exports' },
          ]
          : []),
      ], active: tab, onChange: (t: any) => {
        tab = t;
        if (t === 'wallet') {
          walletResource.reload();
          if (sessionScoped) mountSelfServePanel();
        }
        else if (t === 'ledger') {
          balanceResource.reload();
          ledgerResource.reload();
        } else if (t === 'exports') {
          mountExportsPanel();
        } else invoicesResource.reload();
        render();
      } }),
      tab === 'wallet'
        ? el('div', { className: 'section-block stack' },
          !cid && !sessionScoped
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
                  formatAmountMicro(walletState.data.balance_micro ?? 0, walletState.data.currency),
                ),
              ),
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Overdraft'),
                el('div', { className: 'metric-card__value font-mono' },
                  formatAmountMicro(walletState.data.allowed_overdraft_micro ?? 0, walletState.data.currency),
                ),
              ),
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Low balance'),
                el('div', { className: 'metric-card__value font-mono' },
                  formatAmountMicro(walletState.data.low_balance_threshold_micro ?? 0, walletState.data.currency),
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
          sessionScoped ? selfServeSlot : null,
        )
        : null,
      tab === 'ledger'
        ? el('div', { className: 'section-block stack' },
          !cid
            ? renderAlertBanner({
              variant: 'info',
              message: 'Enter customer_id for ledger.',
            })
            : null,
          balanceState.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
          renderBlocking(balanceState.error),
          renderBlocking(ledgerState.error),
          balanceState.data
            ? el('div', null,
              el('div', { className: 'metric-row mb-4' },
                el('div', { className: 'metric-card' },
                  el('div', { className: 'metric-card__label' }, 'Balance (PG)'),
                  el('div', { className: 'metric-card__value font-mono' },
                    `${formatDecimalDisplay(String(balanceState.data.balance ?? ''))} ${balanceState.data.currency ?? ''}`,
                  ),
                ),
              ),
              ledgerToolbar,
              el('div', { className: 'table-wrapper table-wrapper--scroll elevation-raised' },
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
                    ledgerState.loading && ledgerView.rows.length === 0
                      ? tableSkeletonRows(5)
                      : null,
                    !ledgerState.loading && ledgerView.rows.length === 0
                      ? el('tr', null,
                        el('td', { colSpan: 5 },
                          renderEmptyState({
                            title: 'No ledger entries',
                            description: 'Transactions appear when spend or top-ups are recorded.',
                          }),
                        ),
                      )
                      : null,
                    ledgerView.rows.map((row: LedgerEntryDTO) =>
                      el('tr', null,
                        el('td', null, renderMiddleTruncateUuid(row.id)),
                        el('td', null, displayLabel(row.type)),
                        el('td', { className: 'font-mono' }, formatDecimalDisplay(String(row.amount ?? ''))),
                        el('td', null, row.campaign_id ? renderMiddleTruncateUuid(row.campaign_id) : '—'),
                        el('td', { className: 'text-muted' },
                          row.created_at ? new Date(row.created_at).toLocaleString() : '—',
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            )
            : null,
        )
        : null,
      tab === 'invoices'
        ? el('div', { className: 'section-block stack' },
          invoicesState.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
          renderBlocking(invoicesState.error),
          invoicesState.data
            ? el('div', null,
              invoiceToolbar,
              el('div', { className: 'table-wrapper table-wrapper--scroll elevation-raised' },
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
                    invoiceItems.map((inv: InvoiceDTO) =>
                      el('tr', null,
                        el('td', null,
                          el('a', {
                            href: `/billing/invoices/${inv.id}`,
                          }, renderMiddleTruncateUuid(inv.id)),
                        ),
                        el('td', null, inv.billing_month ?? '—'),
                        el('td', null, inv.status ? renderStatusBadge(inv.status, { kind: 'invoice' }) : '—'),
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
            )
            : null,
        )
        : null,
      tab === 'exports'
        ? el('div', { className: 'section-block' }, exportsSlot)
        : null,
    );
  }

  function mountSelfServePanel() {
    selfServePanelHandle?.destroy?.();
    selfServePanelHandle = mountBillingSelfServePanel(selfServeSlot, {
      customerId: customerId(),
      buyerMode: buyerView,
    });
  }

  function mountExportsPanel() {
    exportsPanelHandle?.destroy?.();
    exportsPanelHandle = mountBillingExportsPanel(exportsSlot, {
      customerId: customerId(),
      tenant: sessionScoped,
    });
  }

  const walletResource = createResource<WalletBalanceDTO>(
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

  const balanceResource = createResource<WalletBalanceDTO>(
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

  const ledgerResource = createResource<LedgerListResponse>(
    () => buildLedgerUrl(customerId(), ledgerPage),
    {
      skip: () => tab !== 'ledger' || !buildLedgerUrl(customerId(), ledgerPage),
      onUpdate: (s) => {
        if (s.error !== lastLedgerError) {
          lastLedgerError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(ledgerState, s);
        render();
      },
    },
  );

  const invoicesResource = createResource<InvoiceListResponse>(
    () => buildInvoiceUrl(customerId(), invoicePage, !sessionScoped),
    {
      skip: () => tab !== 'invoices' || !buildInvoiceUrl(customerId(), invoicePage, !sessionScoped),
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
  if (sessionScoped) mountSelfServePanel();

  return {
    destroy() {
      destroyed = true;
      exportsPanelHandle?.destroy?.();
      exportsPanelHandle = null;
      selfServePanelHandle?.destroy?.();
      selfServePanelHandle = null;
      walletResource.destroy();
      balanceResource.destroy();
      ledgerResource.destroy();
      invoicesResource.destroy();
    },
  };
}
