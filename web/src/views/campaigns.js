import { el, replaceChildren } from '../lib/dom.js';
import { createResource } from '../lib/fetch_resource.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { clickableRow } from '../ui/clickable_row.js';
import * as auth from '../helpers/auth.js';
import { isBuyer } from '../helpers/permissions.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { buyerEmptyCopy } from '../models/empty_state.js';
import { fetchBuyerDashboard } from '../helpers/buyer_dashboard.js';
import { buyerCampaignStat, buyerCampaignIndex } from '../models/buyer.js';
import { formatUsdDecimal } from '../helpers/money.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderCampaignStatusLegend } from '../ui/status_legend.js';
import { renderAlertBanner } from '../ui/alert_banner.js';
import { debounce } from '../helpers/debounce.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { mountFilterToolbar } from '../ui/filter_toolbar.js';
import { touchCustomerContext, isCustomerUuid, shortCustomerId } from '../helpers/customer_context.js';
import { renderRecentCustomers } from '../ui/recent_customers.js';
import { renderIcon } from '../ui/icon.js';
import {
  createSortState,
  toggleSort,
  sortRows,
  sortableTh,
  tableSkeletonRows,
  renderEmptyState,
} from '../ui/data_table.js';

const PAGE_SIZE = 50;
const CAMPAIGNS_EMPTY = buyerEmptyCopy('campaigns_empty');

/**
 * Build the campaigns list API URL for pagination and filters.
 *
 * @param {number} page
 * @param {string} customerId
 * @param {string} status
 * @returns {string}
 */
function buildUrl(page, customerId, status) {
  const offset = page * PAGE_SIZE;
  const params = new URLSearchParams({
    limit: String(PAGE_SIZE),
    offset: String(offset),
  });
  if (customerId) params.set('customer_id', customerId);
  if (status) params.set('status', status);
  return `/api/v1/campaigns?${params.toString()}`;
}

