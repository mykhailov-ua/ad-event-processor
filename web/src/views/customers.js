import { el, replaceChildren } from '../lib/dom.js';
import { createResource } from '../lib/fetch_resource.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { clickableRow } from '../ui/clickable_row.js';
import { formatUsdDecimal } from '../helpers/money.js';
import * as auth from '../helpers/auth.js';
import { isTenantUser } from '../helpers/permissions.js';
import { touchCustomerContext } from '../helpers/customer_context.js';
import { renderRecentCustomers } from '../ui/recent_customers.js';
import { renderIcon } from '../ui/icon.js';
import { mountFilterToolbar } from '../ui/filter_toolbar.js';
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
 * Build the customers list API URL for a page offset.
 *
 * @param {number} page
 * @returns {string}
 */
function buildUrl(page) {
  const offset = page * PAGE_SIZE;
  return `/api/v1/customers?limit=${PAGE_SIZE}&offset=${offset}`;
}

/**
 * Format a customer balance field for display.
 *
 * @param {unknown} bal
 * @returns {string}
 */
function formatBalance(bal) {
  if (!bal) return '—';
  return formatUsdDecimal(String(bal));
}

/**
 * Mount the customers list view with search, sorting, and pagination.
 *
 * @param {HTMLElement} container
 * @param {{ navigate: (path: string) => void }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const user = auth.getUser();
  const tenant = isTenantUser(user?.role);
  const tenantId = user?.customer_id;

  if (tenant && tenantId) {
    window.history.replaceState(null, '', `/customers/${tenantId}`);
    ctx.navigate(`/customers/${tenantId}`);
    replaceChildren(container, el('span', { className: 'text-muted' }, 'Redirecting…'));
    return { destroy() { destroyed = true; } };
  }

  const state = { data: null, loading: true, error: null, page: 0, searchQuery: '' };
  const sortState = createSortState('name', 'asc');

  function render() {
    if (destroyed) return;
    if (state.error) {
      replaceChildren(container, renderErrorBlock(state.error, 'Failed to load customers'));
      return;
    }

    const customers = sortRows(state.data?.items ?? [], sortState, {
      name: (c) => c.name ?? '',
      balance: (c) => Number(c.balance ?? 0),
      currency: (c) => c.currency ?? '',
      active_campaigns: (c) => Number(c.active_campaigns ?? 0),
      created_at: (c) => c.created_at ?? '',
    }).filter((c) => {
      const q = state.searchQuery.trim().toLowerCase();
      if (!q) return true;
      return (c.name ?? '').toLowerCase().includes(q)
        || (c.id ?? '').toLowerCase().includes(q);
    });
    const total = state.data?.total ?? 0;
    const totalPages = Math.ceil(total / PAGE_SIZE);

    const onSort = (key) => {
      toggleSort(sortState, key);
      render();
    };

    const toolbarWrap = el('div', { className: 'mb-4' });
    mountFilterToolbar(toolbarWrap, {
      search: true,
      searchPlaceholder: 'Filter by name or ID…',
      searchValue: state.searchQuery,
      onSearch: (value) => {
        state.searchQuery = value;
        render();
      },
    });

    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            renderIcon('users', { size: 20, className: 'text-muted' }),
            el('h1', { className: 'page-header__title' }, 'Customers'),
          ),
          el('span', { className: 'text-muted text-sm' },
            state.loading ? '' : `${total} total`,
          ),
        ),
        renderRecentCustomers({ tenant }),
      ),
      toolbarWrap,
      el('div', { className: 'table-wrapper table-wrapper--scroll elevation-raised' },
        el('table', { className: 'data-table' },
          el('thead', null,
            el('tr', null,
              sortableTh('Name', 'name', sortState, onSort),
              sortableTh('Balance', 'balance', sortState, onSort),
              sortableTh('Currency', 'currency', sortState, onSort),
              sortableTh('Active Campaigns', 'active_campaigns', sortState, onSort),
              sortableTh('Created', 'created_at', sortState, onSort),
            ),
          ),
          el('tbody', null,
            state.loading && customers.length === 0
              ? tableSkeletonRows(5)
              : null,
            !state.loading && customers.length === 0
              ? el('tr', null,
                el('td', { colSpan: 5 },
                  renderEmptyState({
                    title: 'No customers found',
                    description: 'Customers appear after they are created in the system.',
                    actionLabel: 'Open billing',
                    onAction: () => ctx.navigate('/billing'),
                  }),
                ),
              )
              : null,
            customers.map((c) =>
              clickableRow({
                id: `row-customer-${c.id}`,
                onActivate: () => {
                  touchCustomerContext(c.id);
                  ctx.navigate(`/customers/${c.id}`);
                },
                cells: [
                  el('td', { className: 'font-medium' }, c.name),
                  el('td', { className: 'font-mono' }, formatBalance(c.balance)),
                  el('td', null, c.currency ?? 'USD'),
                  el('td', null, String(c.active_campaigns ?? 0)),
                  el('td', { className: 'text-muted' },
                    c.created_at ? new Date(c.created_at).toLocaleDateString() : '-',
                  ),
                ],
              }),
            ),
          ),
        ),
      ),
      totalPages > 1
        ? el('div', { className: 'pagination-bar' },
          el('button', {
            id: 'customers-prev-btn',
            className: 'btn btn--secondary btn--sm',
            disabled: state.page === 0,
            onClick: () => { state.page = Math.max(0, state.page - 1); resource.reload(); },
          }, 'Prev'),
          el('span', { className: 'text-muted text-xs' },
            `${state.page + 1} / ${totalPages}`,
          ),
          el('button', {
            id: 'customers-next-btn',
            className: 'btn btn--secondary btn--sm',
            disabled: state.page >= totalPages - 1,
            onClick: () => { state.page += 1; resource.reload(); },
          }, 'Next'),
        )
        : null,
    );
  }

  const resource = createResource(
    () => buildUrl(state.page),
    {
      onUpdate: (s) => {
        Object.assign(state, s);
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
