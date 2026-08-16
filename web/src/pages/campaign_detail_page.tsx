import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import type { CampaignDTO } from '../types/api/campaign.js';
import type { ReportRow } from '../types/api/report.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { can, maskLevel } from '../helpers/permissions.js';
import {
  pauseCampaign,
  resumeCampaign,
  pollCampaignStatus,
} from '../helpers/campaign_actions.js';
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
import { CampaignMarginGuardSection } from '../components/campaign_margin_guard_section.js';
import { CampaignTelegramSection } from '../components/campaign_telegram_section.js';
import { CampaignBrandCreativesSection } from '../components/campaign_brand_creatives_section.js';
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
import { useResource } from '../hooks/use_resource.js';

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

function allowedTabIds(masked: boolean): string[] {
  const list = ['overview', 'stats', 'config'];
  if (!masked) {
    list.push('tracking', 'postbacks', 'filters', 'margin', 'events', 'creative', 'telegram');
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
      { id: 'filters', label: 'Filters' },
      { id: 'margin', label: 'Margin guard' },
      { id: 'events', label: 'Event log' },
      { id: 'creative', label: 'Creative' },
      { id: 'telegram', label: 'Telegram' },
    );
  }
  return list;
}

function ConfigGrid({ rows }: { rows: Array<[string, string]> }) {
  return (
    <dl className="definition-list">
      {rows.flatMap(([label, value]) => [
        <dt key={`${label}-dt`}>{label}</dt>,
        <dd key={`${label}-dd`} className="font-mono text-secondary">{value}</dd>,
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
            <td key={`ev-skel-${i}-${j}`}><span className="skeleton-bar" /></td>
          ))}
        </tr>
      ))}
    </>
  );
}

