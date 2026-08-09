import { el } from '../lib/dom.js';
import { formatMicro } from '../helpers/money.js';
import { renderFreshnessBadge } from './freshness_badge.js';

/**
 * @typedef {{
 *   spend_micro?: number,
 *   revenue_micro?: number,
 *   profit_micro?: number,
 *   conversions?: number,
 *   cpa_micro?: number,
 *   roi_pct?: number,
 *   freshness?: { stale?: boolean, ch_lag_seconds?: number },
 * }} MetricsBlockDTO
 */

/**
 * Render a row of commercial KPI cards from server metrics.
 *
 * @param {MetricsBlockDTO|null|undefined} kpis
 * @param {{ masked?: boolean }} [opts]
 * @returns {HTMLElement|null}
 */
export function renderCommercialMetrics(kpis, opts = {}) {
  if (!kpis) return null;
  const masked = opts.masked === true;
  const cards = [];

  if (kpis.spend_micro != null && !masked) {
    cards.push(metricCard('Spend', '$' + formatMicro(kpis.spend_micro)));
  }
  if (kpis.revenue_micro != null && kpis.revenue_micro > 0) {
    cards.push(metricCard('Revenue', '$' + formatMicro(kpis.revenue_micro)));
  }
  if (kpis.profit_micro != null && kpis.profit_micro !== 0) {
    cards.push(metricCard('Profit', '$' + formatMicro(kpis.profit_micro)));
  }
  if (kpis.conversions != null) {
    cards.push(metricCard('Conversions', String(kpis.conversions)));
  }
  if (kpis.cpa_micro != null && kpis.cpa_micro > 0 && !masked) {
    cards.push(metricCard('CPA', '$' + formatMicro(kpis.cpa_micro)));
  }
  if (kpis.roi_pct != null && kpis.roi_pct !== 0) {
    cards.push(metricCard('ROI', `${kpis.roi_pct.toFixed(1)}%`));
  }

  if (cards.length === 0) return null;

  return el('div', { className: 'grid-stats', 'data-testid': 'commercial-metrics' },
    ...cards,
    kpis.freshness ? el('div', { className: 'metric-card metric-card--freshness' }, renderFreshnessBadge(kpis.freshness)) : null,
  );
}

/**
 * Render one KPI metric card.
 *
 * @param {string} label
 * @param {string} value
 * @returns {HTMLElement}
 */
function metricCard(label, value) {
  return el('div', { className: 'metric-card' },
    el('div', { className: 'metric-card__head' },
      el('div', { className: 'metric-card__label' }, label),
    ),
    el('div', { className: 'metric-card__value font-mono' }, value),
  );
}
