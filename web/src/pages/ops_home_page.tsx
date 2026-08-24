import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type {
  BillingInvariantDTO,
  DashboardSummary,
  DLQEntryDTO,
  IncidentSnapshot,
  OpsDoctorSummary,
  OutboxEventDTO,
  OutboxListResponse,
  ShardHealthStatus,
} from '../types/index.js';
import type { DLQListResponse } from '../types/ops_extra.js';
import { to } from '../lib/to.js';
import { api, ApiError, type ApiResult } from '../helpers/api_client.js';
import { isParallelSlotError, parallelAll } from '../helpers/request_multiplex.js';
import { apiBlob } from '../helpers/api_blob.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import {
  createGenerationGuard,
  createInFlightGuard,
  shouldCommitAsyncResult,
} from '../lib/async_guard.js';
import { apiTimingReport } from '../helpers/api_timing.js';
import { flushRUMNow } from '../helpers/rum_collector.js';
import { displayLabel, formatYesNo } from '../helpers/display_labels.js';
import * as storage from '../helpers/storage.js';
import type { OpsChartsLayout, OpsChartsRangeHours } from '../helpers/storage.js';
import {
  OPS_METRIC_API_NAMES,
  parseApiPoints,
  recordSnapshot,
  toRateSeries,
  type ApiMetricRow,
  type MetricPoint,
} from '../helpers/ops_metric_series.js';
import { connectOpsLiveFeed } from '../helpers/ops_live_feed.js';
import { fetchBillingInvariant } from '../helpers/billing_admin_api.js';
import { fetchOpsDlqPage, isOpsDlqEntryRetryable, retryOpsDlq } from '../helpers/ops_dlq_api.js';
import { reloadRoles } from '../helpers/ops_compliance_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { formatAmountMicro } from '../helpers/money.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Button } from '../components/button.js';
import { DoctorPanel } from '../components/doctor_panel.js';
import {
  EdgePanel,
  XDPPanel,
  type EdgePanelData,
  type XDPPanelData,
} from '../components/edge_panel.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { Icon } from '../components/icon.js';
import {
  buildOpsMetricSpecs,
  OpsMetricCharts,
  type OperatorDashCharts,
} from '../components/ops_metric_charts.js';
import { StatusBadge } from '../components/status_badge.js';
import { StatusHint } from '../components/status_hint.js';
import { TabBar } from '../components/tab_bar.js';

type PartialSourceError = { source?: string; code?: string };

type OperatorDash = OperatorDashCharts & Record<string, unknown>;

type MetricsSeriesResponse = {
  points?: ApiMetricRow[];
};

type RumResponse = {
  events?: unknown[];
};

type OpsTab = 'overview' | 'outbox' | 'dlq';

const OPS_POLL_MS = 30_000;

function recordOpsSnapshotMetrics(
  summary: DashboardSummary | null,
  operatorDash: OperatorDash | null
): void {
  if (summary) {
    recordSnapshot('outbox-pending', Number(summary.outbox_pending) || 0);
    recordSnapshot('rps-estimate', Number(summary.rps_estimate) || 0);
    recordSnapshot('drift-alert', Number(summary.drift_micro_max) || 0);
    recordSnapshot(
      'emergency-breaker',
      String(summary.emergency_breaker).toLowerCase() === 'open' ? 1 : 0
    );
  }
  const edge = operatorDash?.edge;
  if (edge) {
    recordSnapshot('ingress-h1', Number(edge.ingress_h1) || 0);
    recordSnapshot('ingress-h2', Number(edge.ingress_h2) || 0);
    recordSnapshot('ingress-h3', Number(edge.ingress_h3) || 0);
    recordSnapshot('edge-tarpit', Number(edge.tarpit_total) || 0);
    recordSnapshot('edge-blacklist-stale', Number(edge.blacklist_stale) || 0);
    recordSnapshot('edge-fraud-tier', Number(edge.blocked?.fraud_tier) || 0);
  }
  const drops = operatorDash?.xdp?.drops;
  if (drops && typeof drops === 'object') {
    for (const key of Object.keys(drops).sort()) {
      recordSnapshot(`drop-${key}`, Number(drops[key]) || 0);
    }
  }
}

function OpsEdgeSection({ operatorDash }: { operatorDash: OperatorDash | null }) {
  if (!operatorDash) return null;
  return (
    <div className="section-block stack stack--lg">
      <EdgePanel edge={operatorDash.edge as EdgePanelData | undefined} />
      <XDPPanel xdp={operatorDash.xdp as XDPPanelData | undefined} />
    </div>
  );
}

