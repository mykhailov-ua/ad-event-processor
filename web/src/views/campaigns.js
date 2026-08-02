import { el, replaceChildren } from '../lib/dom.js';
import { createResource } from '../lib/fetch_resource.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { clickableRow } from '../ui/clickable_row.js';
import * as auth from '../helpers/auth.js';
import { isTenantUser } from '../helpers/permissions.js';
import { formatUsdDecimal } from '../helpers/money.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderCampaignStatusLegend } from '../ui/status_legend.js';
import { renderAlertBanner } from '../ui/alert_banner.js';
import { debounce } from '../helpers/debounce.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { touchCustomerContext, isCustomerUuid, shortCustomerId } from '../helpers/customer_context.js';
import { renderRecentCustomers } from '../ui/recent_customers.js';
import { renderIcon } from '../ui/icon.js';
import { mount as mountChipRow } from '../ui/chip_row.js';
import {
  createSortState,
  toggleSort,
  sortRows,
  sortableTh,
  tableSkeletonRows,
  renderEmptyState,
} from '../ui/data_table.js';

const PAGE_SIZE = 50;

/**
 * @param {number} page
 * @param {string} customerId
 * @param {string} status
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
  const tenantCustomerId = user?.customer_id ?? '';
  const queryCustomer = ctx.query.get('customer_id')?.trim() ?? '';

  const ui = {
    page: 0,
    customerIdInput: tenant ? tenantCustomerId : queryCustomer,
    statusFilter: '',
  };

  const state = { data: null, loading: true, error: null };
  const sortState = createSortState('name', 'asc');
  const skipFetch = tenant && !tenantCustomerId;

  if (queryCustomer && isCustomerUuid(queryCustomer)) {
    touchCustomerContext(queryCustomer);
  }

  const reloadDebounced = debounce(() => resource.reload(), 400);
  let customerFilterError = null;

  function effectiveCustomerId() {
    return tenant ? tenantCustomerId : ui.customerIdInput.trim();
  }

  function focusCustomerInput() {
    const input = container.querySelector('#campaigns-customer-input');
    if (input instanceof HTMLElement) input.focus();
  }

  function render() {
    if (destroyed) return;

    if (skipFetch) {
      replaceChildren(container,
        renderErrorBlock(
          { status: 400, code: 'BAD_REQUEST', message: 'customer_id missing in session' },
          'Failed to load campaigns',
        ),
      );
      return;
    }

    if (state.error) {
      replaceChildren(container, renderErrorBlock(state.error, 'Failed to load campaigns'));
      return;
    }

    const effectiveId = effectiveCustomerId();
    const campaigns = sortRows(state.data?.items ?? [], sortState, {
      name: (c) => c.name ?? '',
      status: (c) => c.status ?? '',
      budget_limit: (c) => Number(c.budget_limit ?? 0),
      current_spend: (c) => Number(c.current_spend ?? 0),
      pacing_mode: (c) => c.pacing_mode ?? '',
      customer_id: (c) => c.customer_id ?? '',
    });
    const total = state.data?.total ?? 0;
    const totalPages = Math.ceil(total / PAGE_SIZE) || 1;

    const onSort = (key) => {
      toggleSort(sortState, key);
      render();
    };

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
          el('span', { className: 'text-muted', style: { fontSize: 13 } },
            state.loading ? '' : `${total} total`,
          ),
        ),
        renderRecentCustomers({ tenant }),
      ),
      el('div', { className: 'filter-bar mb-4' },
        !tenant
          ? el('div', { style: { minWidth: 240 } },
            el('input', {
              id: 'campaigns-customer-input',
              type: 'text',
              className: 'form-input',
              placeholder: 'Customer UUID…',
              value: ui.customerIdInput,
              onInput: (e) => {
                ui.customerIdInput = e.target.value.trim();
                customerFilterError = ui.customerIdInput
                  ? (isCustomerUuid(ui.customerIdInput) ? null : 'Invalid UUID format')
                  : null;
                ui.page = 0;
                if (!customerFilterError && ui.customerIdInput) reloadDebounced();
                else if (!ui.customerIdInput) resource.reload();
              },
            }),
          )
          : null,
        tenant && effectiveId
          ? el('p', { className: 'text-muted', style: { fontSize: 13 } },
            'Customer: ',
            el('a', {
              href: `/customers/${effectiveId}`,
              className: 'font-mono',
            }, effectiveId),
          )
          : null,
        el('div', { className: 'filter-chips-wrap' },
          (() => {
            const wrap = el('div');
            mountChipRow(wrap, {
              items: [
                { value: '', label: 'All Statuses' },
                { value: 'ACTIVE', label: 'Active' },
                { value: 'PAUSED', label: 'Paused' },
                { value: 'ARCHIVED', label: 'Archived' },
              ],
              selected: ui.statusFilter,
              onSelect: (v) => {
                ui.statusFilter = v;
                ui.page = 0;
                resource.reload();
              },
            });
            return wrap;
          })()
        ),
      ),
      customerFilterError
        ? renderAlertBanner({ variant: 'error', message: customerFilterError })
        : null,
      !effectiveId && !tenant
        ? renderAlertBanner({
          variant: 'info',
          message: 'Enter a customer UUID to load the campaign list.',
        })
        : null,
      effectiveId ? renderCampaignStatusLegend() : null,
      el('div', { className: 'table-wrapper table-wrapper--scroll' },
        el('table', { className: 'data-table' },
          el('thead', null,
            el('tr', null,
              sortableTh('Name', 'name', sortState, onSort),
              sortableTh('Status', 'status', sortState, onSort),
              sortableTh('Budget limit', 'budget_limit', sortState, onSort),
              sortableTh('Spend', 'current_spend', sortState, onSort),
              sortableTh('Pacing', 'pacing_mode', sortState, onSort),
              sortableTh('Customer', 'customer_id', sortState, onSort),
            ),
          ),
          el('tbody', null,
            state.loading && campaigns.length === 0
              ? tableSkeletonRows(6)
              : null,
            !state.loading && campaigns.length === 0 && effectiveId
              ? el('tr', null,
                el('td', { colSpan: 6 },
                  renderEmptyState({
                    title: 'No campaigns found',
                    description: 'Try another status filter or verify the customer UUID.',
                    actionLabel: 'Clear status filter',
                    onAction: () => {
                      ui.statusFilter = '';
                      ui.page = 0;
                      resource.reload();
                    },
                  }),
                ),
              )
              : null,
            !state.loading && campaigns.length === 0 && !effectiveId
              ? el('tr', null,
                el('td', { colSpan: 6 },
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
                  el('td', { style: { fontWeight: 500, color: 'var(--text-primary)' } }, c.name),
                  el('td', null, renderStatusBadge(c.status)),
                  el('td', { className: 'font-mono' }, formatUsdDecimal(c.budget_limit ?? '0.00')),
                  el('td', { className: 'font-mono' }, formatUsdDecimal(c.current_spend ?? '0.00')),
                  el('td', null, c.pacing_mode ?? '—'),
                  el('td', null,
                    c.customer_id
                      ? el('a', {
                        href: `/customers/${c.customer_id}`,
                        onClick: (e) => e.stopPropagation(),
                      }, renderMiddleTruncateUuid(c.customer_id))
                      : '—',
                  ),
                ],
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
