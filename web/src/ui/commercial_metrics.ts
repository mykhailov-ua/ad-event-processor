import { el } from '../lib/dom.js';
import { formatMicro } from '../helpers/money.js';
import { renderFreshnessBadge } from './freshness_badge.js';

export type MetricsFreshness = {
  stale?: boolean;
  ch_lag_seconds?: number;
  lagSeconds?: number;
};

export type MetricsBlockDTO = {
  spend_micro?: number;
  revenue_micro?: number;
  profit_micro?: number;
  conversions?: number;
  cpa_micro?: number;
  roi_pct?: number;
  freshness?: MetricsFreshness;
};

export type CommercialMetricsOpts = {
  masked?: boolean;
};

/**
 * Render a row of commercial KPI cards from server metrics.
 */
export function renderCommercialMetrics(
  kpis: MetricsBlockDTO | null | undefined,
  opts: CommercialMetricsOpts = {},
): HTMLElement | null {
  if (!kpis) return null;
  const masked = opts.masked === true;
  const cards: HTMLElement[] = [];

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
    kpis.freshness
      ? el('div', { className: 'metric-card metric-card--freshness' },
        renderFreshnessBadge({
          stale: kpis.freshness.stale,
          lagSeconds: kpis.freshness.lagSeconds ?? kpis.freshness.ch_lag_seconds,
        }),
      )
      : null,
  );
}

/**
 * Render one KPI metric card.
 */
function metricCard(label: string, value: string): HTMLElement {
  return el('div', { className: 'metric-card' },
    el('div', { className: 'metric-card__head' },
      el('div', { className: 'metric-card__label' }, label),
    ),
    el('div', { className: 'metric-card__value font-mono' }, value),
  );
}
