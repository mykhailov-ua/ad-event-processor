import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import type { BuyerCampaignPortfolioRow } from '../types/api/campaign.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderSubsection } from '../ui/stat_tile.js';
import { fraudTierBandRows } from '../helpers/edge_fraud_tier.js';
import { renderCommercialMetrics, type MetricsBlockDTO } from '../ui/commercial_metrics.js';
import { renderFormField } from '../ui/form_field.js';
import { validateCustomerIdField } from '../helpers/validators.js';
import { formatAmountMicro } from '../helpers/money.js';
import { t } from '../helpers/i18n.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { pollReportJob, downloadReportExport } from '../helpers/report_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderEmptyState } from '../ui/data_table.js';
import { renderButton, renderButtonLink } from '../ui/button.js';

type SourceRow = {
  campaign_id?: string;
  sub1?: string;
  sub2?: string;
  ivt_rate?: number;
  quality_score?: number;
  spend_micro?: number;
  [key: string]: unknown;
};

type ExportJobRow = {
  id: string;
  status?: string;
  format?: string;
  [key: string]: unknown;
};

type AccountantClose = {
  billing_month?: string;
  invariant_ok?: boolean;
  [key: string]: unknown;
};

type FraudTierThresholds = {
  pass_max?: number;
  suspect_max?: number;
  ivt_max?: number;
  [key: string]: unknown;
};

type FraudGeoHint = {
  country?: string;
  ivt_rate?: number;
  ivt_events?: number;
  [key: string]: unknown;
};

type FraudLabelRow = {
  ip_hash?: string;
  label?: number;
  reason?: string;
  created_at?: string;
  [key: string]: unknown;
};

type RoleDashboardData = {
  customer_id?: string;
  kpis?: MetricsBlockDTO | null;
  campaigns?: BuyerCampaignPortfolioRow[];
  worst_sources?: SourceRow[];
  billed_micro?: number;
  ar_aging_micro?: number;
  dispute_exposure_micro?: number;
  close?: AccountantClose;
  tax_country?: string;
  tax_scheme?: string;
  tax_vat_id?: string;
  export_jobs?: ExportJobRow[];
  ghost_ivt_campaigns?: number;
  labels_pending?: number;
  edge_blocked_fraud?: number;
  ml_active_version_id?: string;
  ml_artifact_hash?: string;
  ml_precision?: number;
  ml_recall?: number;
  ml_drift_detected?: boolean;
  fraud_tier_thresholds?: FraudTierThresholds;
  geo_hints?: FraudGeoHint[];
  recent_labels?: FraudLabelRow[];
  [key: string]: unknown;
};

