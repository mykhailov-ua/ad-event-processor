import { el, replaceChildren } from '../lib/dom.js';
import { createResource } from '../lib/fetch_resource.js';
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
import { renderStatusBadge } from '../ui/status_badge.js';
import { formatUsdDecimal } from '../helpers/money.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { shortCustomerId, touchCustomerContext } from '../helpers/customer_context.js';
import { renderPacingPanel } from './pacing_panel.js';
import { estimateDeliveryPct } from '../models/buyer.js';
import { openForecastModal } from '../ui/forecast_modal.js';
import { isoDaysAgo, toIsoNow } from '../helpers/date_presets.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { renderCommercialMetrics } from '../ui/commercial_metrics.js';
import { api } from '../helpers/api_client.js';
import { mountCampaignTelegramPanel } from './campaign_telegram_panel.js';

import { renderIcon } from '../ui/icon.js';
import { displayLabel } from '../helpers/display_labels.js';

/**
 * Render a two-column label/value config grid.
 *
 * @param {Array<[string, string]>} rows
 * @returns {HTMLElement}
 */
function configGrid(rows) {
  return el('dl', { className: 'definition-list' },
    rows.flatMap(([label, value]) => [
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
export function mount(container, ctx) {
  let destroyed = false;
  const id = ctx.params.id;
  let tab = 'overview';
  let actionLoading = false;
  let actionError = null;
  let chartHandle = null;
  /** @type {HTMLElement|null} */
  let chartMount = null;

  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const masked = maskLevel(permissions) === 'masked';
  const canPause =
    can(permissions, 'campaigns:write') || can(permissions, 'campaigns:pause');
  const canWriteCampaign = can(permissions, 'campaigns:write');

  const tgSlot = el('div', { 'data-tg-panel': '' });
  /** @type {{ destroy: () => void }|null} */
  let tgPanelHandle = null;

  const campaignState = { data: null, loading: true, error: null };
  const statsState = { data: null, loading: false, error: null };
  const dashboardState = { data: null, loading: true, error: null };
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
    if (!masked) list.push({ id: 'creative', label: 'Creative' });
    if (!masked) list.push({ id: 'telegram', label: 'Telegram' });
    return list;
  }

  function destroyChart() {
    chartHandle?.destroy();
    chartHandle = null;
  }

  function mountChart(hourly) {
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
        ['Status', 'Budget', 'Spend', 'Pacing'].map((label) =>
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

    chartMount = el('div');

    const crumbs = [{ label: 'Campaigns', href: '/campaigns' }];
    if (campaign.customer_id) {
      crumbs.push({
        label: shortCustomerId(campaign.customer_id, 12),
        href: `/customers/${campaign.customer_id}`,
      });
      touchCustomerContext(campaign.customer_id);
    }
    crumbs.push({ label: campaign.name });

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
            ? el('div', { className: 'flex items-center gap-2 ml-auto' },
              isActive
                ? el('button', {
                  type: 'button',
                  className: 'btn btn--danger btn--sm',
                  disabled: actionLoading,
                  onClick: handlePause,
                },
                  renderIcon('pause', { size: 14 }),
                  'Pause',
                )
                : null,
              isPaused
                ? el('button', {
                  type: 'button',
                  className: 'btn btn--primary btn--sm',
                  disabled: actionLoading,
                  onClick: handleResume,
                },
                  renderIcon('play', { size: 14 }),
                  'Resume',
                )
                : null,
            )
            : null,
        ),
        actionError
          ? el('p', { className: 'text-danger text-sm mt-2' }, actionError)
          : null,
      ),
      renderTabBar({ tabs: tabs(), active: tab, onChange: (t) => {
        if (tab === 'telegram' && t !== 'telegram' && tgPanelHandle) {
          tgPanelHandle.destroy();
          tgPanelHandle = null;
        }
        tab = t;
        if (t === 'stats') statsResource.reload();
        else destroyChart();
        render();
      } }),
      tab === 'overview'
        ? el('div', { className: 'section-block stack' },
          dashboardState.loading
            ? el('span', { className: 'text-muted' }, 'Loading economics…')
            : renderCommercialMetrics(dashboardState.data?.kpis, { masked }),
          !masked && campaign
            ? el('button', {
              type: 'button',
              className: 'btn btn--secondary btn--sm shrink-0',
              onClick: () => openForecastModal({
                campaignId: id,
                customerId: campaign.customer_id,
                budgetMicro: Math.round(Number(campaign.budget_limit ?? 0) * 1_000_000),
                startAt: isoDaysAgo(0),
                endAt: toIsoNow(),
              }),
            }, 'Forecast delivery')
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
              statsState.error.message,
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
        ? el('div', { className: 'section-block' },
          configGrid([
            ['ID', campaign.id],
            ['Customer', campaign.customer_id],
            ['Timezone', campaign.timezone ?? 'UTC'],
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
      tab === 'creative' && !masked
        ? el('div', { className: 'section-block' },
          configGrid([
            ['Target URL', campaign.target_url ?? '—'],
            ['Referrer filter', campaign.referrer_filter ?? '—'],
            [
              'Creative payload',
              campaign.creative_payload
                ? JSON.stringify(campaign.creative_payload)
                : '—',
            ],
          ]),
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
  }

  const campaignResource = createResource(
    () => `/api/v1/campaigns/${id}`,
    {
      onUpdate: (s) => {
        Object.assign(campaignState, s);
        render();
      },
    },
  );

  const statsResource = createResource(
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
      dashboardState.data = res.data;
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
      tgPanelHandle?.destroy();
      campaignResource.destroy();
      statsResource.destroy();
    },
  };
}
