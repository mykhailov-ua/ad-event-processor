import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api, ApiError } from '../helpers/api_client.js';
import { parallelAll } from '../helpers/request_multiplex.js';
import { can, isBuyer } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId, hasBoundCustomer } from '../helpers/buyer_session.js';
import { fetchBuyerDashboard } from '../helpers/buyer_dashboard.js';
import { probeReport, probeReset } from '../helpers/perf_probe.js';
import { renderBuyerOverview } from '../ui/buyer_overview.js';
import { pauseCampaign, resumeCampaign } from '../helpers/campaign_actions.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { invalidateBuyerDashboard } from '../helpers/buyer_dashboard.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderFreshnessBadge } from '../ui/freshness_badge.js';
import { renderDoctorPanel } from '../ui/doctor_panel.js';
import { renderIcon } from '../ui/icon.js';
import { renderAlertFeed } from '../ui/recommendation_cards.js';
import { renderStatusHint } from '../ui/status_hint.js';
import { displayLabel, formatYesNo } from '../helpers/display_labels.js';
import { buildHomeAlerts } from '../helpers/home_alerts.js';
import { connectOpsLiveFeed } from '../helpers/ops_live_feed.js';

/**
 * Render one overview KPI metric card.
 *
 * @param {string} label
 * @param {string} value
 * @param {string} icon
 * @returns {HTMLElement}
 */
function renderOverviewMetric(label, value, icon) {
  return el('div', { className: 'metric-card' },
    el('div', { className: 'metric-card__head' },
      el('div', { className: 'metric-card__label' }, label),
      renderIcon(icon, { size: 16, className: 'text-muted' }),
    ),
    el('div', { className: 'metric-card__value font-mono' }, value),
  );
}

/**
 * Build permission-filtered quick navigation links for the overview page.
 *
 * @param {string[]} perms
 * @returns {HTMLElement|null}
 */
function buildQuickLinks(perms) {
  const links = [];
  if (can(perms, 'campaigns:read') || can(perms, 'campaigns:read:masked')) {
    links.push({ href: '/campaigns', label: 'Campaigns', icon: 'megaphone' });
  }
  if (can(perms, 'customers:read') || can(perms, 'billing:read')) {
    links.push({ href: '/billing', label: 'Billing', icon: 'credit-card' });
  }
  if (can(perms, 'shards:read')) {
    links.push({ href: '/ops', label: 'Operations', icon: 'server' });
  }
  if (can(perms, 'settings:read')) {
    links.push({ href: '/settings', label: 'Settings', icon: 'settings' });
  }
  if (links.length === 0) return null;
  return el('div', { className: 'page-header__links' },
    links.map((l) =>
      el('a', { href: l.href, className: 'btn btn--secondary btn--sm' },
        renderIcon(l.icon, { size: 14 }),
        l.label,
      ),
    ),
  );
}

