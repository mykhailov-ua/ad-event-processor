import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId, hasBoundCustomer } from '../helpers/buyer_session.js';
import { fetchBuyerDashboard, invalidateBuyerDashboard } from '../helpers/buyer_dashboard.js';
import { visiblePortfolioRows } from '../models/buyer.js';
import { pauseCampaign, resumeCampaign } from '../helpers/campaign_actions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { parallelAll } from '../helpers/request_multiplex.js';
import { renderPerfBlock } from '../helpers/perf_display.js';
import { renderCheckbox } from '../ui/checkbox.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { clickableRow } from '../ui/clickable_row.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { buyerEmptyCopy } from '../models/empty_state.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { displayLabel } from '../helpers/display_labels.js';

/**
 * Run a bulk campaign mutation with bounded concurrency.
 *
 * @param {string[]} ids
 * @param {(id: string) => Promise<void>} fn
 * @returns {Promise<Error|null>}
 */
async function runBulk(ids, fn) {
  const tasks = ids.map((id) => async () => {
    const [, err] = await to(fn(id));
    return err ?? null;
  });
  const results = await parallelAll(tasks, 3);
  for (let i = 0; i < results.length; i++) {
    const err = results[i];
    if (err instanceof ConfirmCancelledError) return err;
    if (err) return err;
  }
  return null;
}

