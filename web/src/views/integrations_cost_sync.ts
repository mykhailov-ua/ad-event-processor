import type { ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { tableSkeletonRows, renderEmptyTableCell } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { formatMicro } from '../helpers/money.js';
import {
  COST_SYNC_NETWORKS,
  type CostSyncNetwork,
  deleteCostSyncCredential,
  fetchCostSyncCredentials,
  fetchCostSyncHistory,
  runCostSync,
  upsertCostSyncCredential,
} from '../helpers/cost_sync_api.js';

type CostSyncCredential = {
  network: string;
  account_id?: string;
  updated_at?: string;
  [key: string]: unknown;
};

type CostSyncHistoryRow = {
  cost_date?: string;
  network?: string;
  status?: string;
  rows_imported?: number;
  total_amount_usd_micro?: number;
  trigger_source?: string;
  [key: string]: unknown;
};

type CostSyncRunBody = {
  customer_id: string;
  network: string;
  from?: string;
  to?: string;
};

/**
 * Mount Cost Sync integration admin view.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const sessionScoped = hasBoundCustomer(user?.role);
  const tenantCustomerId = boundCustomerId(user);

  const qsCustomer = new URLSearchParams(window.location.search).get('customer_id') || '';
  let customerId = sessionScoped ? tenantCustomerId : qsCustomer;
  let credentials: CostSyncCredential[] = [];
  let history: CostSyncHistoryRow[] = [];
  let loading = true;
  let error: Error | string | null = null;
  let busy = false;

  const credForm = {
    network: 'facebook',
    account_id: '',
    access_token: '',
    refresh_token: '',
    api_key: '',
  };
  const runForm = {
    network: 'facebook',
    from: '',
    to: '',
  };

  async function reload() {
    if (!isCustomerUuid(customerId)) {
      credentials = [];
      history = [];
      loading = false;
      render();
      return;
    }
    loading = true;
    error = null;
    render();
    const [creds, hist] = await Promise.all([
      to(fetchCostSyncCredentials(customerId)),
      to(fetchCostSyncHistory(customerId)),
    ]);
    if (destroyed) return;
    loading = false;
    if (creds[1]) {
      error = creds[1];
      render();
      return;
    }
    credentials = (creds[0] ?? []) as CostSyncCredential[];
    history = hist[1] ? [] : ((hist[0] ?? []) as CostSyncHistoryRow[]);
    render();
  }

  async function saveCredential() {
    if (!canWrite || !isCustomerUuid(customerId)) return;
    busy = true;
    render();
    const [, err] = await to(upsertCostSyncCredential(credForm.network, {
      customer_id: customerId,
      account_id: credForm.account_id.trim(),
      access_token: credForm.access_token,
      refresh_token: credForm.refresh_token,
      api_key: credForm.api_key,
    }));
    busy = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      pushToastMessage({ title: 'Credential save failed', message: mapServiceError(err).message });
      render();
      return;
    }
    credForm.access_token = '';
    credForm.refresh_token = '';
    credForm.api_key = '';
    pushToastMessage({ title: 'Credential saved', message: credForm.network });
    reload();
  }

  async function removeCredential(network: string) {
    if (!canWrite) return;
    const [, err] = await to(deleteCostSyncCredential(network, customerId));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Delete failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Credential removed', message: network });
    reload();
  }

  async function triggerRun() {
    if (!canWrite || !isCustomerUuid(customerId)) return;
    busy = true;
    render();
    const body: CostSyncRunBody = {
      customer_id: customerId,
      network: runForm.network,
    };
    if (runForm.from) body.from = runForm.from;
    if (runForm.to) body.to = runForm.to;
    const [, err] = await to(runCostSync(body));
    busy = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      pushToastMessage({ title: 'Sync failed', message: mapServiceError(err).message });
      render();
      return;
    }
    pushToastMessage({ title: 'Sync queued', message: 'Cost sync run accepted' });
    setTimeout(() => reload(), 1500);
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error, 'Cost Sync unavailable'));
      return;
    }

    const customerField = !sessionScoped
      ? el('label', { className: 'form-field', htmlFor: 'cost-sync-customer' },
        'Customer UUID',
        el('input', {
          id: 'cost-sync-customer',
          className: 'form-input form-input--sm font-mono',
          value: customerId,
          onInput: (e: Event) => { customerId = eventTargetValue(e).trim(); },
          onChange: () => reload(),
        }),
      )
      : el('p', { className: 'text-muted text-sm' },
        'Customer: ',
        el('span', { className: 'font-mono' }, customerId || '—'),
      );

    replaceChildren(container,
      el('section', { className: 'stack', 'data-testid': 'cost-sync-view' },
        el('div', { className: 'page-header' },
          el('h1', { className: 'page-header__title' }, 'Cost Sync'),
          el('p', { className: 'page-header__desc' },
            'Import network spend for reconciliation. Credentials are encrypted at rest. ',
            'After sync, open ',
            el('a', { href: '/reports/true-roi' }, 'True ROI'),
            ' for Ad Spend / True Profit / True ROI / True CPA.',
          ),
        ),
        customerField,
        !isCustomerUuid(customerId)
          ? el('p', { className: 'text-muted' }, 'Enter a customer UUID to manage credentials.')
          : null,
        isCustomerUuid(customerId)
          ? el('div', { className: 'section-card stack' },
            el('h2', { className: 'subsection-title' }, 'Credentials'),
            el('div', { className: 'table-wrapper' },
              el('table', { className: 'data-table' },
                el('thead', null,
                  el('tr', null,
                    el('th', null, 'Network'),
                    el('th', null, 'Account'),
                    el('th', null, 'Updated'),
                    el('th', null, ''),
                  ),
                ),
                el('tbody', null,
                  loading ? tableSkeletonRows(4) : null,
                  !loading && credentials.length === 0
                    ? el('tr', null,
                      renderEmptyTableCell(4, {
                        title: 'No credentials configured',
                        description: 'Add network credentials below to sync spend.',
                        icon: 'key',
                      }),
                    )
                    : null,
                  credentials.map((c) => el('tr', null,
                    el('td', null, c.network),
                    el('td', { className: 'font-mono text-hint' }, c.account_id || '—'),
                    el('td', null, c.updated_at ? new Date(c.updated_at).toLocaleString() : '—'),
                    el('td', null,
                      canWrite
                        ? renderButton({
                          label: 'Remove',
                          variant: 'secondary',
                          size: 'sm',
                          onClick: () => removeCredential(c.network),
                        })
                        : null,
                    ),
                  )),
                ),
              ),
            ),
            canWrite
              ? el('div', { className: 'stack mt-4' },
                el('h3', { className: 'subsection-title' }, 'Add / update credential'),
                el('div', { className: 'form-row' },
                  el('label', { className: 'form-field' },
                    'Network',
                    el('select', {
                      className: 'form-select',
                      value: credForm.network,
                      onChange: (e: Event) => { credForm.network = eventTargetValue(e); },
                    },
                      COST_SYNC_NETWORKS.map((n: CostSyncNetwork) =>
                        el('option', { value: n.id, selected: credForm.network === n.id }, n.label),
                      ),
                    ),
                  ),
                  el('label', { className: 'form-field' },
                    'Account ID',
                    el('input', {
                      className: 'form-input',
                      value: credForm.account_id,
                      onInput: (e: Event) => { credForm.account_id = eventTargetValue(e); },
                    }),
                  ),
                ),
                el('label', { className: 'form-field' },
                  'Access token',
                  el('input', {
                    className: 'form-input font-mono',
                    type: 'password',
                    autocomplete: 'off',
                    value: credForm.access_token,
                    onInput: (e: Event) => { credForm.access_token = eventTargetValue(e); },
                  }),
                ),
                el('label', { className: 'form-field' },
                  'Refresh token (optional)',
                  el('input', {
                    className: 'form-input font-mono',
                    type: 'password',
                    autocomplete: 'off',
                    value: credForm.refresh_token,
                    onInput: (e: Event) => { credForm.refresh_token = eventTargetValue(e); },
                  }),
                ),
                el('label', { className: 'form-field' },
                  'API key (optional)',
                  el('input', {
                    className: 'form-input font-mono',
                    type: 'password',
                    autocomplete: 'off',
                    value: credForm.api_key,
                    onInput: (e: Event) => { credForm.api_key = eventTargetValue(e); },
                  }),
                ),
                renderButton({
                  label: busy ? 'Saving…' : 'Save credential',
                  variant: 'primary',
                  size: 'sm',
                  loading: busy,
                  disabled: busy,
                  onClick: saveCredential,
                }),
              )
              : null,
          )
          : null,
        isCustomerUuid(customerId) && canWrite
          ? el('div', { className: 'section-card stack' },
            el('h2', { className: 'subsection-title' }, 'Manual run'),
            el('div', { className: 'form-row' },
              el('label', { className: 'form-field' },
                'Network',
                el('select', {
                  className: 'form-select',
                  value: runForm.network,
                  onChange: (e: Event) => { runForm.network = eventTargetValue(e); },
                },
                  COST_SYNC_NETWORKS.map((n: CostSyncNetwork) =>
                    el('option', { value: n.id, selected: runForm.network === n.id }, n.label),
                  ),
                ),
              ),
              el('label', { className: 'form-field' },
                'From (YYYY-MM-DD)',
                el('input', {
                  className: 'form-input',
                  placeholder: 'yesterday default',
                  value: runForm.from,
                  onInput: (e: Event) => { runForm.from = eventTargetValue(e); },
                }),
              ),
              el('label', { className: 'form-field' },
                'To (YYYY-MM-DD)',
                el('input', {
                  className: 'form-input',
                  placeholder: 'same as from',
                  value: runForm.to,
                  onInput: (e: Event) => { runForm.to = eventTargetValue(e); },
                }),
              ),
            ),
            renderButton({
              label: busy ? 'Running…' : 'Run sync',
              variant: 'primary',
              size: 'sm',
              loading: busy,
              disabled: busy,
              onClick: triggerRun,
            }),
          )
          : null,
        isCustomerUuid(customerId)
          ? el('div', { className: 'section-card stack' },
            el('h2', { className: 'subsection-title' }, 'History'),
            el('div', { className: 'table-wrapper' },
              el('table', { className: 'data-table' },
                el('thead', null,
                  el('tr', null,
                    el('th', null, 'Date'),
                    el('th', null, 'Network'),
                    el('th', null, 'Status'),
                    el('th', null, 'Rows'),
                    el('th', null, 'Amount'),
                    el('th', null, 'Trigger'),
                  ),
                ),
                el('tbody', null,
                  loading ? tableSkeletonRows(6) : null,
                  !loading && history.length === 0
                    ? el('tr', null, el('td', { colSpan: 6 }, 'No runs yet.'))
                    : null,
                  history.map((row) => el('tr', null,
                    el('td', null, row.cost_date ?? '—'),
                    el('td', null, row.network ?? '—'),
                    el('td', null, renderStatusBadge(
                      row.status === 'success' ? 'ACTIVE' : row.status === 'failed' ? 'ARCHIVED' : 'PAUSED',
                      { label: row.status },
                    )),
                    el('td', null, String(row.rows_imported ?? 0)),
                    el('td', { className: 'font-mono' }, '$' + formatMicro(row.total_amount_usd_micro ?? 0)),
                    el('td', null, row.trigger_source ?? '—'),
                  )),
                ),
              ),
            ),
          )
          : null,
      ),
    );
  }

  if (isCustomerUuid(customerId)) reload();
  else {
    loading = false;
    render();
  }

  return {
    destroy() { destroyed = true; },
  };
}
