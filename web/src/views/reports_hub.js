import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId } from '../helpers/buyer_session.js';
import { listSavedViews, submitReportExport, pollReportJob, downloadReportExport } from '../helpers/report_api.js';
import { REPORT_CATALOG, reportHref, retiredReportAlt } from '../models/report.js';
import { probeStubReport } from '../helpers/report_api.js';
import { parallelAll } from '../helpers/request_multiplex.js';
import { renderPerfBlock } from '../helpers/perf_display.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderStatusHint } from '../ui/status_hint.js';
import { renderEmptyState } from '../ui/data_table.js';
import { renderIcon } from '../ui/icon.js';

/**
 * Render a report catalog row for the hub table.
 *
 * @param {{
 *   key: string,
 *   title: string,
 *   live?: boolean,
 *   retired?: boolean,
 *   probeStub?: boolean,
 *   exportLoading: boolean,
 *   customerId: string|null,
 *   onExport: (key: string) => void,
 * }} opts
 * @returns {HTMLTableRowElement}
 */
function renderReportRow(opts) {
  const retiredAlt = opts.retired ? retiredReportAlt(opts.key) : null;
  const href = retiredAlt?.href ?? reportHref(opts.key);
  const isLive = opts.live === true || opts.probeStub === false;
  return el('tr', { 'data-report-key': opts.key },
    el('td', null,
      el('a', {
        href,
        className: 'reports-hub__link',
      },
        renderIcon('file-text', { size: 14, className: 'reports-hub__link-icon' }),
        opts.title,
      ),
    ),
    el('td', null,
      opts.retired
        ? renderStatusBadge('archived', { label: 'Retired' })
        : isLive
          ? renderStatusBadge('ok', { kind: 'service', label: 'Live' })
          : opts.probeStub === true
            ? renderStatusBadge('planned', { kind: 'service', label: 'Planned' })
            : renderStatusBadge('pending', { kind: 'invoice', label: 'Checking…' }),
    ),
    el('td', { className: 'reports-hub__actions-cell' },
      el('div', { className: 'reports-hub__actions' },
        el('a', {
          href,
          className: 'btn btn--secondary btn--sm',
        }, retiredAlt ? retiredAlt.label : 'Open'),
        !opts.retired ? el('button', {
          type: 'button',
          className: 'btn btn--ghost btn--sm',
          disabled: opts.exportLoading || !opts.customerId,
          title: opts.customerId ? 'Export CSV' : 'Customer context required',
          onClick: (e) => {
            e.preventDefault();
            opts.onExport(opts.key);
          },
        },
          renderIcon('download', { size: 14 }),
          'Export',
        ) : null,
      ),
    ),
  );
}