/**
 * Render a UUID with middle truncation for table cells.
 *
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
 * Mount the campaigns list view with filters, sorting, and pagination.
 *
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams, navigate: (path: string) => void }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const buyerView = isBuyer(user?.role);
  const tenantCustomerId = boundCustomerId(user);
  const queryCustomer = ctx.query.get('customer_id')?.trim() ?? '';

  const ui = {
    page: 0,
    customerIdInput: sessionScoped ? tenantCustomerId : queryCustomer,
    statusFilter: '',
  };

  const state = { data: null, loading: true, error: null };
  const sortState = createSortState('name', 'asc');
  const sortCache = {};
  const skipFetch = sessionScoped && !tenantCustomerId;
  /** @type {import('../models/buyer.js').BuyerPortfolioVM|null} */
  let buyerDashboard = null;
  /** @type {Record<string, object>|null} */
  let buyerIndexCache = null;

  if (queryCustomer && isCustomerUuid(queryCustomer)) {
    touchCustomerContext(queryCustomer);
  }

  async function refreshBuyerDashboard() {
    if (!buyerView) return;
    const cid = effectiveCustomerId();
    if (!isCustomerUuid(cid)) {
      buyerDashboard = null;
      buyerIndexCache = null;
      return;
    }
    const dash = await fetchBuyerDashboard(cid);
    if (destroyed) return;
    buyerDashboard = dash;
    buyerIndexCache = null;
    render();
  }

  if (buyerView && !skipFetch) {
    refreshBuyerDashboard();
  }

  const reloadDebounced = debounce(() => resource.reload(), 400);
  let customerFilterError = null;

  function effectiveCustomerId() {
    return sessionScoped ? tenantCustomerId : ui.customerIdInput.trim();
  }

  function focusCustomerInput() {
    const input = container.querySelector('#campaigns-customer-input');
    if (input instanceof HTMLElement) input.focus();
  }

  function render() {
    if (destroyed) return;

    if (skipFetch) {
      const copy = buyerEmptyCopy('session_customer');
      replaceChildren(container,
        el('section', null,
          el('h1', null, 'Campaigns'),
          el('p', null, copy.title),
          el('p', null, copy.description),
        ),
      );
      return;
    }

    if (state.error) {
      replaceChildren(container, renderErrorBlock(state.error, 'Failed to load campaigns'));
      return;
    }

    const effectiveId = effectiveCustomerId();
    const buyerIndex = buyerView
      ? buyerCampaignIndex(buyerDashboard?.campaigns, buyerIndexCache ?? undefined)
      : null;
    if (buyerIndex) buyerIndexCache = buyerIndex;
    const statFor = (c) => buyerCampaignStat(buyerIndex?.[c.id]);
    const sortAccessors = buyerView
      ? {
        name: (c) => c.name ?? '',
        status: (c) => c.status ?? '',
        impressions: (c) => statFor(c).impressions,
        clicks: (c) => statFor(c).clicks,
        pacing_mode: (c) => c.pacing_mode ?? '',
      }
      : {
        name: (c) => c.name ?? '',
        status: (c) => c.status ?? '',
        budget_limit: (c) => Number(c.budget_limit ?? 0),
        current_spend: (c) => Number(c.current_spend ?? 0),
        pacing_mode: (c) => c.pacing_mode ?? '',
        customer_id: (c) => c.customer_id ?? '',
      };
    const campaigns = sortRows(state.data?.items ?? [], sortState, sortAccessors, sortCache);
    const total = state.data?.total ?? 0;
    const totalPages = Math.ceil(total / PAGE_SIZE) || 1;

    const onSort = (key) => {
      toggleSort(sortState, key);
      render();
    };

    const customerField = !sessionScoped
      ? el('input', {
        id: 'campaigns-customer-input',
        type: 'text',
        className: 'form-input form-input--sm',
        placeholder: 'Customer UUID…',
        value: ui.customerIdInput,
        onInput: (e) => {
          ui.customerIdInput = e.target.value.trim();
          customerFilterError = ui.customerIdInput
            ? (isCustomerUuid(ui.customerIdInput) ? null : 'Invalid UUID format')
            : null;
          ui.page = 0;
          if (!customerFilterError && ui.customerIdInput) {
            reloadDebounced();
            refreshBuyerDashboard();
          } else if (!ui.customerIdInput) {
            resource.reload();
            refreshBuyerDashboard();
          }
        },
      })
      : null;

    const tenantHint = sessionScoped && effectiveId && !buyerView
      ? el('p', { className: 'text-muted text-hint' },
        'Customer: ',
        el('a', {
          href: `/customers/${effectiveId}`,
          className: 'font-mono',
        }, effectiveId),
      )
      : null;

    const toolbarWrap = el('div', { className: 'mb-4' });
    mountFilterToolbar(toolbarWrap, {
      leading: [tenantHint, customerField].filter(Boolean),
      chips: [
        { value: '', label: 'All' },
        { value: 'ACTIVE', label: 'Active' },
        { value: 'PAUSED', label: 'Paused' },
        { value: 'ARCHIVED', label: 'Archived' },
      ],
      chipSelected: ui.statusFilter,
      onChipSelect: (v) => {
        ui.statusFilter = v;
        ui.page = 0;
        resource.reload();
      },
    });

    replaceChildren(container,
      el('div', { className: 'page-header' },
        effectiveId && isCustomerUuid(effectiveId)
          ? renderBreadcrumbs([
            { label: 'Customers', href: '/customers' },
            { label: shortCustomerId(effectiveId, 14), href: `/customers/${effectiveId}` },
            { label: 'Campaigns', current: true },
          ])
          : null,
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            renderIcon('megaphone', { size: 20, className: 'text-muted' }),
            el('h1', { className: 'page-header__title' }, 'Campaigns'),
          ),
          buyerView
            ? el('a', { href: '/campaigns/portfolio', className: 'btn btn--secondary btn--sm' }, 'Portfolio view')
            : null,
          el('span', { className: 'text-muted', style: { fontSize: 13 } },
            state.loading ? '' : `${total} total`,
          ),
        ),
        renderRecentCustomers({ tenant: sessionScoped && !buyerView }),
      ),
      toolbarWrap,
      customerFilterError
        ? renderAlertBanner({ variant: 'error', message: customerFilterError })
        : null,
      !effectiveId && !sessionScoped
        ? renderAlertBanner({
          variant: 'info',
          message: 'Enter a customer UUID to load the campaign list.',
        })
        : null,
      effectiveId ? renderCampaignStatusLegend() : null,
      el('div', { className: 'table-wrapper table-wrapper--scroll elevation-raised' },
        el('table', { className: 'data-table' },
          el('thead', null,
            el('tr', null,
              sortableTh('Name', 'name', sortState, onSort),
              sortableTh('Status', 'status', sortState, onSort),
              buyerView ? sortableTh('Impr. (7d)', 'impressions', sortState, onSort) : sortableTh('Budget limit', 'budget_limit', sortState, onSort),
              buyerView ? sortableTh('Clicks (7d)', 'clicks', sortState, onSort) : sortableTh('Spend', 'current_spend', sortState, onSort),
              sortableTh('Pacing', 'pacing_mode', sortState, onSort),
              buyerView ? null : sortableTh('Customer', 'customer_id', sortState, onSort),
            ),
          ),
          el('tbody', null,
            state.loading && campaigns.length === 0
              ? tableSkeletonRows(buyerView ? 5 : 6)
              : null,
            !state.loading && campaigns.length === 0 && effectiveId
              ? el('tr', null,
                el('td', { colSpan: buyerView ? 5 : 6 },
                  renderEmptyState({
                    title: CAMPAIGNS_EMPTY.title,
                    description: CAMPAIGNS_EMPTY.description,
                    actionLabel: CAMPAIGNS_EMPTY.actionLabel,
                    onAction: () => ctx.navigate(CAMPAIGNS_EMPTY.actionHref ?? '/reports/placements'),
                  }),
                ),
              )
              : null,
            !state.loading && campaigns.length === 0 && !effectiveId
              ? el('tr', null,
                el('td', { colSpan: buyerView ? 5 : 6 },
                  renderEmptyState({
                    title: 'Customer required',
                    description: 'Enter a customer UUID above to load campaigns.',
                    actionLabel: 'Focus customer field',
                    onAction: focusCustomerInput,
                  }),
                ),
              )
              : null,
            campaigns.map((c) =>
              clickableRow({
                id: `row-campaign-${c.id}`,
                onActivate: () => ctx.navigate(`/campaigns/${c.id}`),
                cells: [
                  el('td', null, c.name),
                  el('td', null, renderStatusBadge(c.status)),
                  buyerView
                    ? el('td', null, String(statFor(c).impressions || '—'))
                    : el('td', { className: 'font-mono' }, formatUsdDecimal(c.budget_limit ?? '0.00')),
                  buyerView
                    ? el('td', null, String(statFor(c).clicks || '—'))
                    : el('td', { className: 'font-mono' }, formatUsdDecimal(c.current_spend ?? '0.00')),
                  el('td', null, c.pacing_mode ?? '—'),
                  buyerView
                    ? null
                    : el('td', null,
                      c.customer_id
                        ? el('a', {
                          href: `/customers/${c.customer_id}`,
                          onClick: (e) => e.stopPropagation(),
                        }, renderMiddleTruncateUuid(c.customer_id))
                        : '—',
                    ),
                ].filter(Boolean),
              }),
            ),
          ),
        ),
      ),
      totalPages > 1
        ? el('div', {
          className: 'flex items-center gap-2 mt-4',
          style: { justifyContent: 'flex-end' },
        },
          el('button', {
            id: 'campaigns-prev-btn',
            className: 'btn btn--secondary btn--sm',
            disabled: ui.page === 0,
            onClick: () => { ui.page = Math.max(0, ui.page - 1); resource.reload(); },
          }, 'Prev'),
          el('span', { className: 'text-muted', style: { fontSize: 12 } },
            `${ui.page + 1} / ${totalPages}`,
          ),
          el('button', {
            id: 'campaigns-next-btn',
            className: 'btn btn--secondary btn--sm',
            disabled: ui.page >= totalPages - 1,
            onClick: () => { ui.page += 1; resource.reload(); },
          }, 'Next'),
        )
        : null,
    );
  }

  const resource = createResource(
    () => buildUrl(ui.page, effectiveCustomerId(), ui.statusFilter),
    {
      skip: () => skipFetch,
      onUpdate: (s) => {
        Object.assign(state, s);
        const cid = effectiveCustomerId();
        if (!s.loading && !s.error && isCustomerUuid(cid)) {
          touchCustomerContext(cid);
        }
        render();
      },
    },
  );

  render();

  return {
    destroy() {
      destroyed = true;
      resource.destroy();
    },
  };
}
