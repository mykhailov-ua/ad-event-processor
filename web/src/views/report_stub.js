import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId, hasBoundCustomer } from '../helpers/buyer_session.js';
import { probeStubReport, submitReportExport, pollReportJob, downloadReportExport } from '../helpers/report_api.js';
import { reportTitle } from '../models/report.js';
import { renderPerfBlock } from '../helpers/perf_display.js';
import { renderStubBanner } from '../ui/stub_banner.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { createInFlightGuard } from '../lib/async_guard.js';

/**
 * Mount a stub report page that probes the planned API endpoint.
 *
 * @param {HTMLElement} container
 * @param {{ params: Record<string, string> }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const reportKey = ctx.params.reportKey;
  const title = reportTitle(reportKey);
  const user = auth.getUser();
  const customerId = hasBoundCustomer(user?.role) ? boundCustomerId(user) : '';
  const exportGate = createInFlightGuard();

  const state = {
    loading: true,
    probe: null,
    exportStatus: null,
    exportLoading: false,
  };

  async function load() {
    const [probe] = await to(probeStubReport(reportKey, customerId));
    if (destroyed) return;
    state.probe = probe;
    state.loading = false;
    render();
  }

  async function handleExport() {
    if (!exportGate.tryAcquire()) return;
    if (!customerId) {
      state.exportStatus = 'Customer context required for export.';
      exportGate.release();
      render();
      return;
    }
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
      const polled = await pollReportJob(result.jobId);
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
        ? `Export stub (${result.status}): ${result.message}`
        : `Job ${result.jobId ?? 'queued'}`;
    }
    exportGate.release();
    render();
  }

  function render() {
    if (destroyed) return;

    const children = [
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, title),
        ),
      ),
      el('p', null, el('a', { href: '/reports' }, '← Reports hub')),
    ];

    if (state.loading) {
      children.push(el('p', null, 'Probing API endpoint…'));
    } else if (state.probe?.stub) {
      children.push(renderStubBanner({
        message: state.probe.message || 'Endpoint is planned but not implemented (501).',
      }));
      children.push(
        el('p', null, `Probed: ${state.probe.path} (${state.probe.status})`),
      );
    } else if (state.probe && !state.probe.ok) {
      children.push(el('p', null, `Unexpected response: ${state.probe.status} — ${state.probe.message}`));
    }

    children.push(
      el('p', null,
        el('a', { href: '/reports/placements' }, 'Live: placements'),
        ' · ',
        el('a', { href: '/reports/keywords' }, 'Keywords'),
      ),
      el('button', {
        type: 'button',
        disabled: state.exportLoading || !customerId,
        onClick: handleExport,
      }, state.exportLoading ? 'Exporting…' : 'Request CSV export'),
      state.exportStatus ? el('p', { id: 'stub-export-status' }, state.exportStatus) : null,
      renderPerfBlock('report-stub-perf'),
    );

    replaceChildren(container, el('section', { 'data-testid': `report-stub-${reportKey}` }, ...children));
  }

  load();
  render();

  return {
    destroy() {
      destroyed = true;
      exportGate.release();
    },
  };
}
