import type { ViewHandle } from '../lib/router_types.js';
import type {
  DashboardSummary,
  IncidentSnapshot,
  OpsDoctorSummary,
  OutboxEventDTO,
  OutboxListResponse,
  ShardHealthStatus,
  DLQEntryDTO,
  DLQListResponse,
  BillingInvariantDTO,
} from '../types/api/index.js';
import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api, ApiError, type ApiResult } from '../helpers/api_client.js';
import { isParallelSlotError, parallelAll } from '../helpers/request_multiplex.js';
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
import { createGenerationGuard, createInFlightGuard, shouldCommitAsyncResult } from '../lib/async_guard.js';
import { apiTimingReport } from '../helpers/api_timing.js';
import { flushRUMNow } from '../helpers/rum_collector.js';
import { renderStatusHint } from '../ui/status_hint.js';
import { renderEdgePanel, renderXDPPanel, type EdgePanelData, type XDPPanelData } from '../ui/edge_panel.js';
import { displayLabel, formatYesNo } from '../helpers/display_labels.js';
import * as storage from '../helpers/storage.js';
import {
  metricColorToken,
  OPS_CHART_RANGE_OPTIONS,
  OPS_METRIC_API_NAMES,
  parseApiPoints,
  rangeMsFromHours,
  recordSnapshot,
  snapshotSeries,
  toRateSeries,
  type ApiMetricRow,
  type MetricPoint,
} from '../helpers/ops_metric_series.js';
import { formatChartTick, formatClockTime, formatRefreshCountdown } from '../helpers/chart_format.js';
import { connectOpsLiveFeed } from '../helpers/ops_live_feed.js';
import { fetchBillingInvariant } from '../helpers/billing_admin_api.js';
import { retryOpsDlq } from '../helpers/ops_dlq_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { formatAmountMicro } from '../helpers/money.js';
import { renderSubsection } from '../ui/stat_tile.js';
import { mount as mountSegmentedControl } from '../ui/segmented_control.js';
import { renderEmptyTableCell } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { mountFilterToolbar } from '../ui/filter_toolbar.js';
import type { MetricChartHandle } from '../charts/metric_chart.js';

type PartialSourceError = { source?: string; code?: string };

type OperatorDash = {
  edge?: EdgePanelData;
  xdp?: XDPPanelData;
  [key: string]: unknown;
};

type OpsHomeState = {
  loading: boolean;
  doctor: OpsDoctorSummary | null;
  incidents: IncidentSnapshot | null;
  summary: DashboardSummary | null;
  partialErrors: PartialSourceError[];
  partialDismissed: boolean;
  blockError: unknown | null;
  rumEvents: number;
  slowApiPaths: string[];
  operatorDash: OperatorDash | null;
  metricSeries: Record<string, MetricPoint[]>;
  lastUpdatedAt: number;
  nextRefreshAt: number;
  feedMode: string;
};

type ChartHandleEntry = MetricChartHandle & { id?: string };

type MetricsSeriesResponse = {
  points?: ApiMetricRow[];
};

type RumResponse = {
  events?: unknown[];
};

