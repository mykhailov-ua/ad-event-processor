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

import { renderIcon } from '../ui/icon.js';

/**
 * @param {Array<[string, string]>} rows
 */
function configGrid(rows) {
  return el('div', {
    style: {
      display: 'grid',
      gridTemplateColumns: 'auto 1fr',
      gap: '12px 24px',
      fontSize: 13,
    },
  },
    rows.flatMap(([label, value]) => [
      el('span', { className: 'text-muted' }, label),
      el('span', { className: 'text-secondary font-mono' }, value),
    ]),
  );
}

/**
 * @param {HTMLElement} container
 * @param {{ params: Record<string, string> }} ctx
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

  const campaignState = { data: null, loading: true, error: null };
  const statsState = { data: null, loading: false, error: null };

  function tabs() {
    const list = [
      { id: 'overview', label: 'Overview' },
      { id: 'stats', label: 'Statistics' },
      { id: 'config', label: 'Configuration' },
    ];
    if (!masked) list.push({ id: 'creative', label: 'Creative' });
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
    actionLoading = true;
    actionError = null;
    render();
    const [, pauseErr] = await to(pauseCampaign(id));
    if (pauseErr) {
      if (pauseErr instanceof ConfirmCancelledError) {
        actionLoading = false;
        render();
        return;
      }
      actionError = pauseErr.message || 'Failed to pause campaign';
      actionLoading = false;
      render();
      return;
    }
    const [, pollErr] = await to(pollCampaignStatus(id, 'PAUSED'));
    if (pollErr) {
      actionError = pollErr.message || 'Failed to pause campaign';
    } else {
      campaignResource.reload();
    }
    actionLoading = false;
    render();
  }

  async function handleResume() {
    actionLoading = true;
    actionError = null;
    render();
    const [, resumeErr] = await to(resumeCampaign(id));
    if (resumeErr) {
      if (resumeErr instanceof ConfirmCancelledError) {
        actionLoading = false;
        render();
        return;
      }
      actionError = resumeErr.message || 'Failed to resume campaign';
      actionLoading = false;
      render();
      return;
    }
    const [, pollErr] = await to(pollCampaignStatus(id, 'ACTIVE'));
    if (pollErr) {
      actionError = pollErr.message || 'Failed to resume campaign';
    } else {
      campaignResource.reload();
    }
    actionLoading = false;
    render();
  }

  function renderLoadingCards() {
    replaceChildren(container,
      el('div', { className: 'grid-stats' },
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
            ? el('div', { className: 'flex items-center gap-2', style: { marginLeft: 'auto' } },
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
          ? el('div', {
            className: 'text-muted',
            style: { color: 'var(--error)', fontSize: 13, marginTop: 8 },
          }, actionError)
          : null,
      ),
      renderTabBar({ tabs: tabs(), active: tab, onChange: (t) => {
        tab = t;
        if (t === 'stats') statsResource.reload();
        else destroyChart();
        render();
      } }),
      tab === 'overview'
        ? el('div', { className: 'grid-stats', style: { marginTop: 24 } },
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
            el('div', { className: 'metric-card__value' }, campaign.pacing_mode ?? '—'),
          ),
        )
        : null,
      tab === 'stats'
        ? el('div', { style: { marginTop: 24 } },
          statsState.loading ? el('span', { className: 'text-muted' }, 'Loading statistics…') : null,
          statsState.error
            ? el('div', { className: 'error-page__desc', style: { color: 'var(--error)' } },
              statsState.error.message,
            )
            : null,
          statsState.data
            ? el('div', null,
              el('div', { className: 'flex items-center gap-2 mb-4' },
                el('h2', { style: { fontSize: 14, fontWeight: 600 } }, 'Hourly metrics'),
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
                    formatUsdDecimal(statsState.data.current_spend ?? '0.00'),
                  ),
                ),
              ),
              chartMount,
            )
            : null,
        )
        : null,
      tab === 'config'
        ? el('div', { className: 'mb-4', style: { marginTop: 24 } },
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
        ? el('div', { className: 'mb-4', style: { marginTop: 24 } },
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
    ];

    replaceChildren(container, ...children);

    if (tab === 'stats' && statsState.data) {
      mountChart(statsState.data.hourly ?? []);
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
    () => `/api/v1/campaigns/${id}/stats?granularity=hour`,
    {
      skip: () => tab !== 'stats',
      onUpdate: (s) => {
        Object.assign(statsState, s);
        render();
      },
    },
  );

  render();

  return {
    destroy() {
      destroyed = true;
      destroyChart();
      campaignResource.destroy();
      statsResource.destroy();
    },
  };
}
