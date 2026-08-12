import type { ViewHandle } from '../lib/router_types.js';
import type { IncidentSnapshot, ShardHealthStatus } from '../types/api/index.js';
import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api, ApiError } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { isPageBlockingError, mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { formatYesNo } from '../helpers/display_labels.js';
import { renderButton } from '../ui/button.js';

type ShardHealthReport = IncidentSnapshot;

type CatchupMetricResponse = {
  points?: Array<{ ts?: string; value?: number }>;
};

type ShardsViewState = {
  report: ShardHealthReport | null;
  loading: boolean;
  error: unknown | null;
  catchupLoading: boolean;
  catchupLastSuccess: string | null;
};

/**
 * Mount the Redis shard health report view.
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  const state: ShardsViewState = {
    report: null,
    loading: true,
    error: null,
    catchupLoading: false,
    catchupLastSuccess: null,
  };

  const user = auth.getUser();
  const canCatchup = can(user?.permissions ?? [], 'shards:write');

  function shard0CatchupTarget(): ShardHealthStatus | null {
    const shards = state.report?.shards ?? [];
    const shard0 = shards.find((s) => s.shard_id === 0);
    if (!shard0) return null;
    if (shard0.config_version_synced === false) return shard0;
    return null;
  }

  async function loadCatchupMetric(): Promise<void> {
    const [res] = await to(api<CatchupMetricResponse>(
      '/api/v1/ops/dashboard/metrics?range=24h&name=ad_shard0_catchup_last_success_timestamp',
    ));
    if (destroyed) return;
    const points = res?.data?.points ?? [];
    let latest = 0;
    for (let i = 0; i < points.length; i++) {
      const value = Number(points[i]?.value ?? 0);
      if (value > latest) latest = value;
    }
    state.catchupLastSuccess = latest > 0
      ? new Date(latest * 1000).toLocaleString()
      : null;
  }

  async function runCatchup(): Promise<void> {
    if (!canCatchup || state.catchupLoading) return;
    state.catchupLoading = true;
    render();
    const [, err] = await to(apiConfirmed('/api/v1/ops/shards/0/catchup', { method: 'POST', body: '{}' }));
    state.catchupLoading = false;
    if (destroyed) return;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    pushToastMessage({ title: 'Catch-up started', message: 'Shard 0 config sync worker running.' });
    reload();
  }

  function reload(): void {
    state.loading = true;
    render();
    api<ShardHealthReport>('/api/v1/ops/shards')
      .then(({ data }) => {
        if (!destroyed) state.report = data ?? null;
      })
      .catch((err: unknown) => {
        if (destroyed) return;
        if (err instanceof ApiError && err.status === 503 && err.payload) {
          state.report = err.payload as ShardHealthReport;
        } else {
          state.error = err;
        }
      })
      .finally(() => {
        if (!destroyed) {
          state.loading = false;
          loadCatchupMetric().finally(() => {
            if (!destroyed) render();
          });
        }
      });
  }

  function render() {
    if (destroyed) return;

    if (state.loading) {
      replaceChildren(container, el('span', { className: 'text-muted' }, 'Loading…'));
      return;
    }

    if (state.error) {
      const view = mapServiceError(state.error);
      if (isPageBlockingError(view) || view.kind === 'empty') {
        replaceChildren(container, renderErrorBlock(state.error));
        return;
      }
      replaceChildren(container);
      return;
    }

    const shards = state.report?.shards ?? [];
    const catchupTarget = shard0CatchupTarget();

    replaceChildren(container,
      el('div', { className: 'page-header' },
        renderBreadcrumbs([
          { label: 'Operations', href: '/ops' },
          { label: 'Redis shards' },
        ]),
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, 'Redis shards'),
          catchupTarget && canCatchup
            ? renderButton({
              label: state.catchupLoading ? 'Running…' : 'Run catch-up',
              variant: 'danger',
              size: 'sm',
              className: 'ml-auto',
              testId: 'shard0-catchup-btn',
              loading: state.catchupLoading,
              disabled: state.catchupLoading,
              onClick: runCatchup,
            })
            : null,
        ),
      ),
      catchupTarget
        ? el('div', { className: 'stub-banner mb-4', role: 'status', 'data-testid': 'shard0-catchup-banner' },
          `Shard 0 config is out of sync (lag ${catchupTarget.config_version_lag ?? 0}). Run catch-up to reconcile pub/sub keys.`,
        )
        : null,
      state.catchupLastSuccess
        ? el('p', {
          className: 'text-muted text-sm mb-4',
          'data-testid': 'shard0-catchup-metric',
        }, `Last successful shard 0 catch-up: ${state.catchupLastSuccess}`)
        : null,
      (state.report?.errors?.length ?? 0) > 0
        ? el('div', { className: 'stub-banner mb-4' },
          `Partial: ${(state.report?.errors ?? []).map((e) => e.source).join(', ')}`,
        )
        : null,
      el('div', { className: 'table-wrapper elevation-raised' },
        el('table', { className: 'data-table' },
          el('thead', null,
            el('tr', null,
              el('th', { scope: 'col' }, 'Shard'),
              el('th', { scope: 'col' }, 'Ping OK'),
              el('th', { scope: 'col' }, 'Latency ms'),
              el('th', { scope: 'col' }, 'Config lag'),
              el('th', { scope: 'col' }, 'Synced'),
            ),
          ),
          el('tbody', null,
            shards.length === 0
              ? el('tr', null,
                el('td', {
                  colSpan: 5,
                  className: 'text-muted text-center p-6',
                }, 'No data'),
              )
              : null,
            shards.map((s: ShardHealthStatus) =>
              el('tr', {
                className: !s.ping_ok ? 'data-table__row--danger' : undefined,
              },
                el('td', null, String(s.shard_id)),
                el('td', null, formatYesNo(s.ping_ok)),
                el('td', null, s.ping_latency_ms?.toFixed(1) ?? '—'),
                el('td', null, String(s.config_version_lag ?? 0)),
                el('td', null, formatYesNo(s.config_version_synced)),
              ),
            ),
          ),
        ),
      ),
    );
  }

  reload();
  render();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
