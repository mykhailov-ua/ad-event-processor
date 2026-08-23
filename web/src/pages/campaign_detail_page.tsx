import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import type { CampaignDTO, ClickDeliveryMode } from '../types/campaign.js';
import type { ReportRow } from '../types/report.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { can, maskLevel } from '../helpers/permissions.js';
import { pauseCampaign, resumeCampaign, pollCampaignStatus } from '../helpers/campaign_actions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { formatUsdDecimal, ParseDecimal } from '../helpers/money.js';
import { shortCustomerId, touchCustomerContext } from '../helpers/customer_context.js';
import { PacingHealth } from '../components/pacing_health.js';
import { estimateDeliveryPct } from '../models/buyer.js';
import { CampaignHourlyChart } from '../components/campaign_hourly_chart.js';
import { ForecastModal, useForecastModal } from '../components/forecast_modal.js';
import { isoDaysAgo, toIsoNow } from '../helpers/date_presets.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import type { MetricsBlockDTO } from '../types/metrics.js';
import { patchCampaign } from '../helpers/campaign_admin_api.js';
import { fetchFlows, type FlowDTO } from '../helpers/flows_api.js';
import { displayLabel } from '../helpers/display_labels.js';
import type { HourlyMetricRow } from '../helpers/chart_pool.js';
import { CampaignFiltersSection } from '../components/campaign_filters_section.js';
import { CampaignTrackingSection } from '../components/campaign_tracking_section.js';
import { CampaignPostbackSection } from '../components/campaign_postback_section.js';
import { CampaignFraudSection } from '../components/campaign_fraud_section.js';
import { CampaignMarginGuardSection } from '../components/campaign_margin_guard_section.js';
import { CampaignTelegramSection } from '../components/campaign_telegram_section.js';
import { CampaignBrandCreativesSection } from '../components/campaign_brand_creatives_section.js';
import { CampaignOwnerSection } from '../components/campaign_owner_section.js';
import { CommercialMetrics } from '../components/commercial_metrics.js';
import { FreshnessBadge } from '../components/freshness_badge.js';
import { Breadcrumbs, type BreadcrumbItem } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { Icon } from '../components/icon.js';
import { PaginationBar } from '../components/pagination_bar.js';
import { StatusBadge } from '../components/status_badge.js';
import { TabBar, type TabBarTab } from '../components/tab_bar.js';
import { useResource } from '../helpers/use_resource.js';

type CampaignStatsDTO = {
  metrics?: {
    impressions?: number;
    clicks?: number;
    conversions?: number;
  };
  current_spend?: string;
  stale?: boolean;
  hourly?: unknown[];
};

type CampaignDashboardDTO = {
  kpis?: MetricsBlockDTO;
};

type CampaignEventsResponse = {
  items?: ReportRow[];
  total?: number;
};

const EVENTS_PAGE_SIZE = 50;

function toDatetimeLocalValue(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function datetimeLocalToISO(local: string): string | undefined {
  const trimmed = local.trim();
  if (!trimmed) return undefined;
  const d = new Date(trimmed);
  if (Number.isNaN(d.getTime())) return undefined;
  return d.toISOString();
}

function formatDaypartHours(hours: number[]): string {
  if (!hours?.length) return '';
  return [...hours].sort((a, b) => a - b).join(',');
}

function normalizeClickDelivery(raw?: string): ClickDeliveryMode {
  return raw === 'proxy' ? 'proxy' : 'redirect';
}

function clickDeliveryLabel(mode: ClickDeliveryMode): string {
  return mode === 'proxy' ? 'Reverse proxy' : 'Redirect';
}

function parseDaypartHours(raw: string): number[] {
  if (!raw.trim()) return [];
  const out: number[] = [];
  for (const part of raw.split(',')) {
    const h = Number.parseInt(part.trim(), 10);
    if (!Number.isFinite(h) || h < 0 || h > 23) {
      throw new Error('invalid daypart hour');
    }
    out.push(h);
  }
  return [...new Set(out)].sort((a, b) => a - b);
}

function allowedTabIds(masked: boolean): string[] {
  const list = ['overview', 'stats', 'config'];
  if (!masked) {
    list.push(
      'tracking',
      'postbacks',
      'fraud',
      'filters',
      'margin',
      'events',
      'creative',
      'telegram'
    );
  }
  return list;
}

function resolveTab(requested: string | null, masked: boolean): string {
  const tab = requested?.trim() ?? '';
  return allowedTabIds(masked).includes(tab) ? tab : 'overview';
}

function buildTabs(masked: boolean): TabBarTab[] {
  const list: TabBarTab[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'stats', label: 'Statistics' },
    { id: 'config', label: 'Configuration' },
  ];
  if (!masked) {
    list.push(
      { id: 'tracking', label: 'Integration' },
      { id: 'postbacks', label: 'CAPI & Postbacks' },
      { id: 'fraud', label: 'Fraud' },
      { id: 'filters', label: 'Filters' },
      { id: 'margin', label: 'Margin guard' },
      { id: 'events', label: 'Event log' },
      { id: 'creative', label: 'Creative' },
      { id: 'telegram', label: 'Telegram' }
    );
  }
  return list;
}

function ConfigGrid({ rows }: { rows: Array<[string, string]> }) {
  return (
    <dl className="definition-list">
      {rows.flatMap(([label, value]) => [
        <dt key={`${label}-dt`}>{label}</dt>,
        <dd key={`${label}-dd`} className="font-mono text-secondary">
          {value}
        </dd>,
      ])}
    </dl>
  );
}

