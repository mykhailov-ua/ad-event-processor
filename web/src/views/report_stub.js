import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId, hasBoundCustomer } from '../helpers/buyer_session.js';
import { probeStubReport, submitReportExport, pollReportJob, downloadReportExport } from '../helpers/report_api.js';
import { reportTitle, retiredReportAlt, isRetiredReport } from '../models/report.js';
import { renderPerfBlock } from '../helpers/perf_display.js';
import { renderStubBanner } from '../ui/stub_banner.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { createInFlightGuard } from '../lib/async_guard.js';

/** @type {Record<string, { message: string, live: Array<{ href: string, label: string }> }>} */
const STUB_COPY = {
  'customer-portfolio': {
    message: 'Customer portfolio is not available yet. Use Placements or Keywords for live campaign data.',
    live: [
      { href: '/reports/placements', label: 'Placements' },
      { href: '/reports/keywords', label: 'Keywords' },
    ],
  },
};

/**
 * @param {string} reportKey
 * @returns {{ message: string, live: Array<{ href: string, label: string }> }}
 */
function stubCopy(reportKey) {
  if (STUB_COPY[reportKey]) return STUB_COPY[reportKey];
  return {
    message: `${reportTitle(reportKey)} is not available yet.`,
    live: [
      { href: '/reports/placements', label: 'Placements' },
      { href: '/reports/keywords', label: 'Keywords' },
    ],
  };
}

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
  const copy = stubCopy(reportKey);
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
    if (isRetiredReport(reportKey)) {
      state.probe = { stub: false, ok: false, retired: true };
      state.loading = false;
      render();
      return;
    }
    const [probe] = await to(probeStubReport(reportKey, customerId));
    if (destroyed) return;
    state.probe = probe;
    state.loading = false;
    render();
  }

  async function handleExport() {
    if (!exportGate.tryAcquire()) return;
    if (!customerId) {
      state.exportStatus = 'Select a customer to request an export.';
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
        ? `Export not available yet (${result.status}).`
        : `Job ${result.jobId ?? 'queued'}`;
    }
    exportGate.release();
    render();
  }

  function renderLiveLinks() {
    return el('div', { className: 'page-header__links' },
      ...copy.live.map((link, i) => [
        i > 0 ? el('span', { className: 'text-muted' }, '·') : null,
        el('a', { href: link.href }, link.label),
      ]).flat(),
    );
  }

  function render() {
    if (destroyed) return;

    const children = [
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, title),
        ),
        el('p', { className: 'text-muted' },
          el('a', { href: '/reports' }, '← Reports hub'),
        ),
      ),
    ];

    if (state.loading) {
      children.push(el('p', { className: 'text-muted' }, 'Checking availability…'));
    } else if (isRetiredReport(reportKey)) {
      const alt = retiredReportAlt(reportKey);
      children.push(renderStubBanner({
        message: `${title} was retired. Use ${alt?.label ?? 'a live report'} instead.`,
      }));
    } else if (state.probe?.stub) {
      children.push(renderStubBanner({ message: copy.message }));
      children.push(
        el('p', { className: 'text-sm text-muted' }, 'This report is planned but not built yet.'),
      );
    } else if (state.probe && !state.probe.ok) {
      children.push(
        el('p', { className: 'text-muted' },
          `Unexpected response (${state.probe.status}). ${state.probe.message || ''}`.trim(),
        ),
      );
    }

    children.push(
      el('div', { className: 'section-card' },
        el('h2', { className: 'subsection-title' }, 'Available now'),
        isRetiredReport(reportKey) && retiredReportAlt(reportKey)
          ? el('p', null, el('a', { href: retiredReportAlt(reportKey).href, className: 'btn btn--primary btn--sm' },
            retiredReportAlt(reportKey).label,
          ))
          : renderLiveLinks(),
      ),
      !isRetiredReport(reportKey) ? el('div', { className: 'toolbar-row' },
        el('button', {
          type: 'button',
          className: 'btn btn--secondary',
          disabled: state.exportLoading || !customerId,
          onClick: handleExport,
        }, state.exportLoading ? 'Exporting…' : 'Request CSV export'),
        !customerId
          ? el('span', { className: 'text-sm text-muted' }, 'Customer context required for export.')
          : null,
      ) : null,
      state.exportStatus
        ? el('p', { id: 'stub-export-status', className: 'text-sm text-muted' }, state.exportStatus)
        : null,
      renderPerfBlock('report-stub-perf'),
    );

    replaceChildren(container,
      el('div', { className: 'stack stack--lg', 'data-testid': `report-stub-${reportKey}` }, ...children),
    );
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
