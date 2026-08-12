import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import type { CampaignDTO, CampaignListResponse } from '../types/api/campaign.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { createResource, type ResourceState } from '../lib/fetch_resource.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { clickableRow } from '../ui/clickable_row.js';
import * as auth from '../helpers/auth.js';
import { isBuyer, can } from '../helpers/permissions.js';
import { openCampaignWizard } from '../ui/campaign_wizard.js';
import { api } from '../helpers/api_client.js';
import { attentionByCampaignId } from '../models/campaign_health.js';
import { renderCampaignHealthBadge } from '../ui/campaign_health_badge.js';
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
import { displayLabel } from '../helpers/display_labels.js';
import {
  createSortState,
  toggleSort,
  sortRows,
  sortableTh,
  tableSkeletonRows,
  renderEmptyState,
  renderPaginationBar,
} from '../ui/data_table.js';
import { renderButton, renderButtonLink } from '../ui/button.js';
import { renderCheckbox } from '../ui/checkbox.js';
import { pauseCampaign, resumeCampaign } from '../helpers/campaign_actions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { isParallelSlotError, parallelAll } from '../helpers/request_multiplex.js';
import { to } from '../lib/to.js';
import { renderCopyableUuid } from '../ui/copy_text.js';

const PAGE_SIZE = 50;
const CAMPAIGNS_EMPTY = buyerEmptyCopy('campaigns_empty');

async function runBulkCampaignAction(
  ids: string[],
  fn: (id: string) => Promise<unknown>,
): Promise<Error | null> {
  const tasks = ids.map((id) => async () => {
    const [, err] = await to(fn(id));
    return err;
  });
  const [results] = await to(parallelAll(tasks, 3));
  if (!results) return null;
  for (let i = 0; i < results.length; i++) {
    const slot = results[i];
    if (isParallelSlotError(slot)) {
      return slot.error instanceof Error ? slot.error : new Error(String(slot.error));
    }
    if (slot instanceof ConfirmCancelledError) return slot;
    if (slot) return slot;
  }
  return null;
}

/**
 * Build the campaigns list API URL for pagination and filters.
 *
 * @param {number} page
 * @param {string} customerId
 * @param {string} status
 * @returns {string}
 */