/**
 * Mount the operator overview dashboard with health and quick links.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container) {
  let destroyed = false;
  const user = auth.getUser();
  const perms = user?.permissions ?? [];
  const buyerMode = isBuyer(user?.role);
  const canOps = can(perms, 'shards:read');

  const state = {
    loading: true,
    blockError: null,
    summary: null,
    doctor: null,
    incidents: null,
    meta: null,
    partialErrors: [],
    partialDismissed: false,
    buyerPortfolio: null,
    buyerError: null,
    buyerPerf: null,
    recActionLoading: false,
    homeAlerts: [],
  };

  /**
   * Handle a recommendation card action (pause, resume, or navigate).
   *
   * @param {string} actionId
   * @param {{ campaign_id?: string }} card
   * @returns {Promise<void>}
   */
  async function handleRecommendationAction(actionId, card) {
    const campaignId = card.campaign_id;
    if (!campaignId) return;
    if (actionId === 'edit_budget') {
      window.location.href = `/campaigns/${campaignId}`;
      return;
    }
    state.recActionLoading = true;
    render();
    const [, err] = await to((async () => {
      if (actionId === 'pause') await pauseCampaign(campaignId);
      else if (actionId === 'resume') await resumeCampaign(campaignId);
    })());
    state.recActionLoading = false;
    if (err && !(err instanceof ConfirmCancelledError)) {
      pushToastMessage({ title: 'Action failed', message: err.message ?? String(err) });
    }
    if (!err || err instanceof ConfirmCancelledError) {
      invalidateBuyerDashboard(boundCustomerId(user));
      const [portfolio, loadErr] = await to(fetchBuyerDashboard(boundCustomerId(user)));
      if (!loadErr && portfolio) {
        state.buyerPortfolio = portfolio;
        state.buyerPerf = probeReport();
      }
    }
    render();
  }

  function deriveFreshness() {
    if (state.incidents?.partial) {
      return { stale: true, lagSeconds: 0 };
    }
    const chCard = state.summary?.services?.find((s) =>
      (s.name || '').toLowerCase().includes('clickhouse'),
    );
    if (chCard?.status && chCard.status !== 'ok' && chCard.status !== 'disabled') {
      return { stale: true, lagSeconds: 0 };
    }
    const chCheck = state.doctor?.checks?.find((c) =>
      (c.name || c.id || '').toLowerCase().includes('clickhouse'),
    );
    if (chCheck?.status && chCheck.status !== 'ok' && chCheck.status !== 'pass') {
      return { stale: true, lagSeconds: 0 };
    }
    return null;
  }

  function render() {
    if (destroyed) return;
    if (state.blockError) {
      replaceChildren(container, renderErrorBlock(state.blockError));
      return;
    }

    const freshness = deriveFreshness();
    const children = [
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            el('h1', { className: 'page-header__title' }, 'Overview'),
            freshness ? renderFreshnessBadge(freshness) : null,
          ),
          state.meta?.version
            ? el('span', { className: 'text-muted text-sm' }, `v${state.meta.version}`)
            : null,
        ),
        buildQuickLinks(perms),
      ),
      state.partialErrors.length > 0 && !state.partialDismissed
        ? renderStatusHint({
            tone: 'error',
            message: el('div', { className: 'flex items-center justify-between w-full' },
              el('span', null, `Partial degradation: ${state.partialErrors.join('; ')}`),
              el('button', {
                type: 'button',
                className: 'alert-banner__close',
                onClick: () => {
                  state.partialDismissed = true;
                  render();
                },
              }, 'Dismiss'),
            ),
          })
        : null,
      state.loading ? el('div', { className: 'text-muted' }, 'Loading…') : null,
      !state.loading && (canOps || buyerMode) && state.homeAlerts.length > 0
        ? renderAlertFeed(state.homeAlerts)
        : null,
      !state.loading && canOps && state.summary
        ? el('div', { className: 'grid-stats' },
          renderOverviewMetric('Outbox pending', String(state.summary.outbox_pending ?? 0), 'activity'),
          renderOverviewMetric('RPS (estimate)', String(state.summary.rps_estimate ?? '—'), 'zap'),
          renderOverviewMetric('Drift alert', formatYesNo(state.summary.drift_alert), 'alert-triangle'),
          renderOverviewMetric('Emergency breaker', displayLabel(state.summary.emergency_breaker), 'shield'),
        )
        : null,
      canOps
        ? renderDoctorPanel({
          doctor: state.doctor,
          services: state.summary?.services,
          loading: state.loading,
        })
        : null,
      !state.loading && buyerMode
        ? renderBuyerOverview({
          loading: false,
          portfolio: state.buyerPortfolio,
          perf: state.buyerPerf,
          error: state.buyerError,
          recActionLoading: state.recActionLoading,
        }, { onAction: handleRecommendationAction })
        : null,
      !state.loading && !canOps && !buyerMode
        ? el('p', { className: 'text-muted' },
          'Use the quick links above to manage campaigns and billing.',
        )
        : null,
    ];
    replaceChildren(container, ...children);
  }

  async function loadData() {
    state.loading = true;
    state.blockError = null;

    const tasks = [() => api('/api/v1/meta')];
    let buyerTaskIndex = -1;
    if (canOps) {
      tasks.push(
        () => api('/api/v1/ops/doctor'),
        () => api('/api/v1/ops/incidents').catch((err) => ({ error: err })),
        () => api('/api/v1/ops/dashboard/summary'),
      );
    }
    if (buyerMode) {
      const customerId = boundCustomerId(user);
      buyerTaskIndex = tasks.length;
      tasks.push(async () => {
        if (!customerId) return { error: new Error('customer_id missing in session') };
        probeReset();
        const [portfolio, err] = await to(fetchBuyerDashboard(customerId));
        if (err) return { error: err };
        return portfolio;
      });
    }

    const [results, err] = await to(parallelAll(tasks, 3));
    if (destroyed) return;

    if (err) {
      state.blockError = err;
      state.loading = false;
      render();
      return;
    }

    const metaRes = results[0];
    if (metaRes?.data) state.meta = metaRes.data;

    if (canOps) {
      const docRes = results[1];
      const incRes = results[2];
      const sumRes = results[3];

      if (docRes?.data) state.doctor = docRes.data;
      if (sumRes?.data) state.summary = sumRes.data;
      if (incRes?.data && !incRes.error) state.incidents = incRes.data;

      const errors = [];
      if (incRes?.error) {
        const incErr = incRes.error;
        if (incErr instanceof ApiError && incErr.payload?.errors?.length) {
          errors.push(...incErr.payload.errors);
        } else {
          const view = mapServiceError(incErr);
          pushToastMessage({ title: view.title, message: view.message, code: view.code });
        }
      } else if (incRes?.data?.errors?.length) {
        errors.push(...incRes.data.errors);
      }
      state.partialErrors = errors;
    }

    if (buyerMode && buyerTaskIndex >= 0) {
      const buyerRes = results[buyerTaskIndex];
      if (buyerRes?.error) {
        state.buyerError = buyerRes.error.message || 'Failed to load buyer portfolio';
      } else {
        state.buyerPortfolio = buyerRes;
        state.buyerPerf = probeReport();
      }
    }

    state.homeAlerts = buildHomeAlerts({
      summary: state.summary,
      doctor: state.doctor,
      incidents: state.incidents,
      meta: state.meta,
      buyerPortfolio: state.buyerPortfolio,
      canOps,
      buyerMode,
    });

    state.loading = false;
    render();
  }

  render();
  loadData();

  let opsLiveFeed = null;
  if (canOps) {
    opsLiveFeed = connectOpsLiveFeed({
      pollMs: 30_000,
      onTick: (payload) => {
        if (destroyed || !payload.summary) return;
        state.summary = payload.summary;
        state.homeAlerts = buildHomeAlerts({
          summary: state.summary,
          doctor: state.doctor,
          incidents: state.incidents,
          meta: state.meta,
          buyerPortfolio: state.buyerPortfolio,
          canOps,
          buyerMode,
        });
        render();
      },
      onPoll: async () => {
        if (destroyed) return;
        const [sumRes] = await to(api('/api/v1/ops/dashboard/summary'));
        if (destroyed || !sumRes?.data) return;
        state.summary = sumRes.data;
        state.homeAlerts = buildHomeAlerts({
          summary: state.summary,
          doctor: state.doctor,
          incidents: state.incidents,
          meta: state.meta,
          buyerPortfolio: state.buyerPortfolio,
          canOps,
          buyerMode,
        });
        render();
      },
    });
  }

  return {
    destroy() {
      destroyed = true;
      opsLiveFeed?.destroy();
    },
  };
}