/**
 * Mount the reports hub with live/stub cards, saved views, and export skeleton.
 *
 * @param {HTMLElement} container
 * @param {{ navigate: (path: string) => void }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const user = auth.getUser();
  const customerId = boundCustomerId(user);
  const exportGate = createInFlightGuard();

  const state = {
    savedViews: [],
    savedError: null,
    exportStatus: null,
    exportLoading: false,
    probing: true,
    /** @type {Record<string, boolean|null>} */
    probeStub: {},
  };

  async function loadProbes() {
    const keys = REPORT_CATALOG
      .filter((c) => !c.live && !c.retired)
      .map((c) => c.key);
    if (keys.length === 0) {
      state.probing = false;
      render();
      return;
    }
    const tasks = keys.map((key) => async () => {
      const [probe] = await to(probeStubReport(key, customerId ?? ''));
      return { key, stub: probe?.stub ?? true };
    });
    const [results] = await to(parallelAll(tasks, 3));
    if (destroyed) return;
    state.probing = false;
    if (results) {
      for (let i = 0; i < results.length; i++) {
        state.probeStub[results[i].key] = results[i].stub;
      }
    }
    render();
  }

  async function loadSaved() {
    if (!customerId) return;
    const [views, err] = await to(listSavedViews(customerId));
    if (destroyed) return;
    if (err) {
      state.savedError = err.message ?? 'Failed to load saved views';
      render();
      return;
    }
    state.savedViews = views;
    render();
  }

  let exportAbort = null;

  async function handleExport(reportKey) {
    if (!customerId || !exportGate.tryAcquire()) return;
    if (exportAbort) exportAbort.abort();
    exportAbort = new AbortController();
    const signal = exportAbort.signal;
    state.exportLoading = true;
    state.exportStatus = null;
    render();
    const preset = REPORT_DATE_PRESETS[0];
    const result = await submitReportExport({
      customerId,
      reportKey,
      from: preset.from(),
      to: preset.to(),
      signal,
    });
    if (destroyed) {
      exportGate.release();
      return;
    }
    state.exportLoading = false;
    if (result.rateLimited) {
      state.exportStatus = result.message;
      exportGate.release();
      render();
      return;
    }
    if (result.ok && result.jobId) {
      const polled = await pollReportJob(result.jobId, { signal });
      if (destroyed) {
        exportGate.release();
        return;
      }
      state.exportStatus = polled.ok
        ? `Export ${polled.status}: downloading…`
        : `Export ${polled.status}: ${polled.message}`;
      if (polled.ok) {
        const [dlErr] = await to(downloadReportExport(result.jobId, `${reportKey}.csv`));
        if (!destroyed) {
          state.exportStatus = dlErr
            ? `Export ready but download failed: ${dlErr.message}`
            : `Export downloaded: ${reportKey}.csv`;
        }
      }
    } else {
      state.exportStatus = result.stub
        ? `Export API not ready (${result.status}): ${result.message}`
        : `Export job: ${result.jobId ?? 'queued'}`;
    }
    exportGate.release();
    render();
  }

  function renderSavedPresets() {
    if (!customerId) {
      return renderEmptyState({
        title: 'No customer context',
        description: 'Bind a customer in session to load saved report presets.',
        icon: 'users',
      });
    }
    if (state.savedError) {
      return renderStatusHint({ tone: 'error', message: state.savedError });
    }
    if (state.savedViews.length === 0) {
      return renderEmptyState({
        title: 'No saved presets',
        description: 'Saved views from report pages will appear here.',
        icon: 'bookmark',
      });
    }
    return el('div', { className: 'table-wrapper' },
      el('table', { className: 'data-table' },
        el('thead', null,
          el('tr', null,
            el('th', { scope: 'col' }, 'Preset'),
            el('th', { scope: 'col' }, 'Report'),
            el('th', { scope: 'col', className: 'reports-hub__actions-cell' }, 'Actions'),
          ),
        ),
        el('tbody', null,
          state.savedViews.map((v) =>
            el('tr', null,
              el('td', null, el('span', { className: 'text-label-14' }, v.name ?? v.id)),
              el('td', null,
                el('span', { className: 'font-mono text-copy-13 text-muted' }, v.report_key ?? '—'),
              ),
              el('td', { className: 'reports-hub__actions-cell' },
                el('div', { className: 'reports-hub__actions' },
                  el('button', {
                    type: 'button',
                    className: 'btn btn--secondary btn--sm',
                    onClick: () => ctx.navigate(`/reports/${v.report_key ?? 'placements'}`),
                  }, 'Open'),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  function render() {
    if (destroyed) return;

    const liveCount = REPORT_CATALOG.filter((c) => c.live).length;

    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, 'Reports'),
          el('span', { className: 'text-label-12 text-muted' },
            `${liveCount} live · ${REPORT_CATALOG.length} total`,
          ),
        ),
        el('p', { className: 'page-header__desc' },
          'Open a report or queue a CSV export for the selected customer.',
        ),
      ),

      el('section', { className: 'settings-panel', 'data-testid': 'reports-hub' },
        el('div', { className: 'settings-panel__header' },
          el('h2', { className: 'settings-panel__title' }, 'Scheduled delivery'),
          el('p', { className: 'settings-panel__desc' },
            'Email and webhook schedules are planned. Configure recipients in Settings when available.',
          ),
        ),
      ),

      el('section', { className: 'settings-panel' },
        el('div', { className: 'settings-panel__header' },
          el('h2', { className: 'settings-panel__title' }, 'Report catalog'),
          el('p', { className: 'settings-panel__desc' },
            'Live reports are queryable today. Planned reports may return stub data until the API ships.',
          ),
        ),
        el('div', { className: 'settings-panel__body' },
          el('div', { className: 'table-wrapper table-wrapper--scroll' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Report'),
                  el('th', { scope: 'col' }, 'Status'),
                  el('th', { scope: 'col', className: 'reports-hub__actions-cell' }, 'Actions'),
                ),
              ),
              el('tbody', null,
                REPORT_CATALOG.map((card) => renderReportRow({
                  key: card.key,
                  title: card.title,
                  live: card.live,
                  retired: card.retired,
                  probeStub: card.live ? false : (card.retired ? null : state.probeStub[card.key]),
                  exportLoading: state.exportLoading,
                  customerId,
                  onExport: handleExport,
                })),
              ),
            ),
          ),
        ),
      ),

      el('section', { className: 'settings-panel' },
        el('div', { className: 'settings-panel__header' },
          el('h2', { className: 'settings-panel__title' }, 'Saved presets'),
        ),
        el('div', { className: 'settings-panel__body' }, renderSavedPresets()),
      ),

      state.exportStatus
        ? renderStatusHint({
          tone: state.exportStatus.includes('failed') ? 'error' : 'info',
          message: state.exportStatus,
        })
        : null,

      renderPerfBlock('reports-hub-perf'),
    );
  }

  loadSaved();
  loadProbes();
  render();

  return {
    destroy() {
      destroyed = true;
      exportAbort?.abort();
      exportGate.release();
    },
  };
}