/**
 * Mount the operations home view with doctor, outbox, and shard summary.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  let tab = 'overview';
  let outboxStatus = '';
  let outboxItems: OutboxEventDTO[] = [];
  let outboxCursor = '';
  let outboxLoading = false;
  let dlqItems: DLQEntryDTO[] = [];
  let dlqCursor = '';
  let dlqLoading = false;
  let dlqPartialErrors: Array<{ source?: string; code?: string }> = [];
  let invariantFilter = '';
  let invariantState: BillingInvariantDTO | null = null;
  let invariantLoading = false;
  const outboxGuard = createGenerationGuard();
  const dlqGuard = createGenerationGuard();
  const bundleGate = createInFlightGuard();

  const state: OpsHomeState = {
    loading: true,
    doctor: null,
    incidents: null,
    summary: null,
    partialErrors: [],
    partialDismissed: false,
    blockError: null,
    rumEvents: 0,
    slowApiPaths: [],
    operatorDash: null,
    metricSeries: {},
    lastUpdatedAt: 0,
    nextRefreshAt: 0,
    feedMode: 'poll',
  };

  const OPS_POLL_MS = 30_000;
  let clockTimer: ReturnType<typeof setInterval> | null = null;
  /** @type {HTMLElement | null} */
  let statusBarEl: HTMLElement | null = null;
  /** @type {{ destroy: () => void } | null} */
  let liveFeed: { destroy: () => void } | null = null;

  const user = auth.getUser();
  const canBundle = can(user?.permissions ?? [], 'ops:write');
  const canShardsWrite = can(user?.permissions ?? [], 'shards:write');
  const canShardsRead = can(user?.permissions ?? [], 'shards:read');
  const canBillingRead = can(user?.permissions ?? [], 'customers:read');

  let chartHandles: ChartHandleEntry[] = [];
  /** @type {{ destroy: () => void }|null} */
  let layoutControlHandle: { destroy: () => void } | null = null;
  let chartsLayout = storage.getOpsChartsLayout();
  let chartsRangeHours = storage.getOpsChartsRangeHours();
  /** @type {{ destroy: () => void }|null} */
  let rangeControlHandle: { destroy: () => void } | null = null;

  function destroyCharts() {
    for (let i = 0; i < chartHandles.length; i++) {
      chartHandles[i]?.destroy?.();
    }
    chartHandles = [];
    layoutControlHandle?.destroy?.();
    layoutControlHandle = null;
    rangeControlHandle?.destroy?.();
    rangeControlHandle = null;
  }

  /**
   * Paint the live status bar without a full page render.
   */
  function updateStatusBar() {
    if (!statusBarEl) return;
    const now = Date.now();
    const lastLabel = formatClockTime(state.lastUpdatedAt);
    const countdown = formatRefreshCountdown(state.nextRefreshAt - now);
    const live = state.feedMode === 'stream';
    statusBarEl.replaceChildren(
      el('span', { className: 'ops-charts-status__item' },
        el('span', {
          className: `ops-charts-status__dot${live ? ' ops-charts-status__dot--live' : ''}`,
          'aria-hidden': 'true',
        }),
        live ? 'Live stream + polling' : 'Polling',
      ),
      el('span', { className: 'ops-charts-status__item' },
        'Last update: ',
        el('strong', null, lastLabel),
      ),
      el('span', { className: 'ops-charts-status__item ops-charts-status__item--muted' },
        `Next refresh in ${countdown}`,
      ),
    );
  }

  /**
   * Mark a successful data refresh timestamp.
   */
  function markRefreshed() {
    const now = Date.now();
    state.lastUpdatedAt = now;
    state.nextRefreshAt = now + OPS_POLL_MS;
    updateStatusBar();
  }

  /**
   * @returns {Array<{
   *   id: string,
   *   title: string,
   *   value: number,
   *   points: Array<{ ts: number, value: number }>,
   *   color: string,
   *   max?: number,
   *   formatValue?: (value: number) => string,
   * }>}
   */
  function buildMetricSpecs() {
    /** @type {ReturnType<typeof buildMetricSpecs>} */
    const specs = [];
    let dropIndex = 0;
    const rangeMs = rangeMsFromHours(chartsRangeHours);
    const s = state.summary;
    if (s) {
      const outboxVal = Number(s.outbox_pending) || 0;
      specs.push({
        id: 'outbox-pending',
        title: 'Outbox pending',
        value: outboxVal,
        points: state.metricSeries['outbox-pending'] ?? snapshotSeries('outbox-pending', outboxVal, rangeMs),
        color: metricColorToken('outbox-pending'),
        displayValue: formatChartTick(outboxVal),
      });
      const rpsVal = Number(s.rps_estimate) || 0;
      specs.push({
        id: 'rps-estimate',
        title: 'RPS estimate',
        value: rpsVal,
        points: state.metricSeries['rps-estimate'] ?? snapshotSeries('rps-estimate', rpsVal, rangeMs),
        color: metricColorToken('rps-estimate'),
        formatValue: (v: any) => v.toFixed(1),
        displayValue: rpsVal.toFixed(1),
      });
      const driftVal = Number(s.drift_micro_max) || 0;
      specs.push({
        id: 'drift-alert',
        title: 'Drift (micro)',
        value: driftVal,
        points: state.metricSeries['drift-alert'] ?? snapshotSeries('drift-alert', driftVal, rangeMs),
        color: metricColorToken('drift-alert'),
        formatValue: (v: any) => formatChartTick(v),
        displayValue: formatChartTick(driftVal),
      });
      const breakerVal = String(s.emergency_breaker).toLowerCase() === 'open' ? 1 : 0;
      specs.push({
        id: 'emergency-breaker',
        title: 'Emergency breaker',
        value: breakerVal,
        points: snapshotSeries('emergency-breaker', breakerVal, rangeMs),
        color: metricColorToken('emergency-breaker'),
        max: 1,
        formatValue: (v: any) => (v > 0 ? 'Open' : 'Closed'),
        displayValue: breakerVal > 0 ? 'Open' : 'Closed',
      });
    }

    const edge = state.operatorDash?.edge;
    if (edge) {
      const ingress = [
        { id: 'ingress-h1', title: 'HTTP/1 ingress', value: Number(edge.ingress_h1) || 0 },
        { id: 'ingress-h2', title: 'HTTP/2 ingress', value: Number(edge.ingress_h2) || 0 },
        { id: 'ingress-h3', title: 'HTTP/3 ingress', value: Number(edge.ingress_h3) || 0 },
      ];
      for (let i = 0; i < ingress.length; i++) {
        const item = ingress[i];
        specs.push({
          ...item,
          points: snapshotSeries(item.id, item.value, rangeMs),
          color: metricColorToken(item.id),
          displayValue: formatChartTick(item.value),
        });
      }
      const botSignals = [
        { id: 'edge-tarpit', title: 'Edge tarpit', value: Number(edge.tarpit_total) || 0 },
        { id: 'edge-blacklist-stale', title: 'Blacklist stale', value: Number(edge.blacklist_stale) || 0 },
        { id: 'edge-fraud-tier', title: 'Fraud tier blocks', value: Number(edge.blocked?.fraud_tier) || 0 },
      ];
      for (let i = 0; i < botSignals.length; i++) {
        const item = botSignals[i];
        specs.push({
          ...item,
          points: snapshotSeries(item.id, item.value, rangeMs),
          color: metricColorToken(item.id),
          displayValue: formatChartTick(item.value),
        });
      }
    }

    const drops = state.operatorDash?.xdp?.drops;
    if (drops && typeof drops === 'object') {
      for (const key of Object.keys(drops).sort()) {
        const id = `drop-${key}`;
        const value = Number(drops[key]) || 0;
        specs.push({
          id,
          title: displayLabel(key),
          value,
          points: snapshotSeries(id, value, rangeMs),
          color: metricColorToken(id, dropIndex),
          displayValue: formatChartTick(value),
        });
        dropIndex += 1;
      }
    }

    return specs;
  }

  function recordSnapshotMetrics() {
    const s = state.summary;
    if (s) {
      recordSnapshot('outbox-pending', Number(s.outbox_pending) || 0);
      recordSnapshot('rps-estimate', Number(s.rps_estimate) || 0);
      recordSnapshot('drift-alert', Number(s.drift_micro_max) || 0);
      recordSnapshot(
        'emergency-breaker',
        String(s.emergency_breaker).toLowerCase() === 'open' ? 1 : 0,
      );
    }
    const edge = state.operatorDash?.edge;
    if (edge) {
      recordSnapshot('ingress-h1', Number(edge.ingress_h1) || 0);
      recordSnapshot('ingress-h2', Number(edge.ingress_h2) || 0);
      recordSnapshot('ingress-h3', Number(edge.ingress_h3) || 0);
      recordSnapshot('edge-tarpit', Number(edge.tarpit_total) || 0);
      recordSnapshot('edge-blacklist-stale', Number(edge.blacklist_stale) || 0);
      recordSnapshot('edge-fraud-tier', Number(edge.blocked?.fraud_tier) || 0);
    }
    const drops = state.operatorDash?.xdp?.drops;
    if (drops && typeof drops === 'object') {
      for (const key of Object.keys(drops).sort()) {
        recordSnapshot(`drop-${key}`, Number(drops[key]) || 0);
      }
    }
  }

  async function loadMetricSeries() {
    const ids = Object.keys(OPS_METRIC_API_NAMES);
    const ctrl = typeof AbortController !== 'undefined' ? new AbortController() : null;
    const timer = ctrl ? window.setTimeout(() => ctrl.abort(), 8_000) : 0;
    try {
      const results = await Promise.all(ids.map(async (id) => {
        const name = OPS_METRIC_API_NAMES[id];
        const [res] = await to(api(
          `/api/v1/ops/dashboard/metrics?range=${chartsRangeHours}h&name=${encodeURIComponent(name)}`,
          ctrl ? { signal: ctrl.signal } : {},
        ));
        let points = parseApiPoints(
          (res?.data as MetricsSeriesResponse | undefined)?.points
            ?? (res as MetricsSeriesResponse | null)?.points,
        );
        if (id === 'rps-estimate') points = toRateSeries(points);
        return { id, points };
      }));

      for (let i = 0; i < results.length; i++) {
        const { id, points } = results[i];
        if (points.length > 0) state.metricSeries[id] = points;
      }
    } finally {
      if (timer) window.clearTimeout(timer);
    }
  }

  function mountOpsCharts(opts: { reuse?: boolean } = {}) {
    const reuse = Boolean(opts.reuse);
    if (!reuse) {
      destroyCharts();
    }
    if (destroyed || tab !== 'overview' || state.loading) return;

    const specs = buildMetricSpecs();
    if (specs.length === 0) return;

    const layoutMount = container.querySelector('[data-ops-charts-layout]');
    const rangeMount = container.querySelector('[data-ops-charts-range]');
    if (!reuse && layoutMount instanceof HTMLElement) {
      layoutControlHandle = mountSegmentedControl(layoutMount, {
        items: [
          { value: 'grid', label: 'Grid (2 col)' },
          { value: 'stack', label: 'Stack (1 col)' },
        ],
        selected: chartsLayout,
        onChange: (value: any) => {
          chartsLayout = value === 'stack' ? 'stack' : 'grid';
          storage.setOpsChartsLayout(chartsLayout);
          const grid = container.querySelector('[data-ops-charts-grid]');
          if (grid) {
            grid.classList.toggle('ops-charts-grid--grid', chartsLayout === 'grid');
            grid.classList.toggle('ops-charts-grid--stack', chartsLayout === 'stack');
          }
        },
      });
    }

    if (!reuse && rangeMount instanceof HTMLElement) {
      rangeControlHandle = mountSegmentedControl(rangeMount, {
        items: OPS_CHART_RANGE_OPTIONS.map((opt: any) => ({
          value: String(opt.value),
          label: opt.label,
        })),
        selected: String(chartsRangeHours),
        onChange: (value: any) => {
          const hours = Number(value);
          if ((hours === 1 || hours === 6 || hours === 12 || hours === 24) && hours !== chartsRangeHours) {
            chartsRangeHours = hours;
            storage.setOpsChartsRangeHours(hours);
            loadMetricSeries().then(() => {
              if (!destroyed) mountOpsCharts({ reuse: true });
            });
          }
        },
      });
    }

    import('../charts/metric_chart.js').then((metricMod) => {
      if (destroyed || tab !== 'overview') return;
      const byId = new Map(
        chartHandles.filter((h): h is ChartHandleEntry & { id: string } => Boolean(h.id))
          .map((h) => [h.id, h]),
      );
      const nextHandles: ChartHandleEntry[] = [];

      for (let i = 0; i < specs.length; i++) {
        const spec = specs[i];
        const mount = container.querySelector(`[data-chart-id="${spec.id}"]`);
        if (!(mount instanceof HTMLElement)) continue;

        const payload = {
          title: spec.title,
          points: spec.points,
          value: spec.value,
          max: spec.max,
          color: spec.color,
          rangeHours: chartsRangeHours,
          formatValue: spec.formatValue,
        };

        const existing = reuse ? byId.get(spec.id) : null;
        if (existing?.update) {
          existing.update(payload);
          nextHandles.push(existing);
          continue;
        }

        const handle = metricMod.mountMetricChart(mount, payload);
        nextHandles.push({ ...handle, id: spec.id });
      }

      if (reuse) {
        for (const [id, handle] of byId) {
          if (!nextHandles.some((h) => h.id === id)) handle.destroy();
        }
      }
      chartHandles = nextHandles;
    });
  }

  function render() {
    if (destroyed) return;

    if (state.blockError) {
      replaceChildren(container, renderErrorBlock(state.blockError));
      return;
    }

    const shardSnippet = state.incidents?.shards ?? [];
    const metricSpecs = buildMetricSpecs();
    const chartsLayoutMount = el('div', { 'data-ops-charts-layout': '' });
    const chartsRangeMount = el('div', { 'data-ops-charts-range': '' });
    statusBarEl = el('div', { className: 'ops-charts-status', 'data-ops-status': '' });

    const outboxToolbar = (() => {
      const wrap = el('div', { className: 'mb-4' });
      mountFilterToolbar(wrap, {
        leading: [
          el('div', { className: 'cluster cluster--sm items-center' },
            el('span', { className: 'text-muted text-sm' }, 'Status'),
            renderSelect({
              value: outboxStatus,
              options: [
                { value: '', label: 'All' },
                { value: 'pending', label: 'Pending' },
                { value: 'processed', label: 'Processed' },
                { value: 'failed', label: 'Failed' },
              ],
              className: 'min-w-40',
              'aria-label': 'Outbox status',
              onChange: (v: string) => {
                outboxStatus = v;
                loadOutbox();
              },
            }),
          ),
        ],
        actions: outboxCursor
          ? [
            renderButton({
              label: 'Load more',
              variant: 'secondary',
              size: 'sm',
              loading: outboxLoading,
              disabled: outboxLoading,
              onClick: loadMoreOutbox,
            }),
          ]
          : [],
      });
      return wrap;
    })();

    const dlqToolbar = dlqCursor
      ? (() => {
        const wrap = el('div', { className: 'mb-4' });
        mountFilterToolbar(wrap, {
          actions: [
            renderButton({
              label: 'Load more',
              variant: 'secondary',
              size: 'sm',
              loading: dlqLoading,
              disabled: dlqLoading,
              onClick: loadMoreDlq,
            }),
          ],
        });
        return wrap;
      })()
      : null;

    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, 'Operations'),
          state.doctor
            ? renderStatusBadge(state.doctor.overall, {
              kind: 'service',
              label: `Doctor: ${displayLabel(state.doctor.overall)}`,
            })
            : null,
          canBundle
            ? renderButton({
              label: 'Support bundle',
              variant: 'secondary',
              size: 'sm',
              icon: 'download',
              className: 'ml-auto',
              loading: bundleGate.busy(),
              disabled: bundleGate.busy(),
              onClick: downloadBundle,
            })
            : null,
        ),
      ),
      state.partialErrors.length > 0 && !state.partialDismissed
        ? renderAlertBanner({
          variant: 'warning',
          message: `Partial source errors: ${state.partialErrors.map((e: PartialSourceError) => `${e.source ?? '?'} (${e.code ?? 'err'})`).join('; ')}`,
          dismissKey: 'ops.partial',
          onDismiss: () => {
            state.partialDismissed = true;
            render();
          },
        })
        : null,
      state.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
      !state.loading && state.summary
        ? el('div', { className: 'ops-kpi-strip section-block' },
          state.summary.drift_alert
            ? el('div', { className: 'ops-alert-bar', role: 'status' },
              renderIcon('alert-triangle', { size: 16, className: 'ops-alert-bar__icon' }),
              el('span', { className: 'ops-alert-bar__label' }, 'Drift alert'),
              el('span', { className: 'ops-alert-bar__value' }, formatYesNo(true)),
            )
            : null,
          el('div', { className: 'ops-kpi-row' },
            el('div', { className: 'ops-kpi-chip' },
              el('span', { className: 'ops-kpi-chip__label' }, 'Outbox pending'),
              el('span', { className: 'ops-kpi-chip__value' }, String(state.summary.outbox_pending)),
            ),
            el('div', { className: 'ops-kpi-chip' },
              el('span', { className: 'ops-kpi-chip__label' }, 'RPS estimate'),
              el('span', { className: 'ops-kpi-chip__value' },
                state.summary.rps_estimate?.toFixed(1) ?? '—',
              ),
            ),
            el('div', { className: 'ops-kpi-chip' },
              el('span', { className: 'ops-kpi-chip__label' }, 'Emergency breaker'),
              el('span', { className: 'ops-kpi-chip__value' },
                displayLabel(state.summary.emergency_breaker),
              ),
            ),
          ),
        )
        : null,
      tab === 'overview' && !state.loading && metricSpecs.length > 0
        ? renderSubsection('Operations metrics',
          el('div', { className: 'ops-charts-toolbar' },
            el('div', { className: 'ops-charts-toolbar__controls' },
              el('div', { className: 'ops-charts-toolbar__group' },
                el('p', { className: 'ops-charts-toolbar__label' }, 'Range'),
                chartsRangeMount,
              ),
              el('div', { className: 'ops-charts-toolbar__group' },
                el('p', { className: 'ops-charts-toolbar__label' }, 'Layout'),
                chartsLayoutMount,
              ),
            ),
          ),
          statusBarEl,
          el('div', {
            className: `ops-charts-grid ops-charts-grid--${chartsLayout}`,
            'data-ops-charts-grid': '',
          },
          ...metricSpecs.map((spec: any) =>
            el('article', { className: 'metric-chart-card' },
              el('div', { className: 'metric-chart-card__head' },
                el('h3', { className: 'metric-chart-card__title' }, spec.title),
                el('span', {
                  className: 'metric-chart-card__value',
                  style: `color: var(${spec.color})`,
                }, spec.displayValue ?? formatChartTick(spec.value)),
              ),
              el('div', {
                className: 'metric-chart-mount',
                'data-chart-id': spec.id,
              }),
            ),
          ),
          ),
        )
        : null,
      renderTabBar({ tabs: [
        { id: 'overview', label: 'Overview' },
        { id: 'outbox', label: 'Outbox' },
        ...(canShardsRead ? [{ id: 'dlq', label: 'DLQ' }] : []),
      ], active: tab, onChange: (t: any) => {
        tab = t;
        if (t === 'outbox') loadOutbox();
        if (t === 'dlq') loadDlq();
        render();
      } }),
      tab === 'overview' && canBillingRead && !state.loading
        ? el('section', {
          className: 'section-block',
          'data-testid': 'ops-billing-invariant',
        },
          el('div', { className: 'flex items-center gap-2 mb-3' },
            el('h2', { className: 'subsection-title' }, 'Billing invariant'),
            invariantState
              ? (invariantState.ok
                ? renderStatusBadge('ok', { kind: 'service', label: 'OK' })
                : renderStatusBadge('critical', { kind: 'service', label: 'Mismatch' }))
              : null,
          ),
          el('div', { className: 'filter-row mb-3' },
            el('input', {
              type: 'text',
              className: 'form-input form-input--sm',
              placeholder: 'Optional customer_id filter',
              value: invariantFilter,
              onChange: (e: Event) => {
                invariantFilter = (e.target as HTMLInputElement).value.trim();
              },
            }),
            renderButton({
              label: invariantLoading ? 'Checking…' : 'Check',
              variant: 'secondary',
              size: 'sm',
              loading: invariantLoading,
              disabled: invariantLoading,
              onClick: loadInvariant,
            }),
          ),
          invariantState
            ? el('dl', { className: 'definition-list' },
              el('dt', null, 'OK'),
              el('dd', null, formatYesNo(invariantState.ok)),
              invariantState.customer_id
                ? el('dt', null, 'Customer')
                : null,
              invariantState.customer_id
                ? el('dd', { className: 'font-mono' }, invariantState.customer_id)
                : null,
              invariantState.balance_micro != null
                ? el('dt', null, 'Wallet balance (micro)')
                : null,
              invariantState.balance_micro != null
                ? el('dd', { className: 'font-mono' }, formatAmountMicro(invariantState.balance_micro))
                : null,
              invariantState.ledger_sum_micro != null
                ? el('dt', null, 'Ledger sum (micro)')
                : null,
              invariantState.ledger_sum_micro != null
                ? el('dd', { className: 'font-mono' }, formatAmountMicro(invariantState.ledger_sum_micro))
                : null,
              invariantState.diff_micro != null
                ? el('dt', null, 'Diff (micro)')
                : null,
              invariantState.diff_micro != null
                ? el('dd', {
                  className: invariantState.ok ? 'font-mono' : 'font-mono text-danger',
                }, String(invariantState.diff_micro))
                : null,
              !invariantState.ok && invariantState.customer_id
                ? el('dd', { className: 'mt-2' },
                  el('a', {
                    href: `/billing?customer_id=${encodeURIComponent(invariantState.customer_id)}`,
                    className: 'text-sm',
                  }, 'Open customer billing →'),
                )
                : null,
            )
            : el('p', { className: 'text-muted text-sm' }, 'Run a check to compare wallet vs ledger totals.'),
        )
        : null,
      tab === 'overview' && !state.loading && state.summary
        ? renderDoctorPanel({
          doctor: state.doctor,
          services: state.summary.services,
          loading: false,
        })
        : null,
      tab === 'overview' && (state.slowApiPaths.length > 0 || state.rumEvents > 0)
        ? el('section', { className: 'section-block', 'data-testid': 'client-telemetry-panel' },
          el('h2', { className: 'subsection-title' }, 'Client telemetry'),
          state.slowApiPaths.length > 0
            ? renderStatusHint({
              tone: 'error',
              message: `Slow API (p95 ≥ 500 ms): ${state.slowApiPaths.join(', ')}`,
            })
            : null,
          state.rumEvents > 0
            ? el('p', { className: 'text-muted text-sm' },
              `${state.rumEvents} RUM sample(s) stored server-side`,
            )
            : null,
        )
        : null,
      tab === 'overview' && state.operatorDash
        ? el('div', { className: 'section-block stack stack--lg' },
          renderEdgePanel(state.operatorDash.edge),
          renderXDPPanel(state.operatorDash.xdp),
        )
        : null,
      tab === 'overview' && shardSnippet.length > 0
        ? el('section', { className: 'section-block' },
          el('div', { className: 'flex items-center gap-2 mb-4' },
            el('h2', { className: 'subsection-title' }, 'Shards'),
            el('a', {
              href: '/ops/shards',
              className: 'text-muted text-xs',
            }, 'All shards →'),
          ),
          el('div', { className: 'table-wrapper elevation-raised' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Shard'),
                  el('th', { scope: 'col' }, 'Ping'),
                  el('th', { scope: 'col' }, 'Lag'),
                ),
              ),
              el('tbody', null,
                shardSnippet.slice(0, 8).map((s: ShardHealthStatus) =>
                  el('tr', {
                    className: !s.ping_ok ? 'data-table__row--danger' : undefined,
                  },
                    el('td', null, String(s.shard_id)),
                    el('td', null, s.ping_ok ? 'OK' : displayLabel(s.ping_error ?? 'fail')),
                    el('td', null, String(s.config_version_lag ?? 0)),
                  ),
                ),
              ),
            ),
          ),
        )
        : null,
      tab === 'outbox'
        ? el('div', { className: 'section-block' },
          outboxToolbar,
          outboxLoading && outboxItems.length === 0
            ? el('span', { className: 'text-muted' }, 'Loading…')
            : null,
          el('div', { className: 'table-wrapper elevation-raised' },
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
                outboxItems.map((row: OutboxEventDTO) =>
                  el('tr', null,
                    el('td', { className: 'font-mono' }, row.id != null ? String(row.id) : ''),
                    el('td', null, displayLabel(row.event_type)),
                    el('td', null, row.status ?? ''),
                    el('td', { className: 'text-muted' },
                      row.created_at ? new Date(row.created_at).toLocaleString() : '—',
                    ),
                  ),
                ),
              ),
            ),
          ),
        )
        : null,
      tab === 'dlq'
        ? el('div', { className: 'section-block', 'data-testid': 'ops-dlq-tab' },
          dlqPartialErrors.length > 0
            ? el('div', { className: 'stub-banner mb-4' },
              `Partial shard errors: ${dlqPartialErrors.map((e) => `${e.source ?? '?'} (${e.code ?? 'err'})`).join('; ')}`,
            )
            : null,
          dlqLoading && dlqItems.length === 0
            ? el('span', { className: 'text-muted' }, 'Loading…')
            : null,
          dlqToolbar,
          el('div', { className: 'table-wrapper elevation-raised' },
            el('table', { className: 'data-table' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'ID'),
                  el('th', { scope: 'col' }, 'Shard'),
                  el('th', { scope: 'col' }, 'Stream'),
                  el('th', { scope: 'col' }, 'Entry'),
                  el('th', { scope: 'col' }, 'Campaign'),
                  el('th', { scope: 'col' }, 'Type'),
                  el('th', { scope: 'col' }, 'Error'),
                  el('th', { scope: 'col' }, 'Failed'),
                  el('th', { scope: 'col' }, 'Retries'),
                  canShardsWrite ? el('th', { scope: 'col' }, '') : null,
                ),
              ),
              el('tbody', null,
                dlqItems.length === 0 && !dlqLoading
                  ? el('tr', null,
                    renderEmptyTableCell(canShardsWrite ? 10 : 9, {
                      title: 'No DLQ entries',
                      description: 'Dead-letter queue is empty for the current shard filter.',
                    }),
                  )
                  : null,
                dlqItems.map((row: DLQEntryDTO) => el('tr', null,
                  el('td', { className: 'font-mono text-xs' }, row.id),
                  el('td', null, String(row.shard_id)),
                  el('td', { className: 'font-mono text-xs' }, row.stream_id),
                  el('td', { className: 'font-mono text-xs' }, row.entry_id),
                  el('td', { className: 'font-mono text-xs' }, row.campaign_id ?? '—'),
                  el('td', null, displayLabel(row.event_type ?? '')),
                  el('td', {
                    className: 'text-xs text-muted',
                    title: row.error ?? '',
                  }, row.error ? `${row.error.slice(0, 48)}${row.error.length > 48 ? '…' : ''}` : '—'),
                  el('td', { className: 'text-muted text-xs' },
                    row.failed_at ? new Date(row.failed_at).toLocaleString() : '—',
                  ),
                  el('td', null, String(row.retry_count ?? 0)),
                  canShardsWrite
                    ? el('td', null,
                      renderButton({
                        label: 'Retry',
                        variant: 'secondary',
                        size: 'sm',
                        testId: `ops-dlq-retry-${row.id}`,
                        loading: dlqLoading,
                        disabled: dlqLoading,
                        onClick: () => retryDlqEntry(row),
                      }),
                    )
                    : null,
                )),
              ),
            ),
          ),
        )
        : null,
    );
    mountOpsCharts();
    updateStatusBar();
  }

  async function loadOutbox() {
    const opGen = outboxGuard.next();
    outboxLoading = true;
    outboxItems = [];
    outboxCursor = '';
    render();
    const params = new URLSearchParams();
    if (outboxStatus) params.set('status', outboxStatus);
    const [outboxRes, outboxErr] = await to(api<OutboxListResponse>(`/api/v1/ops/outbox?${params.toString()}`));
    if (!shouldCommitAsyncResult(opGen, outboxGuard.current(), destroyed)) return;
    if (outboxErr) {
      const view = mapServiceError(outboxErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    } else {
      const data = outboxRes?.data ?? {};
      outboxItems = data.items ?? [];
      outboxCursor = data.next_cursor ?? '';
    }
    outboxLoading = false;
    if (!destroyed) render();
  }

  async function loadMoreOutbox() {
    if (!outboxCursor || outboxLoading) return;
    const opGen = outboxGuard.current();
    const cursorAtStart = outboxCursor;
    outboxLoading = true;
    render();
    const params = new URLSearchParams({ cursor: outboxCursor });
    if (outboxStatus) params.set('status', outboxStatus);
    const [outboxRes, outboxErr] = await to(api<OutboxListResponse>(`/api/v1/ops/outbox?${params.toString()}`));
    if (!shouldCommitAsyncResult(opGen, outboxGuard.current(), destroyed)
      || cursorAtStart !== outboxCursor) {
      outboxLoading = false;
      return;
    }
    if (outboxErr) {
      const view = mapServiceError(outboxErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    } else {
      const data = outboxRes?.data ?? {};
      outboxItems = [...outboxItems, ...(data.items ?? [])];
      outboxCursor = data.next_cursor ?? '';
    }
    outboxLoading = false;
    if (!destroyed) render();
  }

  async function loadDlq() {
    const opGen = dlqGuard.next();
    dlqLoading = true;
    dlqItems = [];
    dlqCursor = '';
    dlqPartialErrors = [];
    render();
    const [dlqRes, dlqErr] = await to(api<DLQListResponse>('/api/v1/ops/dlq?limit=50'));
    if (!shouldCommitAsyncResult(opGen, dlqGuard.current(), destroyed)) return;
    if (dlqErr) {
      if (dlqErr instanceof ApiError && dlqErr.status === 503 && dlqErr.payload) {
        const payload = dlqErr.payload as DLQListResponse;
        dlqItems = payload.items ?? [];
        dlqCursor = payload.next_cursor ?? '';
        dlqPartialErrors = payload.errors ?? [];
      } else {
        const view = mapServiceError(dlqErr);
        pushToastMessage({ title: view.title, message: view.message, code: view.code });
      }
    } else {
      const data = dlqRes?.data ?? {};
      dlqItems = data.items ?? [];
      dlqCursor = data.next_cursor ?? '';
      dlqPartialErrors = data.errors ?? [];
    }
    dlqLoading = false;
    if (!destroyed) render();
  }

  async function loadMoreDlq() {
    if (!dlqCursor || dlqLoading) return;
    const opGen = dlqGuard.current();
    const cursorAtStart = dlqCursor;
    dlqLoading = true;
    render();
    const [dlqRes, dlqErr] = await to(api<DLQListResponse>(`/api/v1/ops/dlq?limit=50&cursor=${encodeURIComponent(dlqCursor)}`));
    if (!shouldCommitAsyncResult(opGen, dlqGuard.current(), destroyed)
      || cursorAtStart !== dlqCursor) {
      dlqLoading = false;
      return;
    }
    if (dlqErr) {
      const view = mapServiceError(dlqErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    } else {
      const data = dlqRes?.data ?? {};
      dlqItems = [...dlqItems, ...(data.items ?? [])];
      dlqCursor = data.next_cursor ?? '';
      if (data.errors?.length) dlqPartialErrors = [...dlqPartialErrors, ...data.errors];
    }
    dlqLoading = false;
    if (!destroyed) render();
  }

  async function retryDlqEntry(row: DLQEntryDTO) {
    dlqLoading = true;
    render();
    const [, err] = await to(retryOpsDlq(row.id));
    dlqLoading = false;
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
    pushToastMessage({ title: 'Retry queued', message: row.id });
    loadDlq();
  }

  async function loadInvariant() {
    invariantLoading = true;
    render();
    const [data, err] = await to(fetchBillingInvariant(invariantFilter));
    invariantLoading = false;
    if (destroyed) return;
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    invariantState = data ?? null;
    render();
  }

  async function downloadBundle() {
    if (!bundleGate.tryAcquire()) return;
    render();
    await flushRUMNow();
    const [blob, blobErr] = await to(apiBlob('/api/v1/ops/support/bundle', { method: 'POST' }));
    bundleGate.release();
    if (destroyed) return;
    if (blobErr) {
      const view = mapServiceError(blobErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      render();
      return;
    }
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'espx-support-bundle.tar.gz';
    a.click();
    URL.revokeObjectURL(url);
    render();
  }

  async function loadOpsData(opts: any = {}) {
    const quiet = opts.quiet === true;
    if (!quiet) {
      state.loading = true;
      state.blockError = null;
    }

    const bundleCtrl = typeof AbortController !== 'undefined' ? new AbortController() : null;
    const bundleTimer = bundleCtrl ? window.setTimeout(() => bundleCtrl.abort(), 12_000) : 0;
    const signal = bundleCtrl?.signal;
    type OpsSlot = ApiResult | { error: unknown } | { data: unknown };
    const [results, err] = await to(parallelAll<OpsSlot>([
      () => api('/api/v1/ops/doctor', signal ? { signal } : {}),
      () => api('/api/v1/ops/incidents', signal ? { signal } : {}).catch((e: unknown) => ({ error: e })),
      () => api('/api/v1/ops/dashboard/summary', signal ? { signal } : {}),
      () => api('/api/v1/ops/rum', signal ? { signal } : {}).catch(() => ({ data: { events: [] } })),
      () => api('/api/v1/dashboards/operator', signal ? { signal } : {}).catch(() => ({ data: null })),
    ], 3));
    if (bundleTimer) window.clearTimeout(bundleTimer);

    if (destroyed) return;

    if (err) {
      if (!quiet) {
        state.blockError = err;
        state.loading = false;
        render();
      }
      return;
    }

    const [docRes, incRes, sumRes, rumRes, opDashRes] = results;
    if (!isParallelSlotError(docRes) && 'data' in docRes && docRes.data) {
      state.doctor = docRes.data as OpsDoctorSummary;
    }
    if (!isParallelSlotError(sumRes) && 'data' in sumRes && sumRes.data) {
      state.summary = sumRes.data as DashboardSummary;
    }
    state.operatorDash = (!isParallelSlotError(opDashRes) && 'data' in opDashRes
      ? opDashRes.data as OperatorDash | null
      : null) ?? null;
    const rumData = !isParallelSlotError(rumRes) && 'data' in rumRes
      ? rumRes.data as RumResponse | null
      : null;
    state.rumEvents = rumData?.events?.length ?? 0;
    state.slowApiPaths = apiTimingReport().slowPaths;

    const errors: PartialSourceError[] = [];
    if (isParallelSlotError(incRes) || ('error' in incRes && incRes.error)) {
      const incErr = (incRes as { error: unknown }).error;
      if (incErr instanceof ApiError && incErr.payload) {
        state.incidents = incErr.payload as IncidentSnapshot;
        const payloadErrors = (incErr.payload as IncidentSnapshot).errors;
        if (payloadErrors?.length) errors.push(...payloadErrors);
      } else if (!quiet) {
        const view = mapServiceError(incErr);
        if (view.kind === 'page' || view.kind === 'unavailable') {
          state.blockError = incErr;
        } else {
          pushToastMessage({ title: view.title, message: view.message, code: view.code });
        }
      }
    } else if (!isParallelSlotError(incRes) && 'data' in incRes && incRes.data) {
      state.incidents = incRes.data as IncidentSnapshot;
      const incData = incRes.data as IncidentSnapshot;
      if (incData.errors?.length) errors.push(...incData.errors);
    }
    state.partialErrors = errors;

    recordSnapshotMetrics();

    // Paint Ops shell first — metric series can be slow/unavailable (e.g. CH off).
    if (!quiet) state.loading = false;
    if (destroyed) return;
    markRefreshed();
    if (quiet) {
      mountOpsCharts({ reuse: true });
    } else {
      render();
    }

    try {
      await loadMetricSeries();
    } catch {
      // Charts stay on snapshot series when history metrics fail.
    }
    if (destroyed) return;
    if (tab === 'overview') {
      mountOpsCharts({ reuse: true });
    }
  }

  /**
   * Apply a streamed dashboard summary without a full reload.
   *
   * @param {object} summary
   * @param {string} [generatedAt]
   */
  async function applyStreamSummary(summary: any, generatedAt: any) {
    if (!summary || destroyed) return;
    state.summary = summary;
    recordSnapshotMetrics();
    if (generatedAt) {
      const ts = Date.parse(generatedAt);
      if (Number.isFinite(ts)) state.lastUpdatedAt = ts;
    } else {
      state.lastUpdatedAt = Date.now();
    }
    updateStatusBar();
    if (tab === 'overview' && !state.loading) {
      mountOpsCharts({ reuse: true });
    }
  }

  render();
  loadOpsData();

  liveFeed = connectOpsLiveFeed({
    pollMs: OPS_POLL_MS,
    onModeChange: (mode) => {
      state.feedMode = mode;
      updateStatusBar();
    },
    onTick: (payload) => {
      if (destroyed) return;
      if (payload.source === 'stream' && payload.summary) {
        applyStreamSummary(payload.summary, payload.generatedAt);
      }
    },
    onPoll: () => {
      if (!destroyed && tab === 'overview') loadOpsData({ quiet: true });
    },
  });

  clockTimer = setInterval(() => {
    if (!destroyed) updateStatusBar();
  }, 1000);

  return {
    destroy() {
      destroyed = true;
      if (clockTimer) clearInterval(clockTimer);
      clockTimer = null;
      liveFeed?.destroy();
      liveFeed = null;
      statusBarEl = null;
      destroyCharts();
      outboxGuard.invalidate();
      dlqGuard.invalidate();
      bundleGate.release();
    },
  };
}