export function OpsHomePage() {
  const user = auth.getUser();
  const canBundle = can(user?.permissions ?? [], 'ops:write');
  const canShardsWrite = can(user?.permissions ?? [], 'shards:write');
  const canShardsRead = can(user?.permissions ?? [], 'shards:read');
  const canBillingRead = can(user?.permissions ?? [], 'customers:read');
  const canSettingsWrite = can(user?.permissions ?? [], 'settings:write');

  const [tab, setTab] = useState<OpsTab>('overview');
  const [loading, setLoading] = useState(true);
  const [blockError, setBlockError] = useState<unknown>(null);
  const [doctor, setDoctor] = useState<OpsDoctorSummary | null>(null);
  const [incidents, setIncidents] = useState<IncidentSnapshot | null>(null);
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [partialErrors, setPartialErrors] = useState<PartialSourceError[]>([]);
  const [operatorDash, setOperatorDash] = useState<OperatorDash | null>(null);
  const [rumEvents, setRumEvents] = useState(0);
  const [slowApiPaths, setSlowApiPaths] = useState<string[]>([]);
  const [metricSeries, setMetricSeries] = useState<Record<string, MetricPoint[]>>({});
  const [lastUpdatedAt, setLastUpdatedAt] = useState(0);
  const [nextRefreshAt, setNextRefreshAt] = useState(0);
  const [feedMode, setFeedMode] = useState('poll');
  const [chartsLayout, setChartsLayout] = useState<OpsChartsLayout>(() =>
    storage.getOpsChartsLayout()
  );
  const [chartsRangeHours, setChartsRangeHours] = useState<OpsChartsRangeHours>(() =>
    storage.getOpsChartsRangeHours()
  );
  const [clockTick, setClockTick] = useState(0);

  const [outboxStatus, setOutboxStatus] = useState('');
  const [outboxItems, setOutboxItems] = useState<OutboxEventDTO[]>([]);
  const [outboxCursor, setOutboxCursor] = useState('');
  const [outboxLoading, setOutboxLoading] = useState(false);

  const [dlqItems, setDlqItems] = useState<DLQEntryDTO[]>([]);
  const [dlqCursor, setDlqCursor] = useState('');
  const [dlqLoading, setDlqLoading] = useState(false);
  const [dlqPartialErrors, setDlqPartialErrors] = useState<PartialSourceError[]>([]);

  const [invariantFilter, setInvariantFilter] = useState('');
  const [invariantState, setInvariantState] = useState<BillingInvariantDTO | null>(null);
  const [invariantLoading, setInvariantLoading] = useState(false);

  const outboxGuardRef = useRef(createGenerationGuard());
  const dlqGuardRef = useRef(createGenerationGuard());
  const bundleGateRef = useRef(createInFlightGuard());
  const [bundleBusy, setBundleBusy] = useState(false);
  const [rolesReloading, setRolesReloading] = useState(false);
  const destroyedRef = useRef(false);
  const tabRef = useRef(tab);
  tabRef.current = tab;
  const operatorDashRef = useRef(operatorDash);
  operatorDashRef.current = operatorDash;
  const summaryRef = useRef(summary);
  summaryRef.current = summary;

  const markRefreshed = useCallback(() => {
    const now = Date.now();
    setLastUpdatedAt(now);
    setNextRefreshAt(now + OPS_POLL_MS);
  }, []);

  const loadMetricSeries = useCallback(async (rangeHours: number) => {
    const ids = Object.keys(OPS_METRIC_API_NAMES);
    const ctrl = typeof AbortController !== 'undefined' ? new AbortController() : null;
    const timer = ctrl ? window.setTimeout(() => ctrl.abort(), 8_000) : 0;
    try {
      const results = await Promise.all(
        ids.map(async (id) => {
          const name = OPS_METRIC_API_NAMES[id];
          const [res] = await to(
            api(
              `/api/v1/ops/dashboard/metrics?range=${rangeHours}h&name=${encodeURIComponent(name)}`,
              ctrl ? { signal: ctrl.signal } : {}
            )
          );
          let points = parseApiPoints(
            (res?.data as MetricsSeriesResponse | undefined)?.points ??
              (res as MetricsSeriesResponse | null)?.points
          );
          if (id === 'rps-estimate') points = toRateSeries(points);
          return { id, points };
        })
      );

      if (destroyedRef.current) return;
      setMetricSeries((prev) => {
        const next = { ...prev };
        for (let i = 0; i < results.length; i++) {
          const { id, points } = results[i];
          if (points.length > 0) next[id] = points;
        }
        return next;
      });
    } finally {
      if (timer) window.clearTimeout(timer);
    }
  }, []);

  const loadOpsData = useCallback(
    async (opts: { quiet?: boolean } = {}) => {
      const quiet = opts.quiet === true;
      if (!quiet) {
        setLoading(true);
        setBlockError(null);
      }

      const bundleCtrl = typeof AbortController !== 'undefined' ? new AbortController() : null;
      const bundleTimer = bundleCtrl ? window.setTimeout(() => bundleCtrl.abort(), 12_000) : 0;
      const signal = bundleCtrl?.signal;
      type OpsSlot = ApiResult | { error: unknown } | { data: unknown };
      const [results, err] = await to(
        parallelAll<OpsSlot>(
          [
            () => api('/api/v1/ops/doctor', signal ? { signal } : {}),
            () =>
              api('/api/v1/ops/incidents', signal ? { signal } : {}).catch((e: unknown) => ({
                error: e,
              })),
            () => api('/api/v1/ops/dashboard/summary', signal ? { signal } : {}),
            () =>
              api('/api/v1/ops/rum', signal ? { signal } : {}).catch(() => ({
                data: { events: [] },
              })),
            () =>
              api('/api/v1/dashboards/operator', signal ? { signal } : {}).catch(() => ({
                data: null,
              })),
          ],
          3
        )
      );
      if (bundleTimer) window.clearTimeout(bundleTimer);

      if (destroyedRef.current) return;

      if (err) {
        if (!quiet) {
          setBlockError(err);
          setLoading(false);
        }
        return;
      }

      const [docRes, incRes, sumRes, rumRes, opDashRes] = results;

      if (!isParallelSlotError(docRes) && 'data' in docRes && docRes.data) {
        setDoctor(docRes.data as OpsDoctorSummary);
      }
      if (!isParallelSlotError(sumRes) && 'data' in sumRes && sumRes.data) {
        setSummary(sumRes.data as DashboardSummary);
      }
      const nextOperatorDash =
        (!isParallelSlotError(opDashRes) && 'data' in opDashRes
          ? (opDashRes.data as OperatorDash | null)
          : null) ?? null;
      setOperatorDash(nextOperatorDash);
      operatorDashRef.current = nextOperatorDash;

      const rumData =
        !isParallelSlotError(rumRes) && 'data' in rumRes
          ? (rumRes.data as RumResponse | null)
          : null;
      setRumEvents(rumData?.events?.length ?? 0);
      setSlowApiPaths(apiTimingReport().slowPaths);

      const errors: PartialSourceError[] = [];
      if (isParallelSlotError(incRes) || ('error' in incRes && incRes.error)) {
        const incErr = (incRes as { error: unknown }).error;
        if (incErr instanceof ApiError && incErr.payload) {
          setIncidents(incErr.payload as IncidentSnapshot);
          const payloadErrors = (incErr.payload as IncidentSnapshot).errors;
          if (payloadErrors?.length) errors.push(...payloadErrors);
        } else if (!quiet) {
          const view = mapServiceError(incErr);
          if (view.kind === 'page' || view.kind === 'unavailable') {
            setBlockError(incErr);
            setLoading(false);
            return;
          }
          pushToastMessage({ title: view.title, message: view.message, code: view.code });
        }
      } else if (!isParallelSlotError(incRes) && 'data' in incRes && incRes.data) {
        setIncidents(incRes.data as IncidentSnapshot);
        const incData = incRes.data as IncidentSnapshot;
        if (incData.errors?.length) errors.push(...incData.errors);
      }
      setPartialErrors(errors);

      const nextSummary =
        !isParallelSlotError(sumRes) && 'data' in sumRes && sumRes.data
          ? (sumRes.data as DashboardSummary)
          : summaryRef.current;
      recordOpsSnapshotMetrics(nextSummary, nextOperatorDash);

      if (!quiet) setLoading(false);
      markRefreshed();

      try {
        await loadMetricSeries(chartsRangeHours);
      } catch {}
    },
    [chartsRangeHours, loadMetricSeries, markRefreshed]
  );

  const applyStreamSummary = useCallback(
    (streamSummary: DashboardSummary, generatedAt?: string) => {
      setSummary(streamSummary);
      recordOpsSnapshotMetrics(streamSummary, operatorDashRef.current);
      if (generatedAt) {
        const ts = Date.parse(generatedAt);
        if (Number.isFinite(ts)) setLastUpdatedAt(ts);
      } else {
        setLastUpdatedAt(Date.now());
      }
    },
    []
  );

  const loadOpsDataRef = useRef(loadOpsData);
  loadOpsDataRef.current = loadOpsData;

  useEffect(() => {
    destroyedRef.current = false;
    void loadOpsData();

    const liveFeed = connectOpsLiveFeed({
      pollMs: OPS_POLL_MS,
      onModeChange: (mode) => setFeedMode(mode),
      onTick: (payload) => {
        if (destroyedRef.current) return;
        if (payload.source === 'stream' && payload.summary) {
          applyStreamSummary(payload.summary as DashboardSummary, payload.generatedAt);
        }
      },
      onPoll: () => {
        if (!destroyedRef.current && tabRef.current === 'overview') {
          void loadOpsDataRef.current({ quiet: true });
        }
      },
    });

    const clockTimer = setInterval(() => setClockTick((t) => t + 1), 1000);

    return () => {
      destroyedRef.current = true;
      liveFeed.destroy();
      clearInterval(clockTimer);
      outboxGuardRef.current.invalidate();
      dlqGuardRef.current.invalidate();
      bundleGateRef.current.release();
    };
  }, []); 

  useEffect(() => {
    void loadMetricSeries(chartsRangeHours);
  }, [chartsRangeHours, loadMetricSeries]);

  const loadOutbox = useCallback(
    async (statusOverride?: string) => {
      const status = statusOverride ?? outboxStatus;
      const opGen = outboxGuardRef.current.next();
      setOutboxLoading(true);
      setOutboxItems([]);
      setOutboxCursor('');
      const params = new URLSearchParams();
      if (status) params.set('status', status);
      const [outboxRes, outboxErr] = await to(
        api<OutboxListResponse>(`/api/v1/ops/outbox?${params.toString()}`)
      );
      if (!shouldCommitAsyncResult(opGen, outboxGuardRef.current.current(), destroyedRef.current))
        return;
      if (outboxErr) {
        const view = mapServiceError(outboxErr);
        pushToastMessage({ title: view.title, message: view.message, code: view.code });
      } else {
        const data = outboxRes?.data ?? {};
        setOutboxItems(data.items ?? []);
        setOutboxCursor(data.next_cursor ?? '');
      }
      setOutboxLoading(false);
    },
    [outboxStatus]
  );

  const loadMoreOutbox = useCallback(async () => {
    if (!outboxCursor || outboxLoading) return;
    const opGen = outboxGuardRef.current.current();
    const cursorAtStart = outboxCursor;
    setOutboxLoading(true);
    const params = new URLSearchParams({ cursor: outboxCursor });
    if (outboxStatus) params.set('status', outboxStatus);
    const [outboxRes, outboxErr] = await to(
      api<OutboxListResponse>(`/api/v1/ops/outbox?${params.toString()}`)
    );
    if (
      !shouldCommitAsyncResult(opGen, outboxGuardRef.current.current(), destroyedRef.current) ||
      cursorAtStart !== outboxCursor
    ) {
      setOutboxLoading(false);
      return;
    }
    if (outboxErr) {
      const view = mapServiceError(outboxErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    } else {
      const data = outboxRes?.data ?? {};
      setOutboxItems((prev) => [...prev, ...(data.items ?? [])]);
      setOutboxCursor(data.next_cursor ?? '');
    }
    setOutboxLoading(false);
  }, [outboxCursor, outboxLoading, outboxStatus]);

  const loadDlq = useCallback(async () => {
    const opGen = dlqGuardRef.current.next();
    setDlqLoading(true);
    setDlqItems([]);
    setDlqCursor('');
    setDlqPartialErrors([]);
    const page = await fetchOpsDlqPage();
    if (!shouldCommitAsyncResult(opGen, dlqGuardRef.current.current(), destroyedRef.current))
      return;
    if (page.error) {
      const view = mapServiceError(page.error);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    } else {
      setDlqItems(page.items);
      setDlqCursor(page.nextCursor);
      setDlqPartialErrors(page.partialErrors);
    }
    setDlqLoading(false);
  }, []);

  const loadMoreDlq = useCallback(async () => {
    if (!dlqCursor || dlqLoading) return;
    const opGen = dlqGuardRef.current.current();
    const cursorAtStart = dlqCursor;
    setDlqLoading(true);
    const page = await fetchOpsDlqPage(dlqCursor);
    if (
      !shouldCommitAsyncResult(opGen, dlqGuardRef.current.current(), destroyedRef.current) ||
      cursorAtStart !== dlqCursor
    ) {
      setDlqLoading(false);
      return;
    }
    if (page.error) {
      if (page.error instanceof ApiError && page.error.status === 503 && page.error.payload) {
        const payload = page.error.payload as DLQListResponse;
        setDlqItems((prev) => [...prev, ...(payload.items ?? [])]);
        setDlqCursor(payload.next_cursor ?? dlqCursor);
        setDlqPartialErrors((prev) => [...prev, ...(payload.errors ?? [])]);
      } else {
        const view = mapServiceError(page.error);
        pushToastMessage({ title: view.title, message: view.message, code: view.code });
      }
    } else {
      setDlqItems((prev) => [...prev, ...page.items]);
      setDlqCursor(page.nextCursor);
      if (page.partialErrors.length > 0) {
        setDlqPartialErrors((prev) => [...prev, ...page.partialErrors]);
      }
    }
    setDlqLoading(false);
  }, [dlqCursor, dlqLoading]);

  const retryDlqEntry = useCallback(
    async (row: DLQEntryDTO) => {
      setDlqLoading(true);
      const [, err] = await to(retryOpsDlq(row.id));
      setDlqLoading(false);
      if (err) {
        if (err instanceof ConfirmCancelledError) return;
        const view = mapServiceError(err);
        pushToastMessage({ title: view.title, message: view.message, code: view.code });
        return;
      }
      pushToastMessage({ title: 'Retry queued', message: row.id });
      void loadDlq();
    },
    [loadDlq]
  );

  const loadInvariant = useCallback(async () => {
    setInvariantLoading(true);
    const [data, err] = await to(fetchBillingInvariant(invariantFilter));
    setInvariantLoading(false);
    if (destroyedRef.current) return;
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    setInvariantState(data ?? null);
  }, [invariantFilter]);

  const downloadBundle = useCallback(async () => {
    if (!bundleGateRef.current.tryAcquire()) return;
    setBundleBusy(true);
    await flushRUMNow();
    const [blob, blobErr] = await to(apiBlob('/api/v1/ops/support/bundle', { method: 'POST' }));
    bundleGateRef.current.release();
    setBundleBusy(false);
    if (destroyedRef.current) return;
    if (blobErr) {
      const view = mapServiceError(blobErr);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'ad-event-processor-support-bundle.tar.gz';
    a.click();
    URL.revokeObjectURL(url);
  }, []);

  const handleTabChange = (next: string) => {
    const t = next as OpsTab;
    setTab(t);
    if (t === 'outbox') void loadOutbox();
    if (t === 'dlq') void loadDlq();
  };

  const metricSpecs = useMemo(
    () => buildOpsMetricSpecs(summary, operatorDash, metricSeries, chartsRangeHours),
    [summary, operatorDash, metricSeries, chartsRangeHours]
  );

  const shardSnippet = incidents?.shards ?? [];

  void clockTick;

  if (blockError) {
    return <ErrorBlock error={blockError} />;
  }

  return (
    <>
      <div className="page-header">
        <div className="page-header__row">
          <h1 className="page-header__title">Operations</h1>
          {doctor ? (
            <StatusBadge
              status={doctor.overall}
              kind="service"
              label={`Doctor: ${displayLabel(doctor.overall)}`}
            />
          ) : null}
          {canBundle ? (
            <Button
              label="Support bundle"
              variant="secondary"
              size="sm"
              icon="download"
              className="ml-auto"
              loading={bundleBusy}
              disabled={bundleBusy}
              onClick={() => void downloadBundle()}
            />
          ) : null}
        </div>
      </div>

      {partialErrors.length > 0 ? (
        <AlertBanner
          variant="warning"
          dismissKey="ops.partial"
          message={`Partial source errors: ${partialErrors.map((e) => `${e.source ?? '?'} (${e.code ?? 'err'})`).join('; ')}`}
        />
      ) : null}

      {incidents?.stale_dashboard ? (
        <section className="section-block" data-testid="ops-stale-dashboard">
          <AlertBanner
            variant="warning"
            dismissKey="ops.stale-dashboard"
            message="ClickHouse dashboards are stale - campaign KPIs may lag behind Postgres."
          />
          {incidents.affected_campaigns && incidents.affected_campaigns.length > 0 ? (
            <div className="mt-3">
              <h2 className="subsection-title">Affected campaigns</h2>
              <ul className="text-sm">
                {incidents.affected_campaigns.slice(0, 12).map((c) => (
                  <li key={c.campaign_id}>
                    <a
                      href={`/campaigns/${encodeURIComponent(c.campaign_id)}`}
                      className="font-mono"
                    >
                      {c.name ? `${c.name} (${c.campaign_id})` : c.campaign_id}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </section>
      ) : null}

      {loading ? <span className="text-muted">Loading...</span> : null}

      {!loading && summary ? (
        <div className="ops-kpi-strip section-block">
          {summary.drift_alert ? (
            <div className="ops-alert-bar" role="status">
              <Icon name="alert-triangle" size={16} className="ops-alert-bar__icon" />
              <span className="ops-alert-bar__label">Drift alert</span>
              <span className="ops-alert-bar__value">{formatYesNo(true)}</span>
            </div>
          ) : null}
          <div className="ops-kpi-row">
            <div className="ops-kpi-chip">
              <span className="ops-kpi-chip__label">Outbox pending</span>
              <span className="ops-kpi-chip__value">{String(summary.outbox_pending)}</span>
            </div>
            <div className="ops-kpi-chip">
              <span className="ops-kpi-chip__label">RPS estimate</span>
              <span className="ops-kpi-chip__value">{summary.rps_estimate?.toFixed(1) ?? '-'}</span>
            </div>
            <div className="ops-kpi-chip">
              <span className="ops-kpi-chip__label">Emergency breaker</span>
              <span className="ops-kpi-chip__value">{displayLabel(summary.emergency_breaker)}</span>
            </div>
          </div>
        </div>
      ) : null}

      {tab === 'overview' && !loading && metricSpecs.length > 0 ? (
        <div className="section-block">
          <OpsMetricCharts
            specs={metricSpecs}
            chartsLayout={chartsLayout}
            chartsRangeHours={chartsRangeHours}
            lastUpdatedAt={lastUpdatedAt}
            nextRefreshAt={nextRefreshAt}
            feedMode={feedMode}
            onLayoutChange={(layout) => {
              const next: OpsChartsLayout = layout === 'stack' ? 'stack' : 'grid';
              setChartsLayout(next);
              storage.setOpsChartsLayout(next);
            }}
            onRangeChange={(hours) => {
              const next = hours as OpsChartsRangeHours;
              setChartsRangeHours(next);
              storage.setOpsChartsRangeHours(next);
            }}
          />
        </div>
      ) : null}

      <TabBar
        tabs={[
          { id: 'overview', label: 'Overview' },
          { id: 'outbox', label: 'Outbox' },
          ...(canShardsRead ? [{ id: 'dlq', label: 'DLQ' }] : []),
        ]}
        active={tab}
        onChange={handleTabChange}
      />

      {tab === 'overview' && canBillingRead && !loading ? (
        <section className="section-block" data-testid="ops-billing-invariant">
          <div className="flex items-center gap-2 mb-3">
            <h2 className="subsection-title">Billing invariant</h2>
            {invariantState ? (
              invariantState.ok ? (
                <span data-testid="ops-billing-invariant-ok">
                  <StatusBadge status="ok" kind="service" label="OK" />
                </span>
              ) : (
                <span data-testid="ops-billing-invariant-mismatch">
                  <StatusBadge status="critical" kind="service" label="Mismatch" />
                </span>
              )
            ) : null}
          </div>
          <div className="filter-row mb-3">
            <input
              type="text"
              className="form-input form-input--sm"
              placeholder="Optional customer_id filter"
              value={invariantFilter}
              onChange={(e) => setInvariantFilter(e.target.value.trim())}
            />
            <Button
              label={invariantLoading ? 'Checking...' : 'Check'}
              variant="secondary"
              size="sm"
              loading={invariantLoading}
              disabled={invariantLoading}
              onClick={() => void loadInvariant()}
            />
          </div>
          {invariantState ? (
            <dl className="definition-list">
              <dt>OK</dt>
              <dd>{formatYesNo(invariantState.ok)}</dd>
              {invariantState.customer_id ? (
                <>
                  <dt>Customer</dt>
                  <dd className="font-mono">{invariantState.customer_id}</dd>
                </>
              ) : null}
              {invariantState.balance_micro != null ? (
                <>
                  <dt>Wallet balance (micro)</dt>
                  <dd className="font-mono">{formatAmountMicro(invariantState.balance_micro)}</dd>
                </>
              ) : null}
              {invariantState.ledger_sum_micro != null ? (
                <>
                  <dt>Ledger balance (micro)</dt>
                  <dd className="font-mono">
                    {formatAmountMicro(invariantState.ledger_sum_micro)}
                  </dd>
                </>
              ) : null}
              {invariantState.diff_micro != null ? (
                <>
                  <dt>Diff (micro)</dt>
                  <dd className={invariantState.ok ? 'font-mono' : 'font-mono text-danger'}>
                    {String(invariantState.diff_micro)}
                  </dd>
                </>
              ) : null}
              {invariantState.fleet_scan_limit != null ? (
                <>
                  <dt>Fleet scan</dt>
                  <dd className="text-muted text-sm" data-testid="ops-billing-invariant-fleet-cap">
                    {`Scanned first ${invariantState.fleet_scan_limit} customers (omit customer_id filter for fleet scan).`}
                  </dd>
                </>
              ) : null}
              {!invariantState.ok && invariantState.customer_id ? (
                <dd className="mt-2 flex gap-3">
                  <a
                    href={`/billing?customer_id=${encodeURIComponent(invariantState.customer_id)}`}
                    className="text-sm"
                    data-testid="ops-invariant-billing-link"
                  >
                    Open billing {'->'}
                  </a>
                  <a
                    href={`/billing?customer_id=${encodeURIComponent(invariantState.customer_id)}&tab=ledger`}
                    className="text-sm"
                    data-testid="ops-invariant-ledger-link"
                  >
                    Open ledger {'->'}
                  </a>
                </dd>
              ) : null}
            </dl>
          ) : (
            <p className="text-muted text-sm">Run a check to compare wallet vs ledger totals.</p>
          )}
        </section>
      ) : null}

      {tab === 'overview' && !loading && summary ? (
        <DoctorPanel doctor={doctor} services={summary.services} loading={false} />
      ) : null}

      {tab === 'overview' && (slowApiPaths.length > 0 || rumEvents > 0) ? (
        <section className="section-block" data-testid="client-telemetry-panel">
          <h2 className="subsection-title">Client telemetry</h2>
          {slowApiPaths.length > 0 ? (
            <StatusHint
              tone="error"
              message={`Slow API (p95 >= 500 ms): ${slowApiPaths.join(', ')}`}
            />
          ) : null}
          {rumEvents > 0 ? (
            <p className="text-muted text-sm">{`${rumEvents} RUM sample(s) stored server-side`}</p>
          ) : null}
        </section>
      ) : null}

      {tab === 'overview' ? <OpsEdgeSection operatorDash={operatorDash} /> : null}

      {tab === 'overview' && shardSnippet.length > 0 ? (
        <section className="section-block">
          <div className="flex items-center gap-2 mb-4">
            <h2 className="subsection-title">Shards</h2>
            <a href="/ops/shards" className="text-muted text-xs">
              All shards {'->'}
            </a>
          </div>
          <div className="table-wrapper elevation-raised">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Shard</th>
                  <th scope="col">Ping</th>
                  <th scope="col">Lag</th>
                </tr>
              </thead>
              <tbody>
                {shardSnippet.slice(0, 8).map((s: ShardHealthStatus) => (
                  <tr
                    key={s.shard_id}
                    className={!s.ping_ok ? 'data-table__row--danger' : undefined}
                  >
                    <td>{String(s.shard_id)}</td>
                    <td>{s.ping_ok ? 'OK' : displayLabel(s.ping_error ?? 'fail')}</td>
                    <td>{String(s.config_version_lag ?? 0)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      {tab === 'overview' && canSettingsWrite ? (
        <section className="section-block" data-testid="ops-danger-zone">
          <h2 className="subsection-title">Danger zone</h2>
          <p className="text-muted text-sm mb-3">
            Reload RBAC role definitions from disk. Requires operator confirmation.
          </p>
          <Button
            label={rolesReloading ? 'Reloading...' : 'Reload RBAC'}
            variant="secondary"
            size="sm"
            icon="refresh-cw"
            data-testid="roles-reload"
            loading={rolesReloading}
            disabled={rolesReloading}
            onClick={() =>
              void (async () => {
                setRolesReloading(true);
                try {
                  const res = await reloadRoles();
                  pushToastMessage({
                    title: 'RBAC reloaded',
                    message: res.path ? `Loaded ${res.path}` : res.status,
                  });
                } catch (e) {
                  if (e instanceof ConfirmCancelledError) return;
                  pushToastMessage({ title: 'Reload failed', message: mapServiceError(e).message });
                } finally {
                  setRolesReloading(false);
                }
              })()
            }
          />
        </section>
      ) : null}

      {tab === 'outbox' ? (
        <div className="section-block">
          <div className="mb-4">
            <FilterToolbar
              leading={
                <div className="cluster cluster--sm items-center">
                  <span className="text-muted text-sm">Status</span>
                  <select
                    className="form-input form-input--sm min-w-40"
                    aria-label="Outbox status"
                    value={outboxStatus}
                    onChange={(e) => {
                      const value = e.target.value;
                      setOutboxStatus(value);
                      void loadOutbox(value);
                    }}
                  >
                    <option value="">All</option>
                    <option value="pending">Pending</option>
                    <option value="processed">Processed</option>
                    <option value="failed">Failed</option>
                  </select>
                </div>
              }
              pagination={
                outboxCursor ? (
                  <Button
                    label="Load more"
                    variant="secondary"
                    size="sm"
                    loading={outboxLoading}
                    disabled={outboxLoading}
                    onClick={() => void loadMoreOutbox()}
                  />
                ) : undefined
              }
            />
          </div>
          {outboxLoading && outboxItems.length === 0 ? (
            <span className="text-muted">Loading...</span>
          ) : null}
          <div className="table-wrapper elevation-raised">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">ID</th>
                  <th scope="col">Type</th>
                  <th scope="col">Status</th>
                  <th scope="col">Created</th>
                </tr>
              </thead>
              <tbody>
                {outboxItems.map((row) => (
                  <tr key={String(row.id)}>
                    <td className="font-mono">{row.id != null ? String(row.id) : ''}</td>
                    <td>{displayLabel(row.event_type)}</td>
                    <td>{row.status ?? ''}</td>
                    <td className="text-muted">
                      {row.created_at ? new Date(row.created_at).toLocaleString() : '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      {tab === 'dlq' ? (
        <div className="section-block" data-testid="ops-dlq-tab">
          <div className="row gap-sm items-center mb-4">
            <a href="/ops/dlq" className="text-muted text-sm" data-testid="ops-dlq-full-inbox-link">
              Full DLQ inbox {'->'}
            </a>
          </div>
          {dlqPartialErrors.length > 0 ? (
            <div className="stub-banner mb-4">
              {`Partial shard errors: ${dlqPartialErrors.map((e) => `${e.source ?? '?'} (${e.code ?? 'err'})`).join('; ')}`}
            </div>
          ) : null}
          {dlqLoading && dlqItems.length === 0 ? (
            <span className="text-muted">Loading...</span>
          ) : null}
          {dlqCursor ? (
            <div className="mb-4">
              <FilterToolbar
                pagination={
                  <Button
                    label="Load more"
                    variant="secondary"
                    size="sm"
                    loading={dlqLoading}
                    disabled={dlqLoading}
                    onClick={() => void loadMoreDlq()}
                  />
                }
              />
            </div>
          ) : null}
          <div className="table-wrapper elevation-raised">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">ID</th>
                  <th scope="col">Shard</th>
                  <th scope="col">Stream</th>
                  <th scope="col">Entry</th>
                  <th scope="col">Campaign</th>
                  <th scope="col">Type</th>
                  <th scope="col">Error</th>
                  <th scope="col">Failed</th>
                  <th scope="col">Retries</th>
                  {canShardsWrite ? <th scope="col" /> : null}
                </tr>
              </thead>
              <tbody>
                {dlqItems.length === 0 && !dlqLoading ? (
                  <tr>
                    <td colSpan={canShardsWrite ? 10 : 9}>
                      <div className="empty-state">
                        <div className="empty-state__title">No DLQ entries</div>
                        <div className="empty-state__desc text-muted text-sm">
                          Dead-letter queue is empty for the current shard filter.
                        </div>
                      </div>
                    </td>
                  </tr>
                ) : null}
                {dlqItems.map((row) => (
                  <tr key={row.id}>
                    <td className="font-mono text-xs">{row.id}</td>
                    <td>{String(row.shard_id)}</td>
                    <td className="font-mono text-xs">{row.stream_id}</td>
                    <td className="font-mono text-xs">{row.entry_id}</td>
                    <td className="font-mono text-xs">{row.campaign_id ?? '-'}</td>
                    <td>{displayLabel(row.event_type ?? '')}</td>
                    <td className="text-xs text-muted" title={row.error ?? ''}>
                      {row.error
                        ? `${row.error.slice(0, 48)}${row.error.length > 48 ? '...' : ''}`
                        : '-'}
                    </td>
                    <td className="text-muted text-xs">
                      {row.failed_at ? new Date(row.failed_at).toLocaleString() : '-'}
                    </td>
                    <td>{String(row.retry_count ?? 0)}</td>
                    {canShardsWrite ? (
                      <td>
                        {isOpsDlqEntryRetryable(row) ? (
                          <Button
                            label="Retry"
                            variant="secondary"
                            size="sm"
                            data-testid={`ops-dlq-retry-${row.id}`}
                            loading={dlqLoading}
                            disabled={dlqLoading}
                            onClick={() => void retryDlqEntry(row)}
                          />
                        ) : null}
                      </td>
                    ) : null}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}
    </>
  );
}
