import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import type { CampaignDTO } from '../types/api/campaign.js';
import type { ReportRow } from '../types/api/report.js';
import { el, replaceChildren } from '../lib/dom.js';
import { createResource, type ResourceState } from '../lib/fetch_resource.js';
import { to } from '../lib/to.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderTabBar } from '../ui/tab_bar.js';
import { renderFreshnessBadge } from '../ui/freshness_badge.js';
import * as auth from '../helpers/auth.js';
import { can, maskLevel } from '../helpers/permissions.js';
import {
  pauseCampaign,
  resumeCampaign,
  pollCampaignStatus,
} from '../helpers/campaign_actions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { formatUsdDecimal, ParseDecimal } from '../helpers/money.js';
import { renderBreadcrumbs, type BreadcrumbItem } from '../ui/breadcrumbs.js';
import { shortCustomerId, touchCustomerContext } from '../helpers/customer_context.js';
import { renderPacingPanel } from './pacing_panel.js';
import { estimateDeliveryPct } from '../models/buyer.js';
import { openForecastModal } from '../ui/forecast_modal.js';
import { isoDaysAgo, toIsoNow } from '../helpers/date_presets.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { renderCommercialMetrics, type MetricsBlockDTO } from '../ui/commercial_metrics.js';
import { api } from '../helpers/api_client.js';
import { mountCampaignTelegramPanel } from './campaign_telegram_panel.js';
import { mountCampaignTrackingPanel } from './campaign_tracking_panel.js';
import { mountCampaignPostbackPanel } from './campaign_postback_panel.js';
import { mountCampaignFiltersPanel } from './campaign_filters_panel.js';
import { mountCampaignMarginGuardPanel } from './campaign_margin_guard_panel.js';
import { mountCampaignBrandCreativesPanel } from '../ui/campaign_brand_creatives_panel.js';
import { patchCampaign } from '../helpers/campaign_admin_api.js';
import { tableSkeletonRows, renderEmptyTableCell, renderPaginationBar } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';
import { mountFilterToolbar } from '../ui/filter_toolbar.js';

import { renderIcon } from '../ui/icon.js';
import { displayLabel } from '../helpers/display_labels.js';

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

/**
 * Render a two-column label/value config grid.
 *
 * @param {Array<[string, string]>} rows
 * @returns {HTMLElement}
 */
function configGrid(rows: Array<[string, string]>) {
  return el('dl', { className: 'definition-list' },
    rows.flatMap(([label, value]: [string, string]) => [
      el('dt', null, label),
      el('dd', { className: 'font-mono text-secondary' }, value),
    ]),
  );
}

