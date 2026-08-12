import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

/**
 * One clickable overview metric card.
 */
function renderLinkedMetric(label: string, value: string, icon: string, href: string): HTMLElement {
  return el('a', {
    href,
    className: 'metric-card metric-card--link',
    'data-testid': `fraud-kpi-${label.toLowerCase().replace(/\s+/g, '-')}`,
  },
    el('div', { className: 'metric-card__head' },
      el('div', { className: 'metric-card__label' }, label),
      renderIcon(icon, { size: 16, className: 'text-muted' }),
    ),
    el('div', { className: 'metric-card__value font-mono' }, value),
  );
}

export type FraudKpiGeoHint = {
  ivt_rate?: number;
};

export type FraudKpiPayload = {
  ghost_ivt_campaigns?: number;
  edge_blocked_fraud?: number;
  geo_hints?: FraudKpiGeoHint[];
};

export type FraudKpiTilesState = {
  loading?: boolean;
  fraud?: FraudKpiPayload | null;
  customerId?: string | null;
};

/**
 * Fraud / IVT KPI tiles with drill-down to fraud dashboard and IVT report.
 */
export function renderFraudKpiTiles(state: FraudKpiTilesState): HTMLElement {
  const qs = state.customerId
    ? `?customer_id=${encodeURIComponent(state.customerId)}`
    : '';
  const fraud = state.fraud;
  const highIvt = fraud?.geo_hints
    ? fraud.geo_hints.filter((h) => Number(h.ivt_rate ?? 0) >= 0.1).length
    : null;

  const ghost = state.loading ? '…' : String(fraud?.ghost_ivt_campaigns ?? '—');
  const blocked = state.loading ? '…' : String(fraud?.edge_blocked_fraud ?? '—');
  const geo = state.loading ? '…' : (highIvt != null ? String(highIvt) : '—');

  return el('section', {
    className: 'fraud-kpi-section section-block',
    'data-testid': 'fraud-kpi-tiles',
  },
    el('h2', { className: 'subsection-title' }, 'Fraud & IVT'),
    el('p', { className: 'text-muted text-sm' },
      state.customerId
        ? 'Customer-scoped signals (7d). Open dashboards for detail.'
        : 'Open customer-scoped fraud views (enter customer on destination).',
    ),
    el('div', { className: 'grid-stats' },
      renderLinkedMetric('Ghost IVT campaigns', ghost, 'alert-triangle', `/dashboards/fraud${qs}`),
      renderLinkedMetric('Edge blocked (fraud)', blocked, 'shield', `/dashboards/fraud${qs}`),
      renderLinkedMetric('High-IVT geo hints', geo, 'globe', `/reports/ivt-by-source${qs}`),
    ),
  );
}