function buildUrl(page: any, customerId: any, status: any) {
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
 * Render a UUID with middle truncation that is clickable and copyable.
 *
 * @param {string} uuid
 * @returns {HTMLElement|string}
 */
function renderMiddleTruncateUuid(uuid: any) {
  return renderCopyableUuid(uuid);
}

/**
 * Mount the campaigns list view with filters, sorting, budget precision toggle, and pagination.
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
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const tenantCustomerId = boundCustomerId(user);
  const queryCustomer = ctx.query.get('customer_id')?.trim() ?? '';

  const ui = {
    page: 0,
    customerIdInput: sessionScoped ? tenantCustomerId : queryCustomer,
    statusFilter: '',
    showDetailedBudget: false,
  };

  const state: ResourceState<CampaignListResponse> = { data: null, loading: true, error: null };
  const sortState = createSortState('name', 'asc');
  const sortCache = {};
  const skipFetch = sessionScoped && !tenantCustomerId;
  /** @type {import('../models/buyer.js').BuyerPortfolioVM|null} */
  let buyerDashboard: any = null;
  /** @type {Record<string, object>|null} */
  let buyerIndexCache: any = null;
  let licenseGrace = false;
  const bulkEnabled = !buyerView && canWrite && can(user?.permissions ?? [], 'campaigns:pause');
  const selected = new Set<string>();
  let actionLoading = false;
  let actionError: string | null = null;
  const bulkGate = createInFlightGuard();

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

  async function loadLicenseMeta(): Promise<void> {
    const [res] = await Promise.all([
      api<{ license?: { state?: string } }>('/api/v1/meta').catch(() => null),
    ]);
    if (destroyed) return;
    const state = String(res?.data?.license?.state ?? '').toLowerCase();
    licenseGrace = state === 'grace';
    render();
  }

  loadLicenseMeta();

  const reloadDebounced = debounce(() => resource.reload(), 400);
  let customerFilterError: any = null;

  function effectiveCustomerId() {
    return sessionScoped ? tenantCustomerId : ui.customerIdInput.trim();
  }

  function focusCustomerInput() {
    const input = container.querySelector('#campaigns-customer-input');
    if (input instanceof HTMLElement) input.focus();
  }

  function toggleSelect(id: string, checked: boolean) {
    if (checked) selected.add(id);
    else selected.delete(id);
    render();
  }

  function toggleSelectAll(checked: boolean, visible: CampaignDTO[]) {
    selected.clear();
    if (checked) {
      for (let i = 0; i < visible.length; i++) selected.add(visible[i].id);
    }
    render();
  }

  async function bulkPause() {
    if (!bulkGate.tryAcquire()) return;
    const ids = [...selected];
    if (ids.length === 0) {
      bulkGate.release();
      return;
    }
    actionLoading = true;
    actionError = null;
    render();
    const err = await runBulkCampaignAction(ids, pauseCampaign);
    actionLoading = false;
    if (err && !(err instanceof ConfirmCancelledError)) {
      actionError = err.message ?? 'Bulk pause failed';
    } else if (!err) {
      selected.clear();
      resource.reload();
    }
    bulkGate.release();
    render();
  }

  async function bulkResume() {
    if (!bulkGate.tryAcquire()) return;
    const ids = [...selected];
    if (ids.length === 0) {
      bulkGate.release();
      return;
    }
    actionLoading = true;
    actionError = null;
    render();
    const err = await runBulkCampaignAction(ids, resumeCampaign);
    actionLoading = false;
    if (err && !(err instanceof ConfirmCancelledError)) {
      actionError = err.message ?? 'Bulk resume failed';
    } else if (!err) {
      selected.clear();
      resource.reload();
    }
    bulkGate.release();
    render();
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
    const attentionMap = buyerView
      ? attentionByCampaignId(buyerDashboard?.attention)
      : {};
    const healthCtxFor = (c: CampaignDTO) => {
      const row = buyerIndex?.[c.id];
      const portfolioRow = row && typeof row === 'object' && !Array.isArray(row) ? row : undefined;
      return {
        portfolioRow,
        attentionReason: attentionMap[c.id],
        marginBreach: !!c.margin_breach
          || !!(portfolioRow as { margin_breach?: boolean } | undefined)?.margin_breach,
        licenseGrace,
      };
    };
    const statFor = (c: CampaignDTO) => {
      const entry = buyerIndex?.[c.id];
      const row = entry && typeof entry === 'object' && !Array.isArray(entry) ? entry : null;
      return buyerCampaignStat(row);
    };
    const sortAccessors: Record<string, (c: CampaignDTO) => unknown> = buyerView
      ? {
        name: (c: CampaignDTO) => c.name ?? '',
        status: (c: CampaignDTO) => c.status ?? '',
        impressions: (c: CampaignDTO) => statFor(c).impressions,
        clicks: (c: CampaignDTO) => statFor(c).clicks,
        pacing_mode: (c: CampaignDTO) => c.pacing_mode ?? '',
      }
      : {
        name: (c: CampaignDTO) => c.name ?? '',
        status: (c: CampaignDTO) => c.status ?? '',
        budget_limit: (c: CampaignDTO) => Number(c.budget_limit ?? 0),
        current_spend: (c: CampaignDTO) => Number(c.current_spend ?? 0),
        pacing_mode: (c: CampaignDTO) => c.pacing_mode ?? '',
        customer_id: (c: CampaignDTO) => c.customer_id ?? '',
      };
    const campaigns = sortRows(state.data?.items ?? [], sortState, sortAccessors, sortCache);
    const total = state.data?.total ?? 0;
    const totalPages = Math.ceil(total / PAGE_SIZE) || 1;

    const onSort = (key: string) => {
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
        onInput: (e: Event) => {
          ui.customerIdInput = eventTargetValue(e).trim();
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
      leading: [tenantHint, customerField].filter((x): x is HTMLElement => x != null),
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
      pagination: totalPages > 1
        ? renderPaginationBar({
          label: `${ui.page + 1} / ${totalPages}`,
          prevDisabled: ui.page === 0,
          nextDisabled: ui.page >= totalPages - 1,
          onPrev: () => { ui.page = Math.max(0, ui.page - 1); resource.reload(); },
          onNext: () => { ui.page += 1; resource.reload(); },
        })
        : null,
    });

    replaceChildren(container,
      el('div', { className: 'page-header' },
        effectiveId && isCustomerUuid(effectiveId)
          ? renderBreadcrumbs([
            { label: 'Customers', href: '/customers' },
            { label: shortCustomerId(effectiveId, 14), href: `/customers/${effectiveId}` },
            { label: 'Campaigns' },
          ])
          : null,
          el('div', { className: 'page-header__row cluster--actions' },
          el('div', { className: 'flex items-center gap-3' },
            renderIcon('megaphone', { size: 22, className: 'text-muted', strokeWidth: 1.5 }),
            el('h1', { className: 'page-header__title' }, 'Campaigns'),
          ),
          buyerView
            ? renderButtonLink({
              href: '/campaigns/portfolio',
              label: 'Portfolio view',
              variant: 'secondary',
              size: 'sm',
            })
            : null,
          renderButton({
            label: ui.showDetailedBudget ? 'Precision: Micro ($00.000000)' : 'Precision: Standard ($00.00)',
            variant: 'secondary',
            size: 'sm',
            icon: 'sliders',
            title: 'Toggle budget precision (Standard $00.00 / Detailed $00.000000)',
            onClick: () => {
              ui.showDetailedBudget = !ui.showDetailedBudget;
              render();
            },
          }),
          canWrite && effectiveId && isCustomerUuid(effectiveId)
            ? renderButton({
              label: 'Create campaign',
              variant: 'primary',
              size: 'sm',
              onClick: () => {
                openCampaignWizard({
                  customerId: effectiveId,
                  onCreated: (cid) => ctx.navigate(`/campaigns/${cid}`),
                });
              },
            })
            : null,
          el('span', { className: 'text-muted text-sm' },
            state.loading ? '' : `${total} total`,
          ),
        ),
        renderRecentCustomers({ tenant: sessionScoped && !buyerView }),
      ),
      toolbarWrap,
      customerFilterError
        ? renderAlertBanner({ variant: 'error', message: customerFilterError })
        : null,
      !effectiveId && !sessionScoped && !state.loading && campaigns.length === 0
        ? renderAlertBanner({
          variant: 'info',
          message: 'Enter a customer UUID to load the campaign list.',
        })
        : null,
      effectiveId ? renderCampaignStatusLegend() : null,
      bulkEnabled && selected.size > 0
        ? el('div', { id: 'campaigns-bulk-actions', className: 'toolbar-row mb-3' },
          renderButton({
            label: `Pause selected (${selected.size})`,
            variant: 'secondary',
            size: 'sm',
            action: 'pause',
            disabled: actionLoading,
            onClick: bulkPause,
          }),
          renderButton({
            label: `Resume selected (${selected.size})`,
            variant: 'secondary',
            size: 'sm',
            action: 'resume',
            disabled: actionLoading,
            onClick: bulkResume,
          }),
        )
        : null,
      actionError ? el('p', { className: 'text-danger text-sm mb-3' }, actionError) : null,
      el('div', { className: 'table-wrapper table-wrapper--scroll elevation-raised' },
        el('table', { className: 'data-table' },
          el('thead', null,
            el('tr', null,
              bulkEnabled
                ? el('th', { scope: 'col' },
                  renderCheckbox({
                    id: 'campaigns-select-all',
                    checked: campaigns.length > 0 && campaigns.every((c: CampaignDTO) => selected.has(c.id)),
                    onChange: (checked) => toggleSelectAll(checked, campaigns),
                    label: 'Select all',
                  }),
                )
                : null,
              sortableTh('Name', 'name', sortState, onSort),
              sortableTh('Status', 'status', sortState, onSort),
              buyerView ? sortableTh('Impr. (7d)', 'impressions', sortState, onSort) : sortableTh('Budget limit', 'budget_limit', sortState, onSort),
              buyerView ? sortableTh('Clicks (7d)', 'clicks', sortState, onSort) : sortableTh('Spend', 'current_spend', sortState, onSort),
              sortableTh('Pacing', 'pacing_mode', sortState, onSort),
              el('th', { scope: 'col' }, 'Health'),
              buyerView ? null : sortableTh('Customer', 'customer_id', sortState, onSort),
            ),
          ),
          el('tbody', null,
            state.loading && campaigns.length === 0
              ? tableSkeletonRows(buyerView ? 6 : 7)
              : null,
            !state.loading && campaigns.length === 0 && effectiveId
              ? el('tr', null,
                el('td', { colSpan: bulkEnabled ? (buyerView ? 7 : 8) : (buyerView ? 6 : 7) },
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
                el('td', { colSpan: bulkEnabled ? (buyerView ? 7 : 8) : (buyerView ? 6 : 7) },
                  renderEmptyState({
                    title: 'Customer required',
                    description: 'Enter a customer UUID above to load campaigns.',
                    actionLabel: 'Focus customer field',
                    onAction: focusCustomerInput,
                  }),
                ),
              )
              : null,
            campaigns.map((c: any) =>
              clickableRow({
                id: `row-campaign-${c.id}`,
                onActivate: () => ctx.navigate(`/campaigns/${c.id}`),
                cells: [
                  bulkEnabled
                    ? el('td', { onClick: (e: Event) => e.stopPropagation() },
                      renderCheckbox({
                        checked: selected.has(c.id),
                        onChange: (checked) => toggleSelect(c.id, checked),
                        label: `Select ${c.name}`,
                      }),
                    )
                    : null,
                  el('td', null, c.name),
                  el('td', null, renderStatusBadge(c.status)),
                  buyerView
                    ? el('td', null, String(statFor(c).impressions || '—'))
                    : el('td', { className: 'font-mono' }, formatUsdDecimal(c.budget_limit ?? '0.00', { full: ui.showDetailedBudget })),
                  buyerView
                    ? el('td', null, String(statFor(c).clicks || '—'))
                    : el('td', { className: 'font-mono' }, formatUsdDecimal(c.current_spend ?? '0.00', { full: ui.showDetailedBudget })),
                  el('td', null, displayLabel(c.pacing_mode)),
                  el('td', null, renderCampaignHealthBadge(c, healthCtxFor(c))),
                  buyerView
                    ? null
                    : el('td', null,
                      c.customer_id
                        ? el('a', {
                          href: `/customers/${c.customer_id}`,
                          onClick: (e: Event) => e.stopPropagation(),
                        }, renderMiddleTruncateUuid(c.customer_id))
                        : '—',
                    ),
                ].filter((x): x is HTMLElement => x != null),
              }),
            ),
          ),
        ),
      ),
    );
  }

  const resource = createResource<CampaignListResponse>(
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
