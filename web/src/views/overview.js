import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api, ApiError } from '../helpers/api_client.js';
import { parallelAll } from '../helpers/request_multiplex.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderFreshnessBadge } from '../ui/freshness_badge.js';
import { renderAlertBanner } from '../ui/alert_banner.js';
import { renderDoctorPanel } from '../ui/doctor_panel.js';
import { renderIcon } from '../ui/icon.js';
import { renderSectionCard } from '../ui/section_card.js';
import { renderStatusHint } from '../ui/status_hint.js';

/**
 * @param {string[]} perms
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
  return el('div', { className: 'flex items-center gap-2 mt-4', style: { flexWrap: 'wrap' } },
    links.map((l) =>
      el('a', { href: l.href, className: 'btn btn--secondary btn--sm' },
        renderIcon(l.icon, { size: 14 }),
        l.label,
      ),
    ),
  );
}

/**
 * @param {HTMLElement} container
 */
export function mount(container) {
  let destroyed = false;
  const user = auth.getUser();
  const perms = user?.permissions ?? [];
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
  };

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

    const license = state.meta?.license;
    const freshness = deriveFreshness();
    const children = [
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('h1', { className: 'page-header__title' }, 'Overview'),
          freshness ? renderFreshnessBadge(freshness) : null,
          state.meta?.version
            ? el('span', { className: 'text-muted', style: { fontSize: 13 } }, `v${state.meta.version}`)
            : null,
        ),
        buildQuickLinks(perms),
      ),
      state.partialErrors.length > 0 && !state.partialDismissed
        ? renderStatusHint({
            tone: 'error',
            message: el('div', { style: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%' } },
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
      license?.state && license.state.toLowerCase() !== 'valid' && license.state.toLowerCase() !== 'active'
        ? renderAlertBanner({
          variant: 'warning',
          message: `License: ${license.state}${license.valid_until ? ` · until ${license.valid_until}` : ''}`,
        })
        : null,
      state.loading ? el('span', { className: 'text-muted' }, 'Loading…') : null,
      !state.loading && canOps && state.summary
        ? el('div', { className: 'grid-stats' },
          renderSectionCard({
            className: 'metric-card',
            children: [
              el('div', { className: 'flex items-center justify-between mb-2' },
                el('div', { className: 'metric-card__label' }, 'Outbox pending'),
                renderIcon('activity', { size: 16, className: 'text-muted' }),
              ),
              el('div', { className: 'metric-card__value font-mono' }, String(state.summary.outbox_pending ?? 0)),
            ]
          }),
          renderSectionCard({
            className: 'metric-card',
            children: [
              el('div', { className: 'flex items-center justify-between mb-2' },
                el('div', { className: 'metric-card__label' }, 'RPS (estimate)'),
                renderIcon('zap', { size: 16, className: 'text-muted' }),
              ),
              el('div', { className: 'metric-card__value font-mono' }, String(state.summary.rps_estimate ?? '—')),
            ]
          }),
          renderSectionCard({
            className: 'metric-card',
            children: [
              el('div', { className: 'flex items-center justify-between mb-2' },
                el('div', { className: 'metric-card__label' }, 'Drift alert'),
                renderIcon('alert-triangle', { size: 16, className: 'text-muted' }),
              ),
              el('div', { className: 'metric-card__value' }, state.summary.drift_alert ? 'Yes' : 'No'),
            ]
          }),
          renderSectionCard({
            className: 'metric-card',
            children: [
              el('div', { className: 'flex items-center justify-between mb-2' },
                el('div', { className: 'metric-card__label' }, 'Emergency breaker'),
                renderIcon('shield', { size: 16, className: 'text-muted' }),
              ),
              el('div', { className: 'metric-card__value font-mono' }, state.summary.emergency_breaker || '—'),
            ]
          }),
        )
        : null,
      canOps
        ? renderDoctorPanel({
          doctor: state.doctor,
          services: state.summary?.services,
          loading: state.loading,
        })
        : null,
      !state.loading && !canOps
        ? el('p', { className: 'text-muted', style: { marginTop: 24, fontSize: 14 } },
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
    if (canOps) {
      tasks.push(
        () => api('/api/v1/ops/doctor'),
        () => api('/api/v1/ops/incidents').catch((err) => ({ error: err })),
        () => api('/api/v1/ops/dashboard/summary'),
      );
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

    state.loading = false;
    render();
  }

  render();
  loadData();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