/**
 * Campaign detail with tabs, pause/resume, and lazy-loaded panel sections.
 */
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
    daily_budget: '',
    target_url: '',
    geo: '',
    freq_limit: '0',
    freq_window: '86400',
    safe_page_enabled: false,
    safe_page_url: '',
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
      daily_budget: campaign.daily_budget ?? '',
      target_url: campaign.target_url ?? '',
      geo: (campaign.target_countries ?? []).join(','),
      freq_limit: String(campaign.freq_limit ?? 0),
      freq_window: String(campaign.freq_window ?? 86400),
      safe_page_enabled: campaign.safe_page_enabled === true,
      safe_page_url: campaign.safe_page_url ?? '',
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
    return () => { cancelled = true; };
  }, [id]);

  const setTab = useCallback((nextTab: string) => {
    const next = new URLSearchParams(searchParams);
    if (nextTab === 'overview') next.delete('tab');
    else next.set('tab', nextTab);
    setSearchParams(next, { replace: true });
    if (nextTab === 'stats') reloadStats();
    if (nextTab === 'events') setEventsPage(0);
  }, [searchParams, setSearchParams, reloadStats]);

  const loadEvents = useCallback(async (page: number) => {
    if (!id) return;
    setEventsLoading(true);
    const limit = EVENTS_PAGE_SIZE;
    const offset = page * limit;
    const [res, err] = await to(api(`/api/v1/campaigns/${id}/events?limit=${limit}&offset=${offset}`));
    setEventsLoading(false);
    if (err) {
      setEventsRows([]);
      setEventsTotal(0);
      return;
    }
    const data = (res?.data ?? {}) as CampaignEventsResponse;
    setEventsRows(data.items ?? []);
    setEventsTotal(data.total ?? 0);
  }, [id]);

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
    const url = configForm.target_url.trim();
    if (url && !/^https?:\/\//i.test(url)) {
      setConfigError('Target URL must start with http:// or https://');
      return;
    }
    body.target_url = url;
    body.target_countries = configForm.geo.trim()
      ? configForm.geo.split(',').map((c) => c.trim().toUpperCase()).filter(Boolean)
      : [];
    const freqLimit = Number.parseInt(configForm.freq_limit, 10);
    if (Number.isFinite(freqLimit) && freqLimit >= 0) body.freq_limit = freqLimit;
    const freqWindow = Number.parseInt(configForm.freq_window, 10);
    if (Number.isFinite(freqWindow) && freqWindow > 0) body.freq_window = freqWindow;
    body.safe_page_enabled = configForm.safe_page_enabled;
    body.safe_page_url = configForm.safe_page_url.trim();
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
              onClick={() => openForecast({
                campaignId: id,
                customerId: campaign.customer_id,
                budgetMicro: Math.round(Number(campaign.budget_limit ?? 0) * 1_000_000),
                startAt: isoDaysAgo(0),
                endAt: toIsoNow(),
              })}
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
                  <div className="metric-card__value">{String(stats.metrics?.impressions ?? 0)}</div>
                </div>
                <div className="metric-card">
                  <div className="metric-card__label">Clicks</div>
                  <div className="metric-card__value">{String(stats.metrics?.clicks ?? 0)}</div>
                </div>
                <div className="metric-card">
                  <div className="metric-card__label">Conversions</div>
                  <div className="metric-card__value">{String(stats.metrics?.conversions ?? 0)}</div>
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
          {canWriteCampaign && !masked ? (
            <div className="section-card stack">
              <h3 className="subsection-title">Edit settings</h3>
              {configError ? <p className="text-danger text-sm">{configError}</p> : null}
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
                    <option key={flow.id} value={flow.id}>{flow.name}</option>
                  ))}
                </select>
              </label>
              <p className="text-muted text-sm">
                <a href="/campaigns/flows">Manage landers, offers &amp; flows →</a>
              </p>
              <div className="section-card stack" data-testid="campaign-safe-page-config">
                <h4 className="subsection-title">Safe page (cloak companion)</h4>
                <p className="text-muted text-sm">
                  When enabled, suspicious clicks (IVT / placement blacklist) redirect to the safe URL instead of the money landing. Clean traffic uses brand creatives as usual.
                </p>
                <label className="form-field checkbox-field" htmlFor="cfg-safe-page-enabled">
                  <input
                    id="cfg-safe-page-enabled"
                    type="checkbox"
                    checked={configForm.safe_page_enabled}
                    onChange={(e) => setConfigForm((f) => ({ ...f, safe_page_enabled: e.target.checked }))}
                  />
                  {' '}
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
                    onChange={(e) => setConfigForm((f) => ({ ...f, safe_page_url: e.target.value }))}
                  />
                </label>
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
          <ConfigGrid rows={[
            ['ID', campaign.id],
            ['Customer', campaign.customer_id ?? '—'],
            ['Timezone', campaign.timezone ?? 'UTC'],
            ['Safe page', campaign.safe_page_enabled ? (campaign.safe_page_url || 'enabled (no URL)') : 'off'],
            [
              'Frequency limit',
              campaign.freq_limit
                ? `${campaign.freq_limit} / ${campaign.freq_window}s`
                : 'None',
            ],
            [
              'Geo',
              campaign.target_countries?.length
                ? campaign.target_countries.join(', ')
                : 'All',
            ],
            [
              'Created',
              campaign.created_at
                ? new Date(campaign.created_at).toLocaleString()
                : '—',
            ],
          ]}
          />
        </div>
      ) : null}

      {!masked && tab === 'tracking' ? (
        <div className="section-block">
          <CampaignTrackingSection campaignId={id} />
        </div>
      ) : null}

      {!masked && tab === 'postbacks' ? (
        <div className="section-block">
          <CampaignPostbackSection campaignId={id} canWrite={canWriteCampaign} />
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
                pagination={(
                  <PaginationBar
                    label={`${eventsPage + 1} / ${eventsPageCount}`}
                    prevDisabled={eventsPage === 0}
                    nextDisabled={(eventsPage + 1) * EVENTS_PAGE_SIZE >= eventsTotal}
                    onPrev={() => setEventsPage((p) => p - 1)}
                    onNext={() => setEventsPage((p) => p + 1)}
                  />
                )}
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
                  <tr key={`${String(row.created_at ?? '')}-${String(row.click_id ?? '')}-${index}`}>
                    <td>
                      {row.created_at
                        ? new Date(String(row.created_at)).toLocaleString()
                        : '—'}
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
            <ConfigGrid rows={[
              ['Target URL', campaign.target_url ?? '—'],
              ['Brand ID', campaign.brand_id ?? '—'],
            ]}
            />
            <CampaignBrandCreativesSection
              brandId={campaign.brand_id ?? ''}
              customerId={campaign.customer_id ?? ''}
              canWrite={canWriteCampaign}
              onBrandCreated={() => {
                pushToastMessage({
                  title: 'Brand created',
                  message: 'Creatives work for this session. Linking brand_id on the campaign requires API work (MILESTONE §1.2.4).',
                });
              }}
            />
          </div>
        </div>
      ) : null}

      {!masked && tab === 'telegram' ? (
        <div className="section-block">
          <CampaignTelegramSection campaignId={id} canWrite={canWriteCampaign} />
        </div>
      ) : null}

      <ForecastModal
        open={forecastOpen}
        opts={forecastOpts}
        onClose={closeForecast}
      />
    </>
  );
}