/**
 * Mount a role-specific dashboard (adops, cfo, accountant, fraud).
 *
 * @param {HTMLElement} container
 * @param {{ params: { role: string } }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const role = ctx.params.role;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  let customerInput = sessionScoped ? boundCustomerId(user) : '';
  let customerError: string | null = null;
  let loading = false;
  let data: RoleDashboardData | null = null;
  let blockError: Error | string | null = null;
  let exportJobState: Record<string, { status: string; error?: string }> = {};
  let exportPollTimer: ReturnType<typeof setInterval> | null = null;
  let exportPollAbort: AbortController | null = null;

  const titles: Record<string, string> = {
    adops: 'AdOps dashboard',
    cfo: 'CFO dashboard',
    accountant: 'Accountant dashboard',
    fraud: 'Fraud dashboard',
  };
  const endpoints: Record<string, string> = {
    adops: '/api/v1/dashboards/adops',
    cfo: '/api/v1/dashboards/cfo',
    accountant: '/api/v1/dashboards/accountant',
    fraud: '/api/v1/dashboards/fraud',
  };

  async function load() {
    customerError = sessionScoped ? null : validateCustomerIdField(customerInput);
    const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();
    if (!cid || customerError) {
      stopExportPoll();
      render();
      return;
    }
    loading = true;
    blockError = null;
    render();
    const path = endpoints[role];
    if (!path) {
      blockError = new Error('Unknown dashboard role');
      loading = false;
      render();
      return;
    }
    const params = new URLSearchParams({ customer_id: cid });
    const [res, err] = await to(api(`${path}?${params.toString()}`));
    if (destroyed) return;
    loading = false;
    if (err) {
      blockError = err;
      stopExportPoll();
    } else {
      data = (res?.data ?? null) as RoleDashboardData | null;
      if (role === 'accountant') {
        syncExportJobState(data?.export_jobs ?? []);
        startExportPoll();
      }
    }
    render();
  }

  /**
   * Merge export job rows from the dashboard API into local poll state.
   *
   * @param {ExportJobRow[]} jobs
   */
  function syncExportJobState(jobs: ExportJobRow[]) {
    for (let i = 0; i < jobs.length; i++) {
      const job = jobs[i];
      if (!job?.id) continue;
      exportJobState[job.id] = {
        status: job.status ?? exportJobState[job.id]?.status ?? 'PENDING',
        error: exportJobState[job.id]?.error,
      };
    }
  }

  /**
   * Return true when any tracked export job is still in flight.
   *
   * @returns {boolean}
   */
  function hasPendingExportJobs() {
    const ids = Object.keys(exportJobState);
    for (let i = 0; i < ids.length; i++) {
      const status = exportJobState[ids[i]]?.status ?? '';
      if (status === 'PENDING' || status === 'RUNNING') return true;
    }
    return false;
  }

  /**
   * Poll in-flight report export jobs until terminal status.
   *
   * @returns {Promise<void>}
   */
  async function pollPendingExportJobs() {
    if (destroyed || !hasPendingExportJobs()) return;
    exportPollAbort?.abort();
    exportPollAbort = new AbortController();
    const signal = exportPollAbort.signal;
    const ids = Object.keys(exportJobState);
    for (let i = 0; i < ids.length; i++) {
      const jobId = ids[i];
      const current = exportJobState[jobId]?.status ?? '';
      if (current !== 'PENDING' && current !== 'RUNNING') continue;
      const result = await pollReportJob(jobId, { signal, maxAttempts: 1, intervalMs: 0 });
      if (destroyed || signal.aborted) return;
      exportJobState[jobId] = {
        status: result.status,
        error: result.ok ? undefined : result.message,
      };
    }
    render();
    if (!destroyed && hasPendingExportJobs()) {
      startExportPoll();
    }
  }

  /**
   * Start interval polling for accountant export jobs.
   *
   * @returns {void}
   */
  function startExportPoll() {
    if (role !== 'accountant' || !hasPendingExportJobs()) return;
    if (exportPollTimer) return;
    exportPollTimer = setInterval(() => {
      pollPendingExportJobs();
    }, 2500);
  }

  /**
   * Stop export job polling timers and abort in-flight polls.
   *
   * @returns {void}
   */
  function stopExportPoll() {
    if (exportPollTimer) {
      clearInterval(exportPollTimer);
      exportPollTimer = null;
    }
    exportPollAbort?.abort();
    exportPollAbort = null;
  }

  /**
   * Download a completed report export job.
   *
   * @param {string} jobId
   * @returns {Promise<void>}
   */
  async function downloadExportJob(jobId: string) {
    const [, err] = await to(downloadReportExport(jobId, `export-${jobId.slice(0, 8)}.csv`));
    if (err) {
      pushToastMessage({ title: 'Download failed', message: err.message ?? String(err) });
    }
  }

  function renderAdOps() {
    if (!data) return el('p', { className: 'loading-hint' }, t('status.loading'));
    const dash = data;
    const kpis = renderCommercialMetrics(dash.kpis, { masked: false });
    const campaigns = Array.isArray(dash.campaigns) ? dash.campaigns : [];
    const worst = Array.isArray(dash.worst_sources) ? dash.worst_sources : [];
    return el('div', { className: 'stack stack--lg section-block' },
      kpis,
      renderSubsection('Campaigns',
        campaigns.length === 0
          ? renderEmptyState({
            title: 'No campaigns',
            description: 'Campaign utilization data will appear when campaigns are active.',
            icon: 'megaphone',
          })
          : el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Campaign'),
                  el('th', { scope: 'col' }, 'Util %'),
                  el('th', { scope: 'col' }, 'Drift %'),
                  el('th', { scope: 'col' }, 'Spend'),
                ),
              ),
              el('tbody', null,
                campaigns.map((c) => el('tr', null,
                  el('td', null, el('a', { href: `/campaigns/${c.id}` }, c.name ?? c.id)),
                  el('td', null, c.utilization_pct?.toFixed?.(1) ?? '—'),
                  el('td', null, c.pacing_drift_pct?.toFixed?.(1) ?? '—'),
                  el('td', { className: 'font-mono' }, formatAmountMicro(c.spend_micro ?? 0)),
                )),
              ),
            ),
          ),
      ),
      renderSubsection('Worst sources',
        worst.length === 0
          ? renderEmptyState({
            title: 'No quality flags',
            description: 'No placement issues from the last reporting window.',
            icon: 'grid-four',
          })
          : el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Source'),
                  el('th', { scope: 'col' }, 'Campaign'),
                  el('th', { scope: 'col' }, 'IVT %'),
                  el('th', { scope: 'col' }, 'Quality'),
                  el('th', { scope: 'col' }, 'Spend'),
                  el('th', { scope: 'col' }, ''),
                ),
              ),
              el('tbody', null,
                worst.map((s) => {
                  const placementsHref = dash.customer_id
                    ? `/reports/placements?customer_id=${encodeURIComponent(dash.customer_id)}&campaign_id=${encodeURIComponent(s.campaign_id ?? '')}`
                    : '/reports/placements';
                  return el('tr', null,
                    el('td', null, s.sub1 ?? s.sub2 ?? '—'),
                    el('td', null,
                      s.campaign_id
                        ? el('a', { href: `/campaigns/${s.campaign_id}` }, s.campaign_id.slice(0, 8))
                        : '—',
                    ),
                    el('td', null, `${((s.ivt_rate ?? 0) * 100).toFixed(1)}%`),
                    el('td', null, s.quality_score != null ? s.quality_score.toFixed(1) : '—'),
                    el('td', { className: 'font-mono' }, formatAmountMicro(s.spend_micro ?? 0)),
                    el('td', null, renderButtonLink({
                      href: placementsHref,
                      label: 'Placements',
                      variant: 'ghost',
                      size: 'sm',
                    })),
                  );
                }),
              ),
            ),
          ),
      ),
    );
  }

  function renderCFO() {
    if (!data) return el('p', { className: 'loading-hint' }, t('status.loading'));
    return el('dl', { className: 'definition-list section-block' },
      el('dt', null, 'Billed'),
      el('dd', { className: 'font-mono' }, formatAmountMicro(data.billed_micro ?? 0)),
      el('dt', null, 'AR aging'),
      el('dd', { className: 'font-mono' }, formatAmountMicro(data.ar_aging_micro ?? 0)),
      el('dt', null, 'Dispute exposure'),
      el('dd', { className: 'font-mono' }, formatAmountMicro(data.dispute_exposure_micro ?? 0)),
    );
  }

  function renderAccountant() {
    if (!data) return el('p', { className: 'loading-hint' }, t('status.loading'));
    const close: AccountantClose = data.close ?? {};
    const jobs = Array.isArray(data.export_jobs) ? data.export_jobs : [];
    return el('div', { className: 'stack section-block' },
      el('dl', { className: 'definition-list' },
        el('dt', null, 'Billing month'),
        el('dd', null, close.billing_month ?? '—'),
        el('dt', null, 'Invariant OK'),
        el('dd', null, close.invariant_ok ? 'Yes' : 'No'),
        el('dt', null, 'Tax country'),
        el('dd', null, data.tax_country ?? '—'),
        el('dt', null, 'Tax scheme'),
        el('dd', null, data.tax_scheme ?? data.tax_vat_id ?? '—'),
      ),
      renderSubsection('Export jobs',
        jobs.length === 0
          ? renderEmptyState({
            title: 'No export jobs',
            description: 'Scheduled or manual exports will appear here.',
            icon: 'download',
          })
          : el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table', 'data-testid': 'accountant-export-jobs' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Job'),
                  el('th', { scope: 'col' }, 'Format'),
                  el('th', { scope: 'col' }, 'Status'),
                  el('th', { scope: 'col' }, ''),
                ),
              ),
              el('tbody', null,
                jobs.map((j) => {
                  const live = exportJobState[j.id] ?? { status: j.status };
                  const status = live.status ?? j.status ?? 'PENDING';
                  const badgeStatus = status === 'COMPLETED' ? 'ok'
                    : status === 'FAILED' ? 'failed'
                      : status === 'RUNNING' ? 'pending'
                        : 'pending';
                  const badgeKind = status === 'COMPLETED' || status === 'FAILED' || status === 'RUNNING'
                    ? 'service'
                    : 'invoice';
                  return el('tr', { 'data-testid': `export-job-${j.id}` },
                    el('td', { className: 'font-mono text-sm' }, j.id.slice(0, 8)),
                    el('td', null, j.format ?? 'csv'),
                    el('td', null, renderStatusBadge(badgeStatus, { kind: badgeKind, label: status })),
                    el('td', null,
                      status === 'COMPLETED'
                        ? renderButton({
                          label: 'Download',
                          variant: 'secondary',
                          size: 'sm',
                          onClick: () => downloadExportJob(j.id),
                        })
                        : (status === 'PENDING' || status === 'RUNNING')
                          ? el('span', { className: 'text-muted text-sm' }, 'Polling…')
                          : live.error
                            ? el('span', { className: 'text-danger text-sm' }, live.error)
                            : null,
                    ),
                  );
                }),
              ),
            ),
          ),
      ),
    );
  }

  function renderFraud() {
    if (!data) return el('p', { className: 'loading-hint' }, t('status.loading'));
    const thresholds = data.fraud_tier_thresholds ?? {};
    const geoHints = Array.isArray(data.geo_hints) ? data.geo_hints : [];
    const recentLabels = Array.isArray(data.recent_labels) ? data.recent_labels : [];
    const customerQS = data.customer_id
      ? `?customer_id=${encodeURIComponent(data.customer_id)}`
      : '';

    return el('div', { className: 'stack stack--lg section-block', 'data-testid': 'fraud-dashboard' },
      el('dl', { className: 'definition-list' },
        el('dt', null, 'Ghost IVT campaigns'),
        el('dd', null, String(data.ghost_ivt_campaigns ?? 0)),
        el('dt', null, 'Labels queue (7d)'),
        el('dd', null, String(data.labels_pending ?? 0)),
        el('dt', null, 'Edge blocked (fraud tier)'),
        el('dd', { className: 'font-mono' }, String(data.edge_blocked_fraud ?? 0)),
        el('dt', null, 'ML active version'),
        el('dd', { className: 'font-mono' }, data.ml_active_version_id ?? '—'),
        el('dt', null, 'ML artifact hash'),
        el('dd', { className: 'font-mono text-sm' }, data.ml_artifact_hash ? `${data.ml_artifact_hash.slice(0, 12)}…` : '—'),
        el('dt', null, 'Shadow eval precision'),
        el('dd', null, data.ml_precision != null && data.ml_precision > 0
          ? `${(data.ml_precision * 100).toFixed(1)}%`
          : '—'),
        el('dt', null, 'Shadow eval recall'),
        el('dd', null, data.ml_recall != null && data.ml_recall > 0
          ? `${(data.ml_recall * 100).toFixed(1)}%`
          : '—'),
        el('dt', null, 'Model drift'),
        el('dd', null, data.ml_drift_detected ? renderStatusBadge('warning', { label: 'detected' }) : 'No'),
      ),
      renderSubsection('Edge fraud tier thresholds',
        el('div', { className: 'table-wrapper' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, 'Tier'),
                el('th', { scope: 'col' }, 'Score'),
              ),
            ),
            el('tbody', null,
              fraudTierBandRows().map((row) => el('tr', null,
                el('td', null, row.tier),
                el('td', { className: 'font-mono' }, row.range),
              )),
            ),
          ),
          thresholds.pass_max
            ? el('p', { className: 'text-muted text-xs' },
              `API: pass≤${thresholds.pass_max}, suspect≤${thresholds.suspect_max}, ivt≤${thresholds.ivt_max}`)
            : null,
        ),
      ),
      renderSubsection('Geo suggestions (read-only)',
        geoHints.length === 0
          ? renderEmptyState({
            title: 'No high-IVT countries',
            description: 'No geo suggestions in the last 7 days.',
            icon: 'globe',
          })
          : el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Country'),
                  el('th', { scope: 'col' }, 'IVT %'),
                  el('th', { scope: 'col' }, 'Events'),
                  el('th', { scope: 'col' }, ''),
                ),
              ),
              el('tbody', null,
                geoHints.map((hint) => el('tr', null,
                  el('td', null, hint.country ?? '—'),
                  el('td', null, `${((hint.ivt_rate ?? 0) * 100).toFixed(1)}%`),
                  el('td', { className: 'font-mono' }, String(hint.ivt_events ?? 0)),
                  el('td', null,
                    renderButtonLink({
                      href: `/reports/ivt-by-source${customerQS}`,
                      label: 'IVT report',
                      variant: 'ghost',
                      size: 'sm',
                    }),
                  ),
                )),
              ),
            ),
          ),
        el('p', { className: 'text-muted text-sm' },
          'Review suspicious IPs via ',
          el('a', { href: '/ops/blacklist' }, 'ops blacklist'),
          ' (dry-run: POST with ',
          el('code', null, 'dry_run=1'),
          ').',
        ),
      ),
      renderSubsection('Manual labels queue',
        recentLabels.length === 0
          ? renderEmptyState({
            title: 'No manual labels in queue',
            description: 'Fraud label review queue is empty.',
            icon: 'tag',
          })
          : el('div', { className: 'table-wrapper' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'IP hash'),
                  el('th', { scope: 'col' }, 'Label'),
                  el('th', { scope: 'col' }, 'Reason'),
                  el('th', { scope: 'col' }, 'Created'),
                ),
              ),
              el('tbody', null,
                recentLabels.map((row) => el('tr', null,
                  el('td', { className: 'font-mono text-sm' }, `${row.ip_hash?.slice(0, 8) ?? ''}…`),
                  el('td', null, row.label === 1 ? 'fraud' : 'legit'),
                  el('td', null, row.reason ?? '—'),
                  el('td', { className: 'text-sm text-muted' }, row.created_at ?? '—'),
                )),
              ),
            ),
          ),
      ),
    );
  }

  const renderers: Record<string, () => HTMLElement> = {
    adops: renderAdOps,
    cfo: renderCFO,
    accountant: renderAccountant,
    fraud: renderFraud,
  };

  function render() {
    if (destroyed) return;
    if (blockError) {
      replaceChildren(container, renderErrorBlock(blockError));
      return;
    }
    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, titles[role] ?? 'Dashboard'),
      ),
      el('form', {
        className: 'filter-form section-block',
        onSubmit: (e: Event) => {
          e.preventDefault();
          load();
        },
      },
        !sessionScoped
          ? renderFormField({
            label: 'Customer ID',
            htmlFor: 'role-dash-customer',
            error: customerError,
            children: el('input', {
              id: 'role-dash-customer',
              className: 'form-input',
              value: customerInput,
              onInput: (e: Event) => {
                customerInput = eventTargetValue(e);
                customerError = validateCustomerIdField(customerInput);
              },
            }),
          })
          : null,
        renderButton({
          label: t('action.load'),
          variant: 'primary',
          type: 'submit',
          loading: loading,
          disabled: loading,
        }),
      ),
      loading ? el('p', { className: 'loading-hint' }, t('status.loading')) : (renderers[role]?.() ?? renderEmptyState({ title: 'Unknown role', description: 'This dashboard role is not configured.' })),
    );
  }

  load();
  return {
    destroy() {
      destroyed = true;
      stopExportPoll();
    },
  };
}
