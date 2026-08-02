import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId } from '../helpers/buyer_session.js';
import { listSavedViews, submitReportExport, pollReportJob, downloadReportExport } from '../helpers/report_api.js';
import { REPORT_CATALOG, reportHref } from '../models/report.js';
import { renderPerfBlock } from '../helpers/perf_display.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { createInFlightGuard } from '../lib/async_guard.js';

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
  };

  async function loadSaved() {
    if (!customerId) return;
    const [views, err] = await to(listSavedViews(customerId));
    if (destroyed) return;
    if (err) {
      state.savedError = err.message ?? 'Failed to load saved views';
      return;
    }
    state.savedViews = views;
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
    });
    if (destroyed) {
      exportGate.release();
      return;
    }
    state.exportLoading = false;
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

  function render() {
    if (destroyed) return;

    const cards = REPORT_CATALOG.map((card) =>
      el('li', null,
        el('a', { href: reportHref(card.key), 'data-report-key': card.key }, card.title),
        card.live ? ' (live)' : ' (planned)',
        ' ',
        el('button', {
          type: 'button',
          disabled: state.exportLoading || !customerId,
          onClick: (e) => {
            e.preventDefault();
            handleExport(card.key);
          },
        }, 'Export CSV'),
      ),
    );

    replaceChildren(container,
      el('section', { 'data-testid': 'reports-hub' },
        el('h1', null, 'Reports'),
        el('p', null, 'Choose a report or export a CSV job when the API is ready.'),
        el('h2', null, 'Scheduled delivery'),
        el('p', { className: 'text-muted' },
          'Email/webhook scheduled reports — API planned. Configure recipients in Settings when available.',
        ),
        el('h2', null, 'Reports'),
        el('ul', null, ...cards),
        el('h2', null, 'Saved presets'),
        !customerId
          ? el('p', null, 'Customer context required to load saved views.')
          : null,
        state.savedError ? el('p', null, state.savedError) : null,
        state.savedViews.length === 0 && customerId && !state.savedError
          ? el('p', null, 'No saved presets yet.')
          : null,
        state.savedViews.length > 0
          ? el('ul', null,
            state.savedViews.map((v) =>
              el('li', null,
                el('strong', null, v.name ?? v.id),
                ` — ${v.report_key ?? ''}`,
                ' ',
                el('button', {
                  type: 'button',
                  onClick: () => ctx.navigate(`/reports/${v.report_key ?? 'placements'}`),
                }, 'Open'),
              ),
            ),
          )
          : null,
        state.exportStatus ? el('p', { id: 'report-export-status' }, state.exportStatus) : null,
        renderPerfBlock('reports-hub-perf'),
      ),
    );
  }

  loadSaved();
  render();

  return {
    destroy() {
      destroyed = true;
      exportAbort?.abort();
      exportGate.release();
    },
  };
}