/**
 * Mount the campaign detail view with pause/resume and stats tabs.
 *
 * @param {HTMLElement} container
 * @param {{ params: Record<string, string> }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const id = ctx.params.id;
  let actionLoading = false;
  let actionError: any = null;
  let chartHandle: any = null;
  /** @type {HTMLElement|null} */
  let chartMount: any = null;

  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const masked = maskLevel(permissions) === 'masked';
  const canPause =
    can(permissions, 'campaigns:write') || can(permissions, 'campaigns:pause');
  const canWriteCampaign = can(permissions, 'campaigns:write');

  function allowedTabIds(): string[] {
    const list = ['overview', 'stats', 'config'];
    if (!masked) {
      list.push('tracking', 'postbacks', 'filters', 'margin', 'events', 'creative', 'telegram');
    }
    return list;
  }

  function resolveTabFromQuery(): string {
    const requested = ctx.query.get('tab')?.trim() ?? '';
    return allowedTabIds().includes(requested) ? requested : 'overview';
  }

  let tab = resolveTabFromQuery();

  const tgSlot = el('div', { 'data-tg-panel': '' });
  const trackingSlot = el('div', { 'data-tracking-panel': '' });
  const postbackSlot = el('div', { 'data-postback-panel': '' });
  const filtersSlot = el('div', { 'data-filters-panel': '' });
  const marginSlot = el('div', { 'data-margin-panel': '' });
  const creativeSlot = el('div', { 'data-creative-panel': '' });
  /** @type {{ destroy: () => void }|null} */
  let tgPanelHandle: any = null;
  /** @type {{ destroy: () => void }|null} */
  let trackingPanelHandle: any = null;
  /** @type {{ destroy: () => void }|null} */
  let postbackPanelHandle: any = null;
  /** @type {{ destroy: () => void }|null} */
  let filtersPanelHandle: any = null;
  /** @type {{ destroy: () => void, reload: () => void }|null} */
  let marginPanelHandle: any = null;
  let creativePanelHandle: { destroy: () => void; reload: () => void } | null = null;
  let eventsPage = 0;
  let eventsRows: ReportRow[] = [];
  let eventsTotal = 0;
  let eventsLoading = false;
  let configSaving = false;
  let configError: any = null;

  const campaignState: ResourceState<CampaignDTO> = { data: null, loading: true, error: null };
  const statsState: ResourceState<CampaignStatsDTO> = { data: null, loading: false, error: null };
  const dashboardState: ResourceState<CampaignDashboardDTO> = { data: null, loading: true, error: null };
  const actionGate = createInFlightGuard();

  function statsUrl() {
    const params = new URLSearchParams({ granularity: 'hour' });
    if (masked) {
      params.set('from', isoDaysAgo(7));
      params.set('to', toIsoNow());
    }
    return `/api/v1/campaigns/${id}/stats?${params.toString()}`;
  }

  function overviewImpressions7d() {
    return Number(statsState.data?.metrics?.impressions ?? 0);
  }

  function tabs() {
    const list = [
      { id: 'overview', label: 'Overview' },
      { id: 'stats', label: 'Statistics' },
      { id: 'config', label: 'Configuration' },
    ];
    if (!masked) {
      list.push({ id: 'tracking', label: 'Integration' });
      list.push({ id: 'postbacks', label: 'CAPI & Postbacks' });
      list.push({ id: 'filters', label: 'Filters' });
      list.push({ id: 'margin', label: 'Margin guard' });
      list.push({ id: 'events', label: 'Event log' });
      list.push({ id: 'creative', label: 'Creative' });
      list.push({ id: 'telegram', label: 'Telegram' });
    }
    return list;
  }

  function destroyAuxPanels() {
    tgPanelHandle?.destroy();
    tgPanelHandle = null;
    trackingPanelHandle?.destroy();
    trackingPanelHandle = null;
    postbackPanelHandle?.destroy();
    postbackPanelHandle = null;
    filtersPanelHandle?.destroy();
    filtersPanelHandle = null;
    marginPanelHandle?.destroy();
    marginPanelHandle = null;
    creativePanelHandle?.destroy();
    creativePanelHandle = null;
  }

  async function loadEvents() {
    eventsLoading = true;
    render();
    const limit = 50;
    const offset = eventsPage * limit;
    const [res, err] = await to(api(`/api/v1/campaigns/${id}/events?limit=${limit}&offset=${offset}`));
    if (destroyed) return;
    eventsLoading = false;
    if (err) {
      eventsRows = [];
      eventsTotal = 0;
    } else {
      const data = (res?.data ?? {}) as CampaignEventsResponse;
      eventsRows = data.items ?? [];
      eventsTotal = data.total ?? 0;
    }
    render();
  }

  async function saveConfig(campaign: CampaignDTO) {
    if (!canWriteCampaign || configSaving) return;

    const nameInput = container.querySelector('#cfg-name');
    const pacingInput = container.querySelector('#cfg-pacing');
    const tzInput = container.querySelector('#cfg-timezone');
    const safePageEnabledInput = container.querySelector('#cfg-safe-page-enabled');
    const safePageUrlInput = container.querySelector('#cfg-safe-page-url');
    const dailyBudgetInput = container.querySelector('#cfg-daily-budget');
    const targetUrlInput = container.querySelector('#cfg-target-url');
    const geoInput = container.querySelector('#cfg-geo');
    const freqLimitInput = container.querySelector('#cfg-freq-limit');
    const freqWindowInput = container.querySelector('#cfg-freq-window');

    const body: Record<string, unknown> = {
      name: nameInput instanceof HTMLInputElement ? nameInput.value.trim() : campaign.name,
      pacing_mode: pacingInput instanceof HTMLSelectElement ? pacingInput.value : campaign.pacing_mode,
      timezone: tzInput instanceof HTMLInputElement ? tzInput.value.trim() : campaign.timezone,
    };
    if (!String(body.name ?? '').trim()) {
      configError = 'Name is required';
      render();
      return;
    }
    if (dailyBudgetInput instanceof HTMLInputElement && dailyBudgetInput.value.trim()) {
      try {
        body.daily_budget_micro = ParseDecimal(dailyBudgetInput.value.trim());
      } catch {
        configError = 'Invalid daily budget';
        render();
        return;
      }
    }
    if (targetUrlInput instanceof HTMLInputElement) {
      const url = targetUrlInput.value.trim();
      if (url && !/^https?:\/\//i.test(url)) {
        configError = 'Target URL must start with http:// or https://';
        render();
        return;
      }
      body.target_url = url;
    }
    if (geoInput instanceof HTMLInputElement) {
      const raw = geoInput.value.trim();
      body.target_countries = raw
        ? raw.split(',').map((c) => c.trim().toUpperCase()).filter(Boolean)
        : [];
    }
    if (freqLimitInput instanceof HTMLInputElement) {
      const n = Number.parseInt(freqLimitInput.value, 10);
      if (Number.isFinite(n) && n >= 0) body.freq_limit = n;
    }
    if (freqWindowInput instanceof HTMLInputElement) {
      const n = Number.parseInt(freqWindowInput.value, 10);
      if (Number.isFinite(n) && n > 0) body.freq_window = n;
    }
    if (safePageEnabledInput instanceof HTMLInputElement) {
      body.safe_page_enabled = safePageEnabledInput.checked;
    }
    if (safePageUrlInput instanceof HTMLInputElement) {
      body.safe_page_url = safePageUrlInput.value.trim();
    }

    configSaving = true;
    configError = null;
    render();

    const [, err] = await to(patchCampaign(id, body));
    configSaving = false;
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        render();
        return;
      }
      configError = err.message || 'Save failed';
      render();
      return;
    }
    pushToastMessage({
      title: 'Campaign saved',
      message: 'Config propagates to trackers within ~60s.',
    });
    campaignResource.reload();
  }

  function destroyChart() {
    chartHandle?.destroy();
    chartHandle = null;
  }

  function mountChart(hourly: any) {
    destroyChart();
    if (!chartMount) return;
    import('../charts/campaign_stats_chart.js').then((mod) => {
      if (destroyed || !chartMount) return;
      chartHandle = mod.mountChart(chartMount, hourly);
    });
  }

  async function handlePause() {
    if (!actionGate.tryAcquire()) return;
    actionLoading = true;
    actionError = null;
    render();
    const [, pauseErr] = await to(pauseCampaign(id));
    if (destroyed) {
      actionGate.release();
      return;
    }
    if (pauseErr) {
      if (pauseErr instanceof ConfirmCancelledError) {
        actionLoading = false;
        actionGate.release();
        render();
        return;
      }
      actionError = pauseErr.message || 'Failed to pause campaign';
      actionLoading = false;
      actionGate.release();
      render();
      return;
    }
    const [, pollErr] = await to(pollCampaignStatus(id, 'PAUSED'));
    if (destroyed) {
      actionGate.release();
      return;
    }
    if (pollErr) {
      actionError = pollErr.message || 'Failed to pause campaign';
    } else {
      campaignResource.reload();
    }
    actionLoading = false;
    actionGate.release();
    render();
  }

  async function handleResume() {
    if (!actionGate.tryAcquire()) return;
    actionLoading = true;
    actionError = null;
    render();
    const [, resumeErr] = await to(resumeCampaign(id));
    if (destroyed) {
      actionGate.release();
      return;
    }
    if (resumeErr) {
      if (resumeErr instanceof ConfirmCancelledError) {
        actionLoading = false;
        actionGate.release();
        render();
        return;
      }
      actionError = resumeErr.message || 'Failed to resume campaign';
      actionLoading = false;
      actionGate.release();
      render();
      return;
    }
    const [, pollErr] = await to(pollCampaignStatus(id, 'ACTIVE'));
    if (destroyed) {
      actionGate.release();
      return;
    }
    if (pollErr) {
      actionError = pollErr.message || 'Failed to resume campaign';
    } else {
      campaignResource.reload();
    }
    actionLoading = false;
    actionGate.release();
    render();
  }

  function renderLoadingCards() {
    replaceChildren(container,
      el('div', { className: 'grid-stats section-block' },
        ['Status', 'Budget', 'Spend', 'Pacing'].map((label: any) =>
          el('div', { className: 'metric-card metric-card--loading' },
            el('div', { className: 'metric-card__label' }, label),
            el('div', { className: 'metric-card__value' }, '…'),
          ),
        ),
      ),
    );
  }

  function render() {
    if (destroyed) return;

    if (campaignState.loading && !campaignState.data) {
      renderLoadingCards();
      return;
    }

    if (campaignState.error) {
      replaceChildren(container, renderErrorBlock(campaignState.error));
      return;
    }

    const campaign = campaignState.data;
    if (!campaign) return;

    const status = (campaign.status || '').toUpperCase();
    const isPaused = status === 'PAUSED';
    const isActive = status === 'ACTIVE';

    const eventsPagination = eventsTotal > 50
      ? renderPaginationBar({
        label: `${eventsPage + 1} / ${Math.ceil(eventsTotal / 50)}`,
        prevDisabled: eventsPage === 0,
        nextDisabled: (eventsPage + 1) * 50 >= eventsTotal,
        onPrev: () => { eventsPage -= 1; loadEvents(); },
        onNext: () => { eventsPage += 1; loadEvents(); },
      })
      : null;
    const eventsToolbar = eventsPagination
      ? (() => {
        const wrap = el('div', { className: 'mb-4' });
        mountFilterToolbar(wrap, { pagination: eventsPagination });
        return wrap;
      })()
      : null;

    chartMount = el('div');

    const crumbs: BreadcrumbItem[] = [{ label: 'Campaigns', href: '/campaigns' }];
    if (campaign.customer_id) {
      crumbs.push({
        label: shortCustomerId(campaign.customer_id, 12),
        href: `/customers/${campaign.customer_id}`,
      });
      touchCustomerContext(campaign.customer_id);
    }
    crumbs.push({ label: String(campaign.name ?? '') });

    const children = [
      el('div', { className: 'page-header' },
        renderBreadcrumbs(crumbs),
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            renderIcon('megaphone', { size: 20, className: 'text-muted' }),
            el('h1', { className: 'page-header__title' }, campaign.name),
          ),
          renderStatusBadge(campaign.status),
          canPause
            ? el('div', { className: 'cluster--actions ml-auto' },
              isActive
                ? renderButton({
                  label: 'Pause',
                  variant: 'danger',
                  size: 'sm',
                  icon: 'pause',
                  loading: actionLoading,
                  disabled: actionLoading,
                  onClick: handlePause,
                })
                : null,
              isPaused
                ? renderButton({
                  label: 'Resume',
                  variant: 'primary',
                  size: 'sm',
                  icon: 'play',
                  loading: actionLoading,
                  disabled: actionLoading,
                  onClick: handleResume,
                })
                : null,
            )
            : null,
        ),
        actionError
          ? el('p', { className: 'text-danger text-sm mt-2' }, actionError)
          : null,
      ),
      renderTabBar({ tabs: tabs(), active: tab, onChange: (t: any) => {
        destroyAuxPanels();
        tab = t;
        const qs = new URLSearchParams(window.location.search);
        if (t === 'overview') qs.delete('tab');
        else qs.set('tab', t);
        const suffix = qs.toString();
        window.history.replaceState(null, '', suffix ? `${window.location.pathname}?${suffix}` : window.location.pathname);
        if (t === 'stats') statsResource.reload();
        else if (t === 'events') loadEvents();
        else destroyChart();
        render();
      } }),
      tab === 'overview'
        ? el('div', { className: 'section-block stack' },
          dashboardState.loading
            ? el('span', { className: 'text-muted' }, 'Loading economics…')
            : renderCommercialMetrics(dashboardState.data?.kpis, { masked }),
          !masked && campaign
            ? renderButton({
              label: 'Forecast delivery',
              variant: 'secondary',
              size: 'sm',
              className: 'shrink-0',
              onClick: () => openForecastModal({
                campaignId: id,
                customerId: campaign.customer_id,
                budgetMicro: Math.round(Number(campaign.budget_limit ?? 0) * 1_000_000),
                startAt: isoDaysAgo(0),
                endAt: toIsoNow(),
              }),
            })
            : null,
          masked
            ? renderPacingPanel({
              status: campaign.status,
              pacingMode: campaign.pacing_mode,
              impressions7d: overviewImpressions7d(),
              deliveryPct: estimateDeliveryPct(
                overviewImpressions7d(),
                campaign.status,
              ),
            })
            : el('div', { className: 'grid-stats section-block' },
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Budget limit'),
                el('div', { className: 'metric-card__value font-mono' },
                  formatUsdDecimal(campaign.budget_limit ?? '0.00'),
                ),
              ),
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Current spend'),
                el('div', { className: 'metric-card__value font-mono' },
                  formatUsdDecimal(campaign.current_spend ?? '0.00'),
                ),
              ),
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Daily budget'),
                el('div', { className: 'metric-card__value font-mono' },
                  formatUsdDecimal(campaign.daily_budget ?? '0.00'),
                ),
              ),
              el('div', { className: 'metric-card' },
                el('div', { className: 'metric-card__label' }, 'Pacing'),
                el('div', { className: 'metric-card__value' }, displayLabel(campaign.pacing_mode)),
              ),
            ),
        )
        : null,
      tab === 'stats'
        ? el('div', { className: 'section-block stack' },
          statsState.loading ? el('span', { className: 'text-muted' }, 'Loading statistics…') : null,
          statsState.error
            ? el('p', { className: 'text-danger text-sm' },
              statsState.error instanceof Error
                ? statsState.error.message
                : String(statsState.error),
            )
            : null,
          statsState.data
            ? el('div', { className: 'stack' },
              el('div', { className: 'flex items-center gap-2' },
                el('h2', { className: 'subsection-title' }, 'Hourly metrics'),
                renderFreshnessBadge({ stale: statsState.data.stale }),
              ),
              el('div', { className: 'metric-row' },
                el('div', { className: 'metric-card' },
                  el('div', { className: 'metric-card__label' }, 'Impressions'),
                  el('div', { className: 'metric-card__value' },
                    String(statsState.data.metrics?.impressions ?? 0),
                  ),
                ),
                el('div', { className: 'metric-card' },
                  el('div', { className: 'metric-card__label' }, 'Clicks'),
                  el('div', { className: 'metric-card__value' },
                    String(statsState.data.metrics?.clicks ?? 0),
                  ),
                ),
                el('div', { className: 'metric-card' },
                  el('div', { className: 'metric-card__label' }, 'Conversions'),
                  el('div', { className: 'metric-card__value' },
                    String(statsState.data.metrics?.conversions ?? 0),
                  ),
                ),
                el('div', { className: 'metric-card' },
                  el('div', { className: 'metric-card__label' }, 'Spend (API)'),
                  el('div', { className: 'metric-card__value font-mono' },
                    masked ? '—' : formatUsdDecimal(statsState.data.current_spend ?? '0.00'),
                  ),
                ),
              ),
              el('div', { className: 'section-card' },
                el('h3', { className: 'subsection-title' }, 'Hourly trend'),
                chartMount,
              ),
            )
            : null,
        )
        : null,
      tab === 'config'
        ? el('div', { className: 'section-block stack' },
          canWriteCampaign && !masked
            ? el('div', { className: 'section-card stack' },
              el('h3', { className: 'subsection-title' }, 'Edit settings'),
              configError ? el('p', { className: 'text-danger text-sm' }, configError) : null,
              el('label', { className: 'form-field', htmlFor: 'cfg-name' },
                'Name',
                el('input', {
                  id: 'cfg-name',
                  className: 'form-input',
                  defaultValue: campaign.name,
                }),
              ),
              el('label', { className: 'form-field', htmlFor: 'cfg-pacing' },
                'Pacing',
                el('select', {
                  id: 'cfg-pacing',
                  className: 'form-input form-input--sm',
                  defaultValue: campaign.pacing_mode ?? 'ASAP',
                },
                  el('option', { value: 'ASAP' }, 'ASAP'),
                  el('option', { value: 'EVEN' }, 'Even'),
                  el('option', { value: 'VPP' }, 'VPP'),
                ),
              ),
              el('label', { className: 'form-field', htmlFor: 'cfg-timezone' },
                'Timezone',
                el('input', {
                  id: 'cfg-timezone',
                  className: 'form-input form-input--sm',
                  defaultValue: campaign.timezone ?? 'UTC',
                }),
              ),
              el('label', { className: 'form-field', htmlFor: 'cfg-daily-budget' },
                'Daily budget (USD)',
                el('input', {
                  id: 'cfg-daily-budget',
                  className: 'form-input form-input--sm',
                  inputMode: 'decimal',
                  placeholder: '0.00',
                  defaultValue: campaign.daily_budget ?? '',
                  'data-testid': 'cfg-daily-budget',
                }),
              ),
              el('label', { className: 'form-field', htmlFor: 'cfg-target-url' },
                'Target URL',
                el('input', {
                  id: 'cfg-target-url',
                  className: 'form-input',
                  type: 'url',
                  placeholder: 'https://landing.example/',
                  defaultValue: campaign.target_url ?? '',
                  'data-testid': 'cfg-target-url',
                }),
              ),
              el('label', { className: 'form-field', htmlFor: 'cfg-geo' },
                'Target countries (ISO, comma-separated)',
                el('input', {
                  id: 'cfg-geo',
                  className: 'form-input form-input--sm',
                  placeholder: 'US,GB,DE or empty for all',
                  defaultValue: (campaign.target_countries ?? []).join(','),
                  'data-testid': 'cfg-geo',
                }),
              ),
              el('div', { className: 'filter-row' },
                el('label', { className: 'form-field', htmlFor: 'cfg-freq-limit' },
                  'Freq limit',
                  el('input', {
                    id: 'cfg-freq-limit',
                    className: 'form-input form-input--sm',
                    type: 'number',
                    min: '0',
                    defaultValue: String(campaign.freq_limit ?? 0),
                    'data-testid': 'cfg-freq-limit',
                  }),
                ),
                el('label', { className: 'form-field', htmlFor: 'cfg-freq-window' },
                  'Freq window (sec)',
                  el('input', {
                    id: 'cfg-freq-window',
                    className: 'form-input form-input--sm',
                    type: 'number',
                    min: '1',
                    defaultValue: String(campaign.freq_window ?? 86400),
                    'data-testid': 'cfg-freq-window',
                  }),
                ),
              ),
              el('div', {
                className: 'section-card stack',
                'data-testid': 'campaign-safe-page-config',
              },
                el('h4', { className: 'subsection-title' }, 'Safe page (cloak companion)'),
                el('p', { className: 'text-muted text-sm' },
                  'When enabled, suspicious clicks (IVT / placement blacklist) redirect to the safe URL instead of the money landing. Clean traffic uses brand creatives as usual.',
                ),
                el('label', { className: 'form-field checkbox-field', htmlFor: 'cfg-safe-page-enabled' },
                  el('input', {
                    id: 'cfg-safe-page-enabled',
                    type: 'checkbox',
                    checked: campaign.safe_page_enabled === true,
                  }),
                  ' Enable safe-page redirect',
                ),
                el('label', { className: 'form-field', htmlFor: 'cfg-safe-page-url' },
                  'Safe page URL',
                  el('input', {
                    id: 'cfg-safe-page-url',
                    className: 'form-input',
                    type: 'url',
                    placeholder: 'https://safe.example/white-page',
                    defaultValue: campaign.safe_page_url ?? '',
                  }),
                ),
              ),
              renderButton({
                label: configSaving ? 'Saving…' : 'Save changes',
                variant: 'primary',
                size: 'sm',
                loading: configSaving,
                disabled: configSaving,
                onClick: () => saveConfig(campaign),
              }),
            )
            : null,
          configGrid([
            ['ID', campaign.id],
            ['Customer', campaign.customer_id],
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
          ]),
        )
        : null,
      tab === 'tracking' && !masked ? el('div', { className: 'section-block' }, trackingSlot) : null,
      tab === 'postbacks' && !masked ? el('div', { className: 'section-block' }, postbackSlot) : null,
      tab === 'filters' && !masked ? el('div', { className: 'section-block' }, filtersSlot) : null,
      tab === 'margin' && !masked ? el('div', { className: 'section-block' }, marginSlot) : null,
      tab === 'events' && !masked
        ? el('div', { className: 'section-block' },
          eventsToolbar,
          el('div', { className: 'table-wrapper table-wrapper--scroll' },
            el('table', { className: 'data-table', 'aria-label': 'Campaign events' },
              el('thead', null,
                el('tr', null,
                  el('th', { scope: 'col' }, 'Time'),
                  el('th', { scope: 'col' }, 'Type'),
                  el('th', { scope: 'col' }, 'Click ID'),
                  el('th', { scope: 'col' }, 'User'),
                ),
              ),
              el('tbody', null,
                eventsLoading ? tableSkeletonRows(4) : null,
                !eventsLoading && eventsRows.length === 0
                  ? el('tr', null,
                    renderEmptyTableCell(4, {
                      title: 'No events yet',
                      description: 'Campaign events appear after tracked clicks and conversions.',
                      icon: 'activity',
                    }),
                  )
                  : null,
                eventsRows.map((row: any) => el('tr', null,
                  el('td', null, row.created_at ? new Date(row.created_at).toLocaleString() : '—'),
                  el('td', null, row.event_type ?? '—'),
                  el('td', { className: 'font-mono text-hint' }, row.click_id ?? '—'),
                  el('td', null, row.user_id ?? '—'),
                )),
              ),
            ),
          ),
        )
        : null,
      tab === 'creative' && !masked
        ? el('div', { className: 'section-block' },
          el('div', { className: 'stack' },
            configGrid([
              ['Target URL', campaign.target_url ?? '—'],
              ['Brand ID', campaign.brand_id ?? '—'],
            ]),
            creativeSlot,
          ),
        )
        : null,
      tab === 'telegram' && !masked
        ? el('div', { className: 'section-block' }, tgSlot)
        : null,
    ];

    replaceChildren(container, ...children);

    if (tab === 'stats' && statsState.data) {
      mountChart(statsState.data.hourly ?? []);
    }
    if (tab === 'telegram' && !masked && !tgPanelHandle) {
      tgPanelHandle = mountCampaignTelegramPanel(tgSlot, {
        campaignId: id,
        canWrite: canWriteCampaign,
      });
    }
    if (tab === 'tracking' && !masked && !trackingPanelHandle) {
      trackingPanelHandle = mountCampaignTrackingPanel(trackingSlot, {
        campaignId: id,
        navigate: ctx.navigate,
      });
    }
    if (tab === 'postbacks' && !masked && !postbackPanelHandle) {
      postbackPanelHandle = mountCampaignPostbackPanel(postbackSlot, {
        campaignId: id,
        canWrite: canWriteCampaign,
        navigate: ctx.navigate,
      });
    }
    if (tab === 'filters' && !masked && !filtersPanelHandle) {
      filtersPanelHandle = mountCampaignFiltersPanel(filtersSlot, {
        campaignId: id,
        referrerFilter: campaign.referrer_filter ?? '',
        canWrite: canWriteCampaign,
        onSaved: () => campaignResource.reload(),
      });
    }
    if (tab === 'margin' && !masked && !marginPanelHandle) {
      marginPanelHandle = mountCampaignMarginGuardPanel(marginSlot, {
        campaignId: id,
        canWrite: canWriteCampaign,
      });
    }
    if (tab === 'creative' && !masked && !creativePanelHandle) {
      creativePanelHandle = mountCampaignBrandCreativesPanel(creativeSlot, {
        brandId: campaign.brand_id ?? '',
        customerId: campaign.customer_id ?? '',
        canWrite: canWriteCampaign,
        onBrandCreated: () => {
          pushToastMessage({
            title: 'Brand created',
            message: 'Creatives work for this session. Linking brand_id on the campaign requires API work (MILESTONE §1.2.4).',
          });
        },
      });
    }
  }

  const campaignResource = createResource<CampaignDTO>(
    () => `/api/v1/campaigns/${id}`,
    {
      onUpdate: (s) => {
        Object.assign(campaignState, s);
        render();
      },
    },
  );

  const statsResource = createResource<CampaignStatsDTO>(
    () => statsUrl(),
    {
      skip: () => !masked && tab !== 'stats',
      onUpdate: (s) => {
        Object.assign(statsState, s);
        render();
      },
    },
  );

  async function loadDashboard() {
    dashboardState.loading = true;
    render();
    const [res, err] = await to(api(`/api/v1/dashboards/campaign/${id}`));
    if (destroyed) return;
    dashboardState.loading = false;
    if (err) {
      dashboardState.error = err;
    } else {
      dashboardState.data = (res?.data as CampaignDashboardDTO | null) ?? null;
    }
    render();
  }

  loadDashboard();
  render();

  return {
    destroy() {
      destroyed = true;
      actionGate.release();
      destroyChart();
      destroyAuxPanels();
      campaignResource.destroy();
      statsResource.destroy();
    },
  };
}