/**
 * Mount buyer portfolio view with drift sort and bulk pause/resume.
 *
 * @param {HTMLElement} container
 * @param {{ navigate: (path: string) => void }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const user = auth.getUser();
  const customerId = boundCustomerId(user);
  const sessionScoped = hasBoundCustomer(user?.role);
  const rowCache = { portfolio: null, filter: '', rows: null };
  const bulkGate = createInFlightGuard();

  const state = {
    loading: true,
    error: null,
    portfolio: null,
    statusFilter: '',
    selected: new Set(),
    actionError: null,
    actionLoading: false,
  };

  async function load() {
    state.loading = true;
    state.error = null;
    rowCache.rows = null;
    render();
    const [data, err] = await to(fetchBuyerDashboard(customerId));
    if (destroyed) return;
    if (err) {
      state.error = err.message ?? 'Failed to load portfolio';
      state.loading = false;
      render();
      return;
    }
    state.portfolio = data;
    state.loading = false;
    rowCache.rows = null;
    render();
  }

  function rowsForRender() {
    return visiblePortfolioRows(state.portfolio, state.statusFilter, rowCache);
  }

  function toggleSelect(id, checked) {
    if (checked) state.selected.add(id);
    else state.selected.delete(id);
    renderSelection();
  }

  function toggleSelectAll(checked) {
    state.selected.clear();
    if (checked) {
      const rows = rowsForRender();
      for (let i = 0; i < rows.length; i++) state.selected.add(rows[i].row.id);
    }
    renderSelection();
  }

  function renderSelection() {
    const bulk = container.querySelector('#portfolio-bulk-actions');
    if (bulk) {
      bulk.querySelectorAll('button').forEach((btn) => {
        btn.disabled = state.actionLoading;
      });
      const pauseBtn = bulk.querySelector('[data-action="pause"]');
      const resumeBtn = bulk.querySelector('[data-action="resume"]');
      if (pauseBtn) pauseBtn.textContent = `Pause selected (${state.selected.size})`;
      if (resumeBtn) resumeBtn.textContent = `Resume selected (${state.selected.size})`;
    }
    const tbody = container.querySelector('#portfolio-tbody');
    if (!tbody) return;
    const inputs = tbody.querySelectorAll('input[type="checkbox"]');
    inputs.forEach((input) => {
      const rowId = input.closest('tr')?.id?.replace('portfolio-row-', '');
      if (rowId) input.checked = state.selected.has(rowId);
    });
    const selectAll = container.querySelector('#portfolio-select-all');
    if (selectAll instanceof HTMLInputElement) {
      const rows = rowsForRender();
      selectAll.checked = rows.length > 0 && rows.every((r) => state.selected.has(r.row.id));
    }
  }

  async function bulkPause() {
    if (!bulkGate.tryAcquire()) return;
    const ids = [...state.selected];
    if (ids.length === 0) {
      bulkGate.release();
      return;
    }
    state.actionLoading = true;
    state.actionError = null;
    renderSelection();
    const err = await runBulk(ids, pauseCampaign);
    state.actionLoading = false;
    if (err && !(err instanceof ConfirmCancelledError)) {
      state.actionError = err.message ?? 'Bulk pause failed';
    }
    state.selected.clear();
    invalidateBuyerDashboard(customerId);
    bulkGate.release();
    await load();
  }

  async function bulkResume() {
    if (!bulkGate.tryAcquire()) return;
    const ids = [...state.selected];
    if (ids.length === 0) {
      bulkGate.release();
      return;
    }
    state.actionLoading = true;
    state.actionError = null;
    renderSelection();
    const err = await runBulk(ids, resumeCampaign);
    state.actionLoading = false;
    if (err && !(err instanceof ConfirmCancelledError)) {
      state.actionError = err.message ?? 'Bulk resume failed';
    }
    state.selected.clear();
    invalidateBuyerDashboard(customerId);
    bulkGate.release();
    await load();
  }

  function render() {
    if (destroyed) return;

    if (!sessionScoped || !customerId) {
      const copy = buyerEmptyCopy('session_customer');
      replaceChildren(container,
        el('div', { className: 'page-header' },
          el('h1', { className: 'page-header__title' }, 'Portfolio'),
          el('p', { className: 'page-header__desc' }, copy.title),
          el('p', { className: 'text-muted text-sm' }, copy.description),
        ),
      );
      return;
    }

    if (state.error) {
      replaceChildren(container, renderErrorBlock(state.error, 'Portfolio unavailable'));
      return;
    }

    const rows = rowsForRender();
    const allSelected = rows.length > 0 && rows.every((r) => state.selected.has(r.row.id));
    const perfBlock = renderPerfBlock('portfolio-perf-metrics');

    replaceChildren(container,
      el('section', { className: 'stack', 'data-testid': 'buyer-portfolio-view' },
        el('div', { className: 'page-header' },
          el('h1', { className: 'page-header__title' }, 'Portfolio'),
          el('p', { className: 'page-header__desc' },
            'Sorted by pacing drift. ',
            el('a', { href: '/campaigns' }, 'Table view'),
          ),
        ),
        state.loading ? el('p', { className: 'loading-hint' }, 'Loading portfolio…') : null,
        !state.loading ? el('div', { className: 'filter-row' },
          el('label', { className: 'form-label form-label--flush' }, 'Status filter'),
          el('select', {
            className: 'form-select min-w-40',
            onChange: (e) => {
              state.statusFilter = e.target.value;
              rowCache.rows = null;
              render();
            },
          },
            el('option', { value: '', selected: !state.statusFilter }, 'All'),
            el('option', { value: 'ACTIVE', selected: state.statusFilter === 'ACTIVE' }, 'Active'),
            el('option', { value: 'PAUSED', selected: state.statusFilter === 'PAUSED' }, 'Paused'),
            el('option', { value: 'ARCHIVED', selected: state.statusFilter === 'ARCHIVED' }, 'Archived'),
          ),
        ) : null,
        state.selected.size > 0 ? el('div', { id: 'portfolio-bulk-actions', className: 'toolbar-row' },
          el('button', {
            type: 'button',
            className: 'btn btn--secondary btn--sm',
            'data-action': 'pause',
            disabled: state.actionLoading,
            onClick: bulkPause,
          }, `Pause selected (${state.selected.size})`),
          el('button', {
            type: 'button',
            className: 'btn btn--secondary btn--sm',
            'data-action': 'resume',
            disabled: state.actionLoading,
            onClick: bulkResume,
          }, `Resume selected (${state.selected.size})`),
        ) : null,
        state.actionError ? el('p', { className: 'text-danger text-sm' }, state.actionError) : null,
        !state.loading ? el('div', { className: 'table-wrapper table-wrapper--scroll elevation-raised' },
          el('table', { className: 'data-table' },
          el('thead', null,
            el('tr', null,
              el('th', null,
                renderCheckbox({
                  id: 'portfolio-select-all',
                  checked: allSelected,
                  onChange: toggleSelectAll,
                  label: 'Select all',
                }),
              ),
              el('th', null, 'Campaign'),
              el('th', null, 'Status'),
              el('th', null, 'Drift %'),
              el('th', null, 'Util %'),
              el('th', null, 'Risk'),
              el('th', null, 'Impr. 7d'),
              el('th', null, 'Clicks 7d'),
              el('th', null, 'Pacing'),
            ),
          ),
          el('tbody', { id: 'portfolio-tbody' },
            rows.map(({ row: c, driftScore }) =>
              clickableRow({
                id: `portfolio-row-${c.id}`,
                onActivate: () => ctx.navigate(`/campaigns/${c.id}`),
                cells: [
                  el('td', { onClick: (e) => e.stopPropagation() },
                    renderCheckbox({
                      checked: state.selected.has(c.id),
                      onChange: (checked) => toggleSelect(c.id, checked),
                    }),
                  ),
                  el('td', null, c.name ?? c.id),
                  el('td', null, renderStatusBadge(c.status)),
                  el('td', null, String(c.pacing_drift_pct != null ? c.pacing_drift_pct.toFixed(0) : driftScore)),
                  el('td', null, c.utilization_pct != null ? `${c.utilization_pct.toFixed(0)}%` : '—'),
                  el('td', null, c.overspend_risk ? renderStatusBadge('warning', { label: 'risk' }) : '—'),
                  el('td', null, String(c.impressions_7d ?? 0)),
                  el('td', null, String(c.clicks_7d ?? 0)),
                  el('td', null, displayLabel(c.pacing_mode)),
                ],
              }),
            ),
          ),
        ),
        ) : null,
        perfBlock,
      ),
    );
  }

  load();
  render();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
