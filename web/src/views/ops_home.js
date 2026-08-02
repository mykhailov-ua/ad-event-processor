import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api, ApiError } from '../helpers/api_client.js';
import { parallelAll } from '../helpers/request_multiplex.js';
import { apiBlob } from '../helpers/api_blob.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderTabBar } from '../ui/tab_bar.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderSelect } from '../ui/select.js';
import { renderAlertBanner } from '../ui/alert_banner.js';
import { renderDoctorPanel } from '../ui/doctor_panel.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderIcon } from '../ui/icon.js';

/**
 * @param {HTMLElement} container
 */
export function mount(container) {
  let destroyed = false;
  let tab = 'overview';
  let outboxStatus = '';
  let outboxItems = [];
  let outboxCursor = '';
  let outboxLoading = false;

  const state = {
    loading: true,
    doctor: null,
    incidents: null,
    summary: null,
    partialErrors: [],
    partialDismissed: false,
    blockError: null,
  };

  const user = auth.getUser();
  const canBundle = can(user?.permissions ?? [], 'ops:write');

  function render() {
    if (destroyed) return;

    if (state.blockError) {
      replaceChildren(container, renderErrorBlock(state.blockError));
      return;
    }

    const shardSnippet = state.incidents?.shards ?? [];

    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, 'Operations'),
          state.doctor
            ? renderStatusBadge(state.doctor.overall, {
              kind: 'service',
              label: `doctor: ${state.doctor.overall}`,
            })
            : null,
          canBundle
            ? el('button', {
              type: 'button',
              className: 'btn btn--secondary btn--sm',
              style: { marginLeft: 'auto' },
              onClick: downloadBundle,
            },
              renderIcon('download', { size: 14 }),
              'Support bundle',
            )
            : null,
        ),
      ),
      state.partialErrors.length > 0 && !state.partialDismissed
        ? renderAlertBanner({
          variant: 'warning',
          message: `Partial source errors: ${state.partialErrors.map((e) => `${e.source ?? '?'} (${e.code ?? 'err'})`).join('; ')}`,
          dismissKey: 'ops.partial',
          onDismiss: () => {
            state.partialDismissed = true;
            render();
          },
        })
        : null,
      state.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
      !state.loading && state.summary
        ? el('div', { className: 'grid-stats' },
          el('div', { className: 'metric-card' },
            el('div', { className: 'flex items-center justify-between mb-2' },
              el('div', { className: 'metric-card__label' }, 'Outbox pending'),
              renderIcon('activity', { size: 16, className: 'text-muted' }),
            ),
            el('div', { className: 'metric-card__value' }, String(state.summary.outbox_pending)),
          ),
          el('div', { className: 'metric-card' },
            el('div', { className: 'flex items-center justify-between mb-2' },
              el('div', { className: 'metric-card__label' }, 'RPS estimate'),
              renderIcon('zap', { size: 16, className: 'text-muted' }),
            ),
            el('div', { className: 'metric-card__value' },
              state.summary.rps_estimate?.toFixed(1) ?? '—',
            ),
          ),
          el('div', { className: 'metric-card' },
            el('div', { className: 'flex items-center justify-between mb-2' },
              el('div', { className: 'metric-card__label' }, 'Emergency breaker'),
              renderIcon('shield', { size: 16, className: 'text-muted' }),
            ),
            el('div', { className: 'metric-card__value' }, state.summary.emergency_breaker ?? '—'),
          ),
          el('div', { className: 'metric-card' },
            el('div', { className: 'flex items-center justify-between mb-2' },
              el('div', { className: 'metric-card__label' }, 'Drift alert'),
              renderIcon('alert-triangle', { size: 16, className: 'text-muted' }),
            ),
            el('div', { className: 'metric-card__value' }, state.summary.drift_alert ? 'yes' : 'no'),
          ),
        )
        : null,
      renderTabBar({ tabs: [
        { id: 'overview', label: 'Overview' },
        { id: 'outbox', label: 'Outbox' },
      ], active: tab, onChange: (t) => {
        tab = t;
        if (t === 'outbox') loadOutbox();
        render();
      } }),
      tab === 'overview' && !state.loading && state.summary
        ? renderDoctorPanel({
          doctor: state.doctor,
          services: state.summary.services,
          loading: false,
        })
        : null,
      tab === 'overview' && shardSnippet.length > 0
        ? el('div', { style: { marginTop: 24 } },
          el('div', { className: 'flex items-center gap-2 mb-4' },
            el('h2', { style: { fontSize: 14, fontWeight: 600 } }, 'Shards'),
            el('a', {
              href: '/ops/shards',
              className: 'text-muted',
              style: { fontSize: 12 },
            }, 'All shards →'),
          ),
          el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Shard'),
                  el('th', { scope: 'col' }, 'Ping'),
                  el('th', { scope: 'col' }, 'Lag'),
                ),
              ),
              el('tbody', null,
                shardSnippet.slice(0, 8).map((s) =>
                  el('tr', {
                    className: !s.ping_ok ? 'data-table__row--danger' : undefined,
                  },
                    el('td', null, String(s.shard_id)),
                    el('td', null, s.ping_ok ? 'ok' : (s.ping_error ?? 'fail')),
                    el('td', null, String(s.config_version_lag ?? 0)),
                  ),
                ),
              ),
            ),
          ),
        )
        : null,
      tab === 'outbox'
        ? el('div', { style: { marginTop: 24 } },
          el('div', { className: 'filter-row mb-4' },
            el('label', { className: 'form-label', style: { margin: 0 } }, 'Status'),
            renderSelect({
              value: outboxStatus,
              options: [
                { value: '', label: 'All' },
                { value: 'pending', label: 'pending' },
                { value: 'processed', label: 'processed' },
                { value: 'failed', label: 'failed' },
              ],
              style: { width: '160px' },
              'aria-label': 'Outbox status',
              onChange: (v) => {
                outboxStatus = v;
                loadOutbox();
              },
            }),
          ),
          outboxLoading && outboxItems.length === 0
            ? el('span', { className: 'text-muted' }, 'Loading…')
            : null,
          el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'ID'),
                  el('th', { scope: 'col' }, 'Type'),
                  el('th', { scope: 'col' }, 'Status'),
                  el('th', { scope: 'col' }, 'Created'),
                ),
              ),
              el('tbody', null,
                outboxItems.map((row) =>
                  el('tr', null,
                    el('td', { className: 'font-mono' }, row.id),
                    el('td', null, row.event_type),
                    el('td', null, row.status),
                    el('td', { className: 'text-muted' },
                      row.created_at ? new Date(row.created_at).toLocaleString() : '—',
                    ),
                  ),
                ),
              ),
            ),
          ),
          outboxCursor
            ? el('button', {
              type: 'button',
              className: 'btn btn--secondary btn--sm mt-4',
              disabled: outboxLoading,
              onClick: loadMoreOutbox,
            }, 'Load more')
            : null,
        )
        : null,
    );
  }

  async function loadOutbox() {
    outboxLoading = true;
    outboxItems = [];
    outboxCursor = '';
    render();
    const params = new URLSearchParams();
    if (outboxStatus) params.set('status', outboxStatus);
    const [outboxRes, outboxErr] = await to(api(`/api/v1/ops/outbox?${params.toString()}`));
    if (destroyed) return;
    if (outboxErr) {
      const view = mapServiceError(outboxErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    } else {
      outboxItems = outboxRes?.data?.items ?? [];
      outboxCursor = outboxRes?.data?.next_cursor ?? '';
    }
    outboxLoading = false;
    if (!destroyed) render();
  }

  async function loadMoreOutbox() {
    if (!outboxCursor) return;
    outboxLoading = true;
    render();
    const params = new URLSearchParams({ cursor: outboxCursor });
    if (outboxStatus) params.set('status', outboxStatus);
    const [outboxRes, outboxErr] = await to(api(`/api/v1/ops/outbox?${params.toString()}`));
    if (destroyed) return;
    if (outboxErr) {
      const view = mapServiceError(outboxErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    } else {
      outboxItems = [...outboxItems, ...(outboxRes?.data?.items ?? [])];
      outboxCursor = outboxRes?.data?.next_cursor ?? '';
    }
    outboxLoading = false;
    if (!destroyed) render();
  }

  async function downloadBundle() {
    const [blob, blobErr] = await to(apiBlob('/api/v1/ops/support/bundle', { method: 'POST' }));
    if (blobErr) {
      const view = mapServiceError(blobErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'espx-support-bundle.tar.gz';
    a.click();
    URL.revokeObjectURL(url);
  }

  async function loadOpsData() {
    state.loading = true;
    state.blockError = null;

    const [results, err] = await to(parallelAll([
      () => api('/api/v1/ops/doctor'),
      () => api('/api/v1/ops/incidents').catch((e) => ({ error: e })),
      () => api('/api/v1/ops/dashboard/summary'),
    ], 3));

    if (destroyed) return;

    if (err) {
      state.blockError = err;
      state.loading = false;
      render();
      return;
    }

    const [docRes, incRes, sumRes] = results;
    if (docRes?.data) state.doctor = docRes.data;
    if (sumRes?.data) state.summary = sumRes.data;

    const errors = [];
    if (incRes?.error) {
      const incErr = incRes.error;
      if (incErr instanceof ApiError && incErr.payload) {
        state.incidents = incErr.payload;
        if (incErr.payload.errors?.length) errors.push(...incErr.payload.errors);
      } else {
        const view = mapServiceError(incErr);
        if (view.kind === 'page' || view.kind === 'unavailable') {
          state.blockError = incErr;
        } else {
          pushToastMessage({ title: view.title, message: view.message, code: view.code });
        }
      }
    } else if (incRes?.data) {
      state.incidents = incRes.data;
      if (incRes.data.errors?.length) errors.push(...incRes.data.errors);
    }
    state.partialErrors = errors;

    state.loading = false;
    render();
  }

  render();
  loadOpsData();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