function EventsTableSkeleton() {
  return (
    <>
      {Array.from({ length: 4 }, (_, i) => (
        <tr key={`ev-skel-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: 4 }, (__, j) => (
            <td key={`ev-skel-${i}-${j}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function CampaignDetailPage() {
  const { id = '' } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();

  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const masked = maskLevel(permissions) === 'masked';
  const canPause = can(permissions, 'campaigns:write') || can(permissions, 'campaigns:pause');
  const canWriteCampaign = can(permissions, 'campaigns:write');

  const tab = resolveTab(searchParams.get('tab'), masked);
  const actionGateRef = useRef(createInFlightGuard());
  const { forecastOpen, forecastOpts, openForecast, closeForecast } = useForecastModal();

  const [actionLoading, setActionLoading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [eventsPage, setEventsPage] = useState(0);
  const [eventsRows, setEventsRows] = useState<ReportRow[]>([]);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [configSaving, setConfigSaving] = useState(false);
  const [configError, setConfigError] = useState<string | null>(null);
  const [dashboard, setDashboard] = useState<{
    data: CampaignDashboardDTO | null;
    loading: boolean;
    error: unknown;
  }>({ data: null, loading: true, error: null });

  const [configForm, setConfigForm] = useState({
    name: '',
    pacing_mode: 'ASAP',
    timezone: 'UTC',
    budget_limit: '',
    status: 'ACTIVE',
    start_at: '',
    end_at: '',
    daypart_hours: '',
    daily_budget: '',
    target_url: '',
    geo: '',
    freq_limit: '0',
    freq_window: '86400',
    safe_page_enabled: false,
    safe_page_url: '',
    attestation_enabled: false,
    attestation_ttl_sec: '300',
    dmr_enabled: false,
    click_delivery: 'redirect' as ClickDeliveryMode,
    proxy_upstream_url: '',
    proxy_rewrite_assets: false,
    l1_cidr_block_enabled: true,
    l15_proxy_vpn_block_enabled: true,
    tls_fingerprint_block_enabled: true,
    conn_type_policy: 'block_vpn_hosting',
    link_signing_enabled: false,
    link_signing_ttl_sec: '900',
    flow_id: '',
  });

  const [flowOptions, setFlowOptions] = useState<FlowDTO[]>([]);

  const statsUrl = useMemo(() => {
    const params = new URLSearchParams({ granularity: 'hour' });
    if (masked) {
      params.set('from', isoDaysAgo(7));
      params.set('to', toIsoNow());
    }
    return `/api/v1/campaigns/${id}/stats?${params.toString()}`;
  }, [id, masked]);

  const {
    data: campaign,
    loading: campaignLoading,
    error: campaignError,
    reload: reloadCampaign,
  } = useResource<CampaignDTO>(id ? `/api/v1/campaigns/${id}` : null);

  const {
    data: stats,
    loading: statsLoading,
    error: statsError,
    reload: reloadStats,
  } = useResource<CampaignStatsDTO>(statsUrl, {
    skip: !masked && tab !== 'stats',
  });

  const campaignSyncedRef = useRef<string | null>(null);
  useEffect(() => {
    if (!campaign || campaignSyncedRef.current === campaign.id) return;
    campaignSyncedRef.current = campaign.id;
    if (campaign.customer_id) touchCustomerContext(campaign.customer_id);
    setConfigForm({
      name: campaign.name ?? '',
      pacing_mode: campaign.pacing_mode ?? 'ASAP',
      timezone: campaign.timezone ?? 'UTC',
      budget_limit: campaign.budget_limit ?? '',
      status: (campaign.status ?? 'ACTIVE').toUpperCase(),
      start_at: toDatetimeLocalValue(campaign.start_at),
      end_at: toDatetimeLocalValue(campaign.end_at),
      daypart_hours: formatDaypartHours(campaign.daypart_hours ?? []),
      daily_budget: campaign.daily_budget ?? '',
      target_url: campaign.target_url ?? '',
      geo: (campaign.target_countries ?? []).join(','),
      freq_limit: String(campaign.freq_limit ?? 0),
      freq_window: String(campaign.freq_window ?? 86400),
      safe_page_enabled: campaign.safe_page_enabled === true,
      safe_page_url: campaign.safe_page_url ?? '',
      attestation_enabled: campaign.attestation_enabled === true,
      attestation_ttl_sec: String(campaign.attestation_ttl_sec ?? 300),
      dmr_enabled: campaign.dmr_enabled === true,
      click_delivery: normalizeClickDelivery(campaign.click_delivery),
      proxy_upstream_url: campaign.proxy_upstream_url ?? '',
      proxy_rewrite_assets: campaign.proxy_rewrite_assets === true,
      l1_cidr_block_enabled: campaign.l1_cidr_block_enabled !== false,
      l15_proxy_vpn_block_enabled: campaign.l15_proxy_vpn_block_enabled !== false,
      tls_fingerprint_block_enabled: campaign.tls_fingerprint_block_enabled !== false,
      conn_type_policy: campaign.conn_type_policy ?? 'block_vpn_hosting',
      link_signing_enabled: campaign.link_signing_enabled === true,
      link_signing_ttl_sec: String(campaign.link_signing_ttl_sec ?? 900),
      flow_id: campaign.flow_id ?? '',
    });
  }, [campaign]);

  useEffect(() => {
    if (!canWriteCampaign || masked || tab !== 'config') return;
    void fetchFlows()
      .then(setFlowOptions)
      .catch(() => setFlowOptions([]));
  }, [canWriteCampaign, masked, tab]);

  useEffect(() => {
    if (!id) return undefined;
    let cancelled = false;
    setDashboard({ data: null, loading: true, error: null });
    void (async () => {
      const [res, err] = await to(api(`/api/v1/dashboards/campaign/${id}`));
      if (cancelled) return;
      if (err) {
        setDashboard({ data: null, loading: false, error: err });
        return;
      }
      setDashboard({
        data: (res?.data as CampaignDashboardDTO | null) ?? null,
        loading: false,
        error: null,
      });
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  const setTab = useCallback(
    (nextTab: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextTab === 'overview') next.delete('tab');
      else next.set('tab', nextTab);
      setSearchParams(next, { replace: true });
      if (nextTab === 'stats') reloadStats();
      if (nextTab === 'events') setEventsPage(0);
    },
    [searchParams, setSearchParams, reloadStats]
  );

  const loadEvents = useCallback(
    async (page: number) => {
      if (!id) return;
      setEventsLoading(true);
      const limit = EVENTS_PAGE_SIZE;
      const offset = page * limit;
      const [res, err] = await to(
        api(`/api/v1/campaigns/${id}/events?limit=${limit}&offset=${offset}`)
      );
      setEventsLoading(false);
      if (err) {
        setEventsRows([]);
        setEventsTotal(0);
        return;
      }
      const data = (res?.data ?? {}) as CampaignEventsResponse;
      setEventsRows(data.items ?? []);
      setEventsTotal(data.total ?? 0);
    },
    [id]
  );

  useEffect(() => {
    if (tab !== 'events' || masked) return;
    void loadEvents(eventsPage);
  }, [tab, eventsPage, masked, loadEvents]);

  const overviewImpressions7d = Number(stats?.metrics?.impressions ?? 0);

  const handlePause = async () => {
    const gate = actionGateRef.current;
    if (!gate.tryAcquire()) return;
    setActionLoading(true);
    setActionError(null);
    const [, pauseErr] = await to(pauseCampaign(id));
    if (pauseErr) {
      if (pauseErr instanceof ConfirmCancelledError) {
        setActionLoading(false);
        gate.release();
        return;
      }
      setActionError(pauseErr.message || 'Failed to pause campaign');
      setActionLoading(false);
      gate.release();
      return;
    }
    const [, pollErr] = await to(pollCampaignStatus(id, 'PAUSED'));
    if (pollErr) {
      setActionError(pollErr.message || 'Failed to pause campaign');
    } else {
      reloadCampaign();
    }
    setActionLoading(false);
    gate.release();
  };

  const handleResume = async () => {
    const gate = actionGateRef.current;
    if (!gate.tryAcquire()) return;
    setActionLoading(true);
    setActionError(null);
    const [, resumeErr] = await to(resumeCampaign(id));
    if (resumeErr) {
      if (resumeErr instanceof ConfirmCancelledError) {
        setActionLoading(false);
        gate.release();
        return;
      }
      setActionError(resumeErr.message || 'Failed to resume campaign');
      setActionLoading(false);
      gate.release();
      return;
    }
    const [, pollErr] = await to(pollCampaignStatus(id, 'ACTIVE'));
    if (pollErr) {
      setActionError(pollErr.message || 'Failed to resume campaign');
    } else {
      reloadCampaign();
    }
    setActionLoading(false);
    gate.release();
  };

  const saveConfig = async () => {
    if (!canWriteCampaign || configSaving || !campaign) return;
    const body: Record<string, unknown> = {
      name: configForm.name.trim(),
      pacing_mode: configForm.pacing_mode,
      timezone: configForm.timezone.trim(),
    };
    if (!String(body.name ?? '').trim()) {
      setConfigError('Name is required');
      return;
    }
    if (configForm.daily_budget.trim()) {
      try {
        body.daily_budget_micro = ParseDecimal(configForm.daily_budget.trim());
      } catch {
        setConfigError('Invalid daily budget');
        return;
      }
    }
    if (configForm.budget_limit.trim()) {
      try {
        body.budget_limit_micro = ParseDecimal(configForm.budget_limit.trim());
      } catch {
        setConfigError('Invalid total budget');
        return;
      }
    }
    if (configForm.status === 'ACTIVE' || configForm.status === 'PAUSED') {
      body.status = configForm.status.toLowerCase();
    }
    const startISO = datetimeLocalToISO(configForm.start_at);
    const endISO = datetimeLocalToISO(configForm.end_at);
    if (startISO) body.start_at = startISO;
    if (endISO) body.end_at = endISO;
    try {
      body.daypart_hours = parseDaypartHours(configForm.daypart_hours);
    } catch {
      setConfigError('Daypart hours must be 0–23, comma-separated');
      return;
    }
    const url = configForm.target_url.trim();
    if (url && !/^https?:\/\//i.test(url)) {
      setConfigError('Target URL must start with http:// or https://');
      return;
    }
    body.target_url = url;
    body.target_countries = configForm.geo.trim()
      ? configForm.geo
          .split(',')
          .map((c) => c.trim().toUpperCase())
          .filter(Boolean)
      : [];
    const freqLimit = Number.parseInt(configForm.freq_limit, 10);
    if (Number.isFinite(freqLimit) && freqLimit >= 0) body.freq_limit = freqLimit;
    const freqWindow = Number.parseInt(configForm.freq_window, 10);
    if (Number.isFinite(freqWindow) && freqWindow > 0) body.freq_window = freqWindow;
    body.safe_page_enabled = configForm.safe_page_enabled;
    body.safe_page_url = configForm.safe_page_url.trim();
    body.attestation_enabled = configForm.attestation_enabled;
    const attTTL = Number.parseInt(configForm.attestation_ttl_sec, 10);
    if (Number.isFinite(attTTL)) body.attestation_ttl_sec = attTTL;
    body.dmr_enabled = configForm.dmr_enabled;
    const clickDelivery = normalizeClickDelivery(configForm.click_delivery);
    body.click_delivery = clickDelivery;
    if (clickDelivery === 'proxy') {
      const proxyURL = configForm.proxy_upstream_url.trim();
      if (!proxyURL) {
        setConfigError('Proxy upstream URL is required when click delivery is reverse proxy');
        return;
      }
      if (!/^https?:\/\//i.test(proxyURL)) {
        setConfigError('Proxy upstream URL must start with http:// or https://');
        return;
      }
      body.proxy_upstream_url = proxyURL;
      body.proxy_rewrite_assets = configForm.proxy_rewrite_assets;
    }
    body.tls_fingerprint_block_enabled = configForm.tls_fingerprint_block_enabled;
    body.l1_cidr_block_enabled = configForm.l1_cidr_block_enabled;
    body.l15_proxy_vpn_block_enabled = configForm.l15_proxy_vpn_block_enabled;
    body.conn_type_policy = configForm.conn_type_policy;
    body.link_signing_enabled = configForm.link_signing_enabled;
    const linkTTL = Number.parseInt(configForm.link_signing_ttl_sec, 10);
    if (Number.isFinite(linkTTL) && linkTTL >= 60 && linkTTL <= 3600) {
      body.link_signing_ttl_sec = linkTTL;
    }
    body.flow_id = configForm.flow_id.trim()
      ? configForm.flow_id.trim()
      : '00000000-0000-0000-0000-000000000000';

    setConfigSaving(true);
    setConfigError(null);
    const [, err] = await to(patchCampaign(id, body));
    setConfigSaving(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setConfigError(err.message || 'Save failed');
      return;
    }
    pushToastMessage({
      title: 'Campaign saved',
      message: 'Config propagates to trackers within ~60s.',
    });
    reloadCampaign();
  };

  const linkCampaignBrand = async (brandId: string) => {
    if (!canWriteCampaign || !brandId.trim()) return;
    const [, err] = await to(patchCampaign(id, { brand_id: brandId.trim() }));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({
        title: 'Brand link failed',
        message: err.message || 'Could not link brand to campaign',
      });
      return;
    }
    pushToastMessage({
      title: 'Brand linked',
      message: 'Campaign uses this brand for weighted creatives. Tracker sync ~60s.',
    });
    reloadCampaign();
  };

  if (campaignLoading && !campaign) {
    return (
      <div className="grid-stats section-block">
        {['Status', 'Budget', 'Spend', 'Pacing'].map((label) => (
          <div key={label} className="metric-card metric-card--loading">
            <div className="metric-card__label">{label}</div>
            <div className="metric-card__value">…</div>
          </div>
        ))}
      </div>
    );
  }

  if (campaignError) {
    return <ErrorBlock error={campaignError} />;
  }

  if (!campaign) return null;

  const status = (campaign.status || '').toUpperCase();
  const isPaused = status === 'PAUSED';
  const isActive = status === 'ACTIVE';

  const crumbs: BreadcrumbItem[] = [{ label: 'Campaigns', href: '/campaigns' }];
  if (campaign.customer_id) {
    crumbs.push({
      label: shortCustomerId(campaign.customer_id, 12),
      href: `/customers/${campaign.customer_id}`,
    });
  }
  crumbs.push({ label: String(campaign.name ?? '') });

  const eventsPageCount = Math.ceil(eventsTotal / EVENTS_PAGE_SIZE);

  return (
    <>
      <div className="page-header">
        <Breadcrumbs items={crumbs} />
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="megaphone" size={20} className="text-muted" />
            <h1 className="page-header__title">{campaign.name}</h1>
          </div>
          <StatusBadge status={campaign.status} />
          {canPause ? (
            <div className="cluster--actions ml-auto">
              {isActive ? (
                <Button
                  label="Pause"
                  variant="danger"
                  size="sm"
                  icon="pause"
                  loading={actionLoading}
                  disabled={actionLoading}
                  onClick={() => void handlePause()}
                />
              ) : null}
              {isPaused ? (
                <Button
                  label="Resume"
                  variant="primary"
                  size="sm"
                  icon="play"
                  loading={actionLoading}
                  disabled={actionLoading}
                  onClick={() => void handleResume()}
                />
              ) : null}
            </div>
          ) : null}
        </div>
        {actionError ? <p className="text-danger text-sm mt-2">{actionError}</p> : null}
      </div>

      <TabBar tabs={buildTabs(masked)} active={tab} onChange={setTab} />

      {tab === 'overview' ? (
        <div className="section-block stack">
          {dashboard.loading ? <span className="text-muted">Loading economics…</span> : null}
          {!dashboard.loading ? (
            <CommercialMetrics kpis={dashboard.data?.kpis} masked={masked} />
          ) : null}
          {!masked && campaign ? (
            <Button
              label="Forecast delivery"
              variant="secondary"
              size="sm"
              className="shrink-0"
              onClick={() =>
                openForecast({
                  campaignId: id,
                  customerId: campaign.customer_id,
                  budgetMicro: Math.round(Number(campaign.budget_limit ?? 0) * 1_000_000),
                  startAt: isoDaysAgo(0),
                  endAt: toIsoNow(),
                })
              }
            />
          ) : null}
          {masked && campaign ? (
            <PacingHealth
              status={campaign.status}
              pacingMode={campaign.pacing_mode}
              impressions7d={overviewImpressions7d}
              deliveryPct={estimateDeliveryPct(overviewImpressions7d, campaign.status)}
            />
          ) : (
            <div className="grid-stats section-block">
              <div className="metric-card">
                <div className="metric-card__label">Budget limit</div>
                <div className="metric-card__value font-mono">
                  {formatUsdDecimal(campaign.budget_limit ?? '0.00')}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-card__label">Current spend</div>
                <div className="metric-card__value font-mono">
                  {formatUsdDecimal(campaign.current_spend ?? '0.00')}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-card__label">Daily budget</div>
                <div className="metric-card__value font-mono">
                  {formatUsdDecimal(campaign.daily_budget ?? '0.00')}
                </div>
              </div>
              <div className="metric-card">
                <div className="metric-card__label">Pacing</div>
                <div className="metric-card__value">{displayLabel(campaign.pacing_mode)}</div>
              </div>
            </div>
          )}
        </div>
      ) : null}

      {tab === 'stats' ? (
        <div className="section-block stack">
          {statsLoading ? <span className="text-muted">Loading statistics…</span> : null}
          {statsError ? (
            <p className="text-danger text-sm">
              {statsError instanceof Error ? statsError.message : String(statsError)}
            </p>
          ) : null}
          {stats ? (
            <div className="stack">
              <div className="flex items-center gap-2">
                <h2 className="subsection-title">Hourly metrics</h2>
                <FreshnessBadge stale={stats.stale} />
              </div>
              <div className="metric-row">
                <div className="metric-card">
                  <div className="metric-card__label">Impressions</div>
                  <div className="metric-card__value">
                    {String(stats.metrics?.impressions ?? 0)}
                  </div>
                </div>
                <div className="metric-card">
                  <div className="metric-card__label">Clicks</div>
                  <div className="metric-card__value">{String(stats.metrics?.clicks ?? 0)}</div>
                </div>
                <div className="metric-card">
                  <div className="metric-card__label">Conversions</div>
                  <div className="metric-card__value">
                    {String(stats.metrics?.conversions ?? 0)}
                  </div>
                </div>
                <div className="metric-card">
                  <div className="metric-card__label">Spend (API)</div>
                  <div className="metric-card__value font-mono">
                    {masked ? '—' : formatUsdDecimal(stats.current_spend ?? '0.00')}
                  </div>
                </div>
              </div>
              <div className="section-card">
                <h3 className="subsection-title">Hourly trend</h3>
                <CampaignHourlyChart hourly={(stats.hourly ?? []) as HourlyMetricRow[]} />
              </div>
            </div>
          ) : null}
        </div>
      ) : null}

      {tab === 'config' ? (
        <div className="section-block stack">
          {campaign?.customer_id && canWriteCampaign && !masked ? (
            <CampaignOwnerSection
              campaignId={id}
              customerId={campaign.customer_id}
              ownerUserId={campaign.owner_user_id ?? ''}
              canWrite={canWriteCampaign}
              onAssigned={() => reloadCampaign()}
            />
          ) : null}
          {canWriteCampaign && !masked ? (
            <div className="section-card stack">
              <h3 className="subsection-title">Edit settings</h3>
              {configError ? <p className="text-danger text-sm">{configError}</p> : null}
              <h4 className="subsection-title">Budget &amp; schedule</h4>
              <label className="form-field" htmlFor="cfg-budget-total">
                Total budget (USD)
                <input
                  id="cfg-budget-total"
                  className="form-input form-input--sm"
                  inputMode="decimal"
                  placeholder="0.00"
                  value={configForm.budget_limit}
                  data-testid="campaign-budget-total"
                  onChange={(e) => setConfigForm((f) => ({ ...f, budget_limit: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="cfg-status">
                Status
                <select
                  id="cfg-status"
                  className="form-input form-input--sm"
                  value={configForm.status}
                  data-testid="cfg-status"
                  onChange={(e) => setConfigForm((f) => ({ ...f, status: e.target.value }))}
                >
                  <option value="ACTIVE">Active</option>
                  <option value="PAUSED">Paused</option>
                </select>
              </label>
              <div className="filter-row">
                <label className="form-field" htmlFor="cfg-start-at">
                  Start
                  <input
                    id="cfg-start-at"
                    className="form-input form-input--sm"
                    type="datetime-local"
                    value={configForm.start_at}
                    data-testid="cfg-start-at"
                    onChange={(e) => setConfigForm((f) => ({ ...f, start_at: e.target.value }))}
                  />
                </label>
                <label className="form-field" htmlFor="cfg-end-at">
                  End
                  <input
                    id="cfg-end-at"
                    className="form-input form-input--sm"
                    type="datetime-local"
                    value={configForm.end_at}
                    data-testid="cfg-end-at"
                    onChange={(e) => setConfigForm((f) => ({ ...f, end_at: e.target.value }))}
                  />
                </label>
              </div>
              <label className="form-field" htmlFor="cfg-daypart">
                Daypart hours (0–23, comma-separated; empty = all hours)
                <input
                  id="cfg-daypart"
                  className="form-input form-input--sm"
                  placeholder="9,10,11,12"
                  value={configForm.daypart_hours}
                  data-testid="cfg-daypart"
                  onChange={(e) => setConfigForm((f) => ({ ...f, daypart_hours: e.target.value }))}
                />
              </label>
              <h4 className="subsection-title">Delivery</h4>
              <label className="form-field" htmlFor="cfg-name">
                Name
                <input
                  id="cfg-name"
                  className="form-input"
                  value={configForm.name}
                  onChange={(e) => setConfigForm((f) => ({ ...f, name: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="cfg-pacing">
                Pacing
                <select
                  id="cfg-pacing"
                  className="form-input form-input--sm"
                  value={configForm.pacing_mode}
                  onChange={(e) => setConfigForm((f) => ({ ...f, pacing_mode: e.target.value }))}
                >
                  <option value="ASAP">ASAP</option>
                  <option value="EVEN">Even</option>
                  <option value="VPP">VPP</option>
                </select>
              </label>
              <label className="form-field" htmlFor="cfg-timezone">
                Timezone
                <input
                  id="cfg-timezone"
                  className="form-input form-input--sm"
                  value={configForm.timezone}
                  onChange={(e) => setConfigForm((f) => ({ ...f, timezone: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="cfg-daily-budget">
                Daily budget (USD)
                <input
                  id="cfg-daily-budget"
                  className="form-input form-input--sm"
                  inputMode="decimal"
                  placeholder="0.00"
                  value={configForm.daily_budget}
                  data-testid="cfg-daily-budget"
                  onChange={(e) => setConfigForm((f) => ({ ...f, daily_budget: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="cfg-target-url">
                Target URL
                <input
                  id="cfg-target-url"
                  className="form-input"
                  type="url"
                  placeholder="https://landing.example/"
                  value={configForm.target_url}
                  data-testid="cfg-target-url"
                  onChange={(e) => setConfigForm((f) => ({ ...f, target_url: e.target.value }))}
                />
              </label>
              <label className="form-field" htmlFor="cfg-geo">
                Target countries (ISO, comma-separated)
                <input
                  id="cfg-geo"
                  className="form-input form-input--sm"
                  placeholder="US,GB,DE or empty for all"
                  value={configForm.geo}
                  data-testid="cfg-geo"
                  onChange={(e) => setConfigForm((f) => ({ ...f, geo: e.target.value }))}
                />
              </label>
              <div className="filter-row">
                <label className="form-field" htmlFor="cfg-freq-limit">
                  Freq limit
                  <input
                    id="cfg-freq-limit"
                    className="form-input form-input--sm"
                    type="number"
                    min={0}
                    value={configForm.freq_limit}
                    data-testid="cfg-freq-limit"
                    onChange={(e) => setConfigForm((f) => ({ ...f, freq_limit: e.target.value }))}
                  />
                </label>
                <label className="form-field" htmlFor="cfg-freq-window">
                  Freq window (sec)
                  <input
                    id="cfg-freq-window"
                    className="form-input form-input--sm"
                    type="number"
                    min={1}
                    value={configForm.freq_window}
                    data-testid="cfg-freq-window"
                    onChange={(e) => setConfigForm((f) => ({ ...f, freq_window: e.target.value }))}
                  />
                </label>
              </div>
              <label className="form-field" htmlFor="cfg-flow">
                Flow routing
                <select
                  id="cfg-flow"
                  className="form-input form-input--sm"
                  value={configForm.flow_id}
                  data-testid="campaign-flow-select"
                  onChange={(e) => setConfigForm((f) => ({ ...f, flow_id: e.target.value }))}
                >
                  <option value="">None (brand creatives)</option>
                  {flowOptions.map((flow) => (
                    <option key={flow.id} value={flow.id}>
                      {flow.name}
                    </option>
                  ))}
                </select>
              </label>
              <p className="text-muted text-sm">
                <a href="/campaigns/flows">Manage landers, offers &amp; flows →</a>
              </p>
              <div className="section-card stack" data-testid="campaign-safe-page-config">
                <h4 className="subsection-title">Safe page (cloak companion)</h4>
                <p className="text-muted text-sm">
                  When enabled, suspicious clicks (IVT / placement blacklist) redirect to the safe
                  URL instead of the money landing. Clean traffic uses brand creatives as usual.
                </p>
                <label className="form-field checkbox-field" htmlFor="cfg-safe-page-enabled">
                  <input
                    id="cfg-safe-page-enabled"
                    type="checkbox"
                    checked={configForm.safe_page_enabled}
                    onChange={(e) =>
                      setConfigForm((f) => ({ ...f, safe_page_enabled: e.target.checked }))
                    }
                  />{' '}
                  Enable safe-page redirect
                </label>
                <label className="form-field" htmlFor="cfg-safe-page-url">
                  Safe page URL
                  <input
                    id="cfg-safe-page-url"
                    className="form-input"
                    type="url"
                    placeholder="https://safe.example/white-page"
                    value={configForm.safe_page_url}
                    onChange={(e) =>
                      setConfigForm((f) => ({ ...f, safe_page_url: e.target.value }))
                    }
                  />
                </label>
                <label className="form-field checkbox-field" htmlFor="cfg-attestation-enabled">
                  <input
                    id="cfg-attestation-enabled"
                    type="checkbox"
                    checked={configForm.attestation_enabled}
                    disabled={!configForm.safe_page_enabled}
                    data-testid="cfg-attestation-enabled"
                    onChange={(e) =>
                      setConfigForm((f) => ({ ...f, attestation_enabled: e.target.checked }))
                    }
                  />{' '}
                  Require L2 attestation cookie (RP-M2)
                </label>
                {configForm.attestation_enabled ? (
                  <label className="form-field" htmlFor="cfg-attestation-ttl">
                    Attestation cookie TTL (seconds)
                    <input
                      id="cfg-attestation-ttl"
                      className="form-input"
                      type="number"
                      min={60}
                      max={900}
                      value={configForm.attestation_ttl_sec}
                      data-testid="cfg-attestation-ttl"
                      onChange={(e) =>
                        setConfigForm((f) => ({ ...f, attestation_ttl_sec: e.target.value }))
                      }
                    />
                  </label>
                ) : null}
              </div>
              <div className="section-card stack" data-testid="campaign-click-delivery-config">
                <h4 className="subsection-title">Click delivery (reverse proxy)</h4>
                <p className="text-muted text-sm">
                  Default redirect sends the browser to the landing URL. Reverse proxy mode fetches
                  the upstream through the tracker edge (RP-M3).
                </p>
                <label className="form-field" htmlFor="cfg-click-delivery">
                  Delivery mode
                  <select
                    id="cfg-click-delivery"
                    className="form-input"
                    value={configForm.click_delivery}
                    data-testid="cfg-click-delivery"
                    onChange={(e) =>
                      setConfigForm((f) => ({
                        ...f,
                        click_delivery: normalizeClickDelivery(e.target.value),
                      }))
                    }
                  >
                    <option value="redirect">Redirect (default)</option>
                    <option value="proxy">Reverse proxy</option>
                  </select>
                </label>
                {configForm.click_delivery === 'proxy' ? (
                  <>
                    <label className="form-field" htmlFor="cfg-proxy-upstream-url">
                      Proxy upstream URL
                      <input
                        id="cfg-proxy-upstream-url"
                        className="form-input"
                        type="url"
                        placeholder="https://upstream.example/offer"
                        value={configForm.proxy_upstream_url}
                        data-testid="cfg-proxy-upstream-url"
                        onChange={(e) =>
                          setConfigForm((f) => ({ ...f, proxy_upstream_url: e.target.value }))
                        }
                      />
                    </label>
                    <label className="form-field checkbox-field" htmlFor="cfg-proxy-rewrite-assets">
                      <input
                        id="cfg-proxy-rewrite-assets"
                        type="checkbox"
                        checked={configForm.proxy_rewrite_assets}
                        data-testid="cfg-proxy-rewrite-assets"
                        onChange={(e) =>
                          setConfigForm((f) => ({ ...f, proxy_rewrite_assets: e.target.checked }))
                        }
                      />{' '}
                      Rewrite asset URLs through proxy
                    </label>
                  </>
                ) : null}
              </div>
              <div className="section-card stack" data-testid="campaign-dmr-config">
                <h4 className="subsection-title">Referrer hiding (DMR)</h4>
                <p className="text-muted text-sm">
                  Serves an intermediate HTML page so the browser navigates to the landing without
                  sending Referer. Also activates when <code>dmr=1</code> is on the click URL.
                </p>
                <label className="form-field checkbox-field" htmlFor="cfg-dmr-enabled">
                  <input
                    id="cfg-dmr-enabled"
                    type="checkbox"
                    checked={configForm.dmr_enabled}
                    onChange={(e) =>
                      setConfigForm((f) => ({ ...f, dmr_enabled: e.target.checked }))
                    }
                  />{' '}
                  Enable DMR for campaign clicks
                </label>
              </div>
              <div className="section-card stack" data-testid="campaign-gma-config">
                <h4 className="subsection-title">Gray-market defenses (GMA)</h4>
                <p className="text-muted text-sm">
                  TLS fingerprint blocklist, connection-type policy, L1/L1.5 safe-view gates, and
                  signed offer links. Tracker env <code>LINK_SIGNING_HMAC_SECRET</code> must be set
                  for link signing. Apply preset <strong>Gray market (GMA)</strong> on the Fraud tab
                  to enable safe page, attestation, L1/L1.5, TLS block, and link signing in one step
                  (set <code>safe_page_url</code> separately). IPv6 /64 rotation velocity is
                  separate from the DC CIDR feed — configure <code>IPV6_ROTATION_MODE</code> on the
                  tracker (shadow/live); IPv4 /24 sticky rotation is planned (residential pools).
                </p>
                <label className="form-field checkbox-field" htmlFor="cfg-l1-cidr-block">
                  <input
                    id="cfg-l1-cidr-block"
                    type="checkbox"
                    checked={configForm.l1_cidr_block_enabled}
                    data-testid="cfg-l1-cidr-block"
                    onChange={(e) =>
                      setConfigForm((f) => ({ ...f, l1_cidr_block_enabled: e.target.checked }))
                    }
                  />{' '}
                  L1 DC/hosting CIDR feed (AWS, GCP, Azure, Tor)
                </label>
                <p className="text-muted text-sm">
                  Static datacenter/hosting prefixes from edge feeds — not /24 or /64 rotation
                  detection. Also gates L1 IPv6 rotation when enabled on the tracker.
                </p>
                <label className="form-field checkbox-field" htmlFor="cfg-l15-proxy-vpn-block">
                  <input
                    id="cfg-l15-proxy-vpn-block"
                    type="checkbox"
                    checked={configForm.l15_proxy_vpn_block_enabled}
                    data-testid="cfg-l15-proxy-vpn-block"
                    onChange={(e) =>
                      setConfigForm((f) => ({
                        ...f,
                        l15_proxy_vpn_block_enabled: e.target.checked,
                      }))
                    }
                  />{' '}
                  L1.5 proxy/VPN safe view
                </label>
                <label className="form-field checkbox-field" htmlFor="cfg-tls-fp-block">
                  <input
                    id="cfg-tls-fp-block"
                    type="checkbox"
                    checked={configForm.tls_fingerprint_block_enabled}
                    data-testid="cfg-tls-fp-block"
                    onChange={(e) =>
                      setConfigForm((f) => ({
                        ...f,
                        tls_fingerprint_block_enabled: e.target.checked,
                      }))
                    }
                  />{' '}
                  Block known bad TLS fingerprints (JA3/JA4)
                </label>
                <label className="form-field" htmlFor="cfg-conn-type-policy">
                  Connection type policy
                  <select
                    id="cfg-conn-type-policy"
                    className="form-input"
                    value={configForm.conn_type_policy}
                    data-testid="cfg-conn-type-policy"
                    onChange={(e) =>
                      setConfigForm((f) => ({ ...f, conn_type_policy: e.target.value }))
                    }
                  >
                    <option value="block_vpn_hosting">Block VPN/hosting (default)</option>
                    <option value="mobile_only">Mobile only</option>
                    <option value="residential_only">Residential only</option>
                  </select>
                </label>
                <label className="form-field checkbox-field" htmlFor="cfg-link-signing">
                  <input
                    id="cfg-link-signing"
                    type="checkbox"
                    checked={configForm.link_signing_enabled}
                    data-testid="cfg-link-signing"
                    onChange={(e) =>
                      setConfigForm((f) => ({ ...f, link_signing_enabled: e.target.checked }))
                    }
                  />{' '}
                  Sign outbound offer links (HMAC)
                </label>
                {configForm.link_signing_enabled ? (
                  <label className="form-field" htmlFor="cfg-link-signing-ttl">
                    Link signature TTL (seconds)
                    <input
                      id="cfg-link-signing-ttl"
                      className="form-input"
                      type="number"
                      min={60}
                      max={3600}
                      value={configForm.link_signing_ttl_sec}
                      data-testid="cfg-link-signing-ttl"
                      onChange={(e) =>
                        setConfigForm((f) => ({ ...f, link_signing_ttl_sec: e.target.value }))
                      }
                    />
                  </label>
                ) : null}
              </div>
              <Button
                label={configSaving ? 'Saving…' : 'Save changes'}
                variant="primary"
                size="sm"
                loading={configSaving}
                disabled={configSaving}
                onClick={() => void saveConfig()}
              />
            </div>
          ) : null}
          <ConfigGrid
            rows={[
              ['ID', campaign.id],
              ['Customer', campaign.customer_id ?? '—'],
              ['Timezone', campaign.timezone ?? 'UTC'],
              [
                'Safe page',
                campaign.safe_page_enabled ? campaign.safe_page_url || 'enabled (no URL)' : 'off',
              ],
              ['DMR', campaign.dmr_enabled ? 'on' : 'off'],
              [
                'Click delivery',
                clickDeliveryLabel(normalizeClickDelivery(campaign.click_delivery)),
              ],
              ...(normalizeClickDelivery(campaign.click_delivery) === 'proxy'
                ? [
                    ['Proxy upstream', campaign.proxy_upstream_url ?? '—'] as [string, string],
                    ['Proxy rewrite assets', campaign.proxy_rewrite_assets ? 'yes' : 'no'] as [
                      string,
                      string,
                    ],
                  ]
                : []),
              ['TLS fingerprint block', campaign.tls_fingerprint_block_enabled ? 'on' : 'off'],
              ['L1 DC/hosting CIDR feed', campaign.l1_cidr_block_enabled ? 'on' : 'off'],
              ['L1.5 proxy/VPN block', campaign.l15_proxy_vpn_block_enabled ? 'on' : 'off'],
              ['Conn type policy', campaign.conn_type_policy ?? 'block_vpn_hosting'],
              [
                'Link signing',
                campaign.link_signing_enabled
                  ? `on (${campaign.link_signing_ttl_sec ?? 900}s TTL)`
                  : 'off',
              ],
              [
                'Frequency limit',
                campaign.freq_limit ? `${campaign.freq_limit} / ${campaign.freq_window}s` : 'None',
              ],
              [
                'Geo',
                campaign.target_countries?.length ? campaign.target_countries.join(', ') : 'All',
              ],
              [
                'Schedule',
                campaign.start_at || campaign.end_at
                  ? `${campaign.start_at ? new Date(campaign.start_at).toLocaleString() : '—'} → ${campaign.end_at ? new Date(campaign.end_at).toLocaleString() : '—'}`
                  : 'None',
              ],
              [
                'Daypart',
                campaign.daypart_hours?.length ? campaign.daypart_hours.join(', ') : 'All hours',
              ],
              [
                'Created',
                campaign.created_at ? new Date(campaign.created_at).toLocaleString() : '—',
              ],
            ]}
          />
        </div>
      ) : null}

      {!masked && tab === 'tracking' ? (
        <div className="section-block">
          <CampaignTrackingSection campaignId={id} canWrite={canWriteCampaign} />
        </div>
      ) : null}

      {!masked && tab === 'postbacks' ? (
        <div className="section-block">
          <CampaignPostbackSection campaignId={id} canWrite={canWriteCampaign} />
        </div>
      ) : null}

      {!masked && tab === 'fraud' ? (
        <div className="section-block">
          <CampaignFraudSection
            campaignId={id}
            canWrite={canWriteCampaign}
            onCampaignFlagsChanged={() => reloadCampaign()}
          />
        </div>
      ) : null}

      {!masked && tab === 'filters' ? (
        <div className="section-block">
          <CampaignFiltersSection
            campaignId={id}
            referrerFilter={campaign.referrer_filter ?? ''}
            canWrite={canWriteCampaign}
            onSaved={() => reloadCampaign()}
          />
        </div>
      ) : null}

      {!masked && tab === 'margin' ? (
        <div className="section-block">
          <CampaignMarginGuardSection campaignId={id} canWrite={canWriteCampaign} />
        </div>
      ) : null}

      {!masked && tab === 'events' ? (
        <div className="section-block">
          {eventsTotal > EVENTS_PAGE_SIZE ? (
            <div className="mb-4">
              <FilterToolbar
                pagination={
                  <PaginationBar
                    label={`${eventsPage + 1} / ${eventsPageCount}`}
                    prevDisabled={eventsPage === 0}
                    nextDisabled={(eventsPage + 1) * EVENTS_PAGE_SIZE >= eventsTotal}
                    onPrev={() => setEventsPage((p) => p - 1)}
                    onNext={() => setEventsPage((p) => p + 1)}
                  />
                }
              />
            </div>
          ) : null}
          <div className="table-wrapper table-wrapper--scroll">
            <table className="data-table" aria-label="Campaign events">
              <thead>
                <tr>
                  <th scope="col">Time</th>
                  <th scope="col">Type</th>
                  <th scope="col">Click ID</th>
                  <th scope="col">User</th>
                </tr>
              </thead>
              <tbody>
                {eventsLoading ? <EventsTableSkeleton /> : null}
                {!eventsLoading && eventsRows.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="data-table__empty">
                      <div className="empty-state">
                        <div className="empty-state__title">No events yet</div>
                        <div className="empty-state__desc text-muted text-sm">
                          Campaign events appear after tracked clicks and conversions.
                        </div>
                      </div>
                    </td>
                  </tr>
                ) : null}
                {eventsRows.map((row, index) => (
                  <tr
                    key={`${String(row.created_at ?? '')}-${String(row.click_id ?? '')}-${index}`}
                  >
                    <td>
                      {row.created_at ? new Date(String(row.created_at)).toLocaleString() : '—'}
                    </td>
                    <td>{String(row.event_type ?? '—')}</td>
                    <td className="font-mono text-hint">{String(row.click_id ?? '—')}</td>
                    <td>{String(row.user_id ?? '—')}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      {!masked && tab === 'creative' ? (
        <div className="section-block">
          <div className="stack">
            <ConfigGrid
              rows={[
                ['Target URL', campaign.target_url ?? '—'],
                ['Brand ID', campaign.brand_id ?? '—'],
              ]}
            />
            <CampaignBrandCreativesSection
              brandId={campaign.brand_id ?? ''}
              customerId={campaign.customer_id ?? ''}
              canWrite={canWriteCampaign}
              onBrandCreated={(brandId) => void linkCampaignBrand(brandId)}
            />
          </div>
        </div>
      ) : null}

      {!masked && tab === 'telegram' ? (
        <div className="section-block">
          <CampaignTelegramSection campaignId={id} canWrite={canWriteCampaign} />
        </div>
      ) : null}

      <ForecastModal open={forecastOpen} opts={forecastOpts} onClose={closeForecast} />
    </>
  );
}
