import { el, replaceChildren } from '../lib/dom.js';
import { api, ApiError } from '../helpers/api_client.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { isPageBlockingError, mapServiceError } from '../helpers/service_error.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';

/**
 * @param {HTMLElement} container
 */
export function mount(container) {
  let destroyed = false;
  const state = { report: null, loading: true, error: null };

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

    replaceChildren(container,
      el('div', { className: 'page-header' },
        renderBreadcrumbs([
          { label: 'Operations', href: '/ops' },
          { label: 'Redis shards', current: true },
        ]),
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, 'Redis shards'),
        ),
      ),
      state.report?.errors?.length > 0
        ? el('div', { className: 'stub-banner mb-4' },
          `Partial: ${state.report.errors.map((e) => e.source).join(', ')}`,
        )
        : null,
      el('div', { className: 'table-wrapper' },
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
                  className: 'text-muted',
                  style: { textAlign: 'center', padding: 24 },
                }, 'No data'),
              )
              : null,
            shards.map((s) =>
              el('tr', {
                className: !s.ping_ok ? 'data-table__row--danger' : undefined,
              },
                el('td', null, String(s.shard_id)),
                el('td', null, s.ping_ok ? 'yes' : 'no'),
                el('td', null, s.ping_latency_ms?.toFixed(1) ?? '—'),
                el('td', null, String(s.config_version_lag ?? 0)),
                el('td', null, s.config_version_synced ? 'yes' : 'no'),
              ),
            ),
          ),
        ),
      ),
    );
  }

  api('/api/v1/ops/shards')
    .then(({ data }) => {
      if (!destroyed) state.report = data;
    })
    .catch((err) => {
      if (destroyed) return;
      if (err instanceof ApiError && err.status === 503 && err.payload) {
        state.report = err.payload;
      } else {
        state.error = err;
      }
    })
    .finally(() => {
      if (!destroyed) {
        state.loading = false;
        render();
      }
    });

  render();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
