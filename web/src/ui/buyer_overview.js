import { el } from '../lib/dom.js';
import { buyerEmptyCopy } from '../models/empty_state.js';
import { renderCommercialMetrics } from './commercial_metrics.js';
import { renderStatusBadge } from './status_badge.js';
import { renderRecommendationCards, renderAlertFeed } from './recommendation_cards.js';

/**
 * Render the buyer portfolio skeleton panel.
 *
 * @param {{
 *   loading?: boolean,
 *   portfolio?: {
 *     active: number,
 *     paused: number,
 *     archived: number,
 *     impressions7d: number,
 *     clicks7d: number,
 *     sampled: number,
 *     overspendCount?: number,
 *     kpis?: object|null,
 *     recommendations?: object[],
 *     alerts?: object[],
 *     attention: Array<{ id: string, name: string, reason: string }>,
 *   }|null,
 *   perf?: Record<string, { count: number, nsPerOp: number, allocPerOp: number, bytesPerOp: number }>,
 *   error?: string|null,
 * }} state
 * @returns {HTMLElement}
 */
export function renderBuyerOverview(state) {
  const root = el('section', { 'data-testid': 'buyer-overview' },
    el('h2', null, 'Buyer portfolio'),
  );

  if (state.loading) {
    root.appendChild(el('p', null, 'Loading portfolio metrics…'));
    return root;
  }

  if (state.error) {
    const copy = buyerEmptyCopy('campaigns_blocked');
    root.appendChild(el('p', null, copy.title));
    root.appendChild(el('p', null, copy.description));
    root.appendChild(el('p', null, state.error));
    return root;
  }

  const p = state.portfolio;
  if (!p) {
    root.appendChild(el('p', null, 'Portfolio metrics unavailable.'));
    return root;
  }

  const commercial = renderCommercialMetrics(p.kpis, { masked: true });
  if (commercial) root.appendChild(commercial);

  if (p.overspendCount > 0) {
    root.appendChild(el('p', { 'data-testid': 'buyer-overspend-alert' },
      renderStatusBadge('warning', { label: `${p.overspendCount} campaign(s) at overspend risk` }),
    ));
  }

  const recs = renderRecommendationCards(p.recommendations ?? []);
  if (recs) root.appendChild(recs);
  const alerts = renderAlertFeed(p.alerts ?? []);
  if (alerts) root.appendChild(alerts);

  const metrics = el('dl', null,
    el('dt', null, 'Active campaigns'),
    el('dd', { id: 'buyer-metric-active' }, String(p.active)),
    el('dt', null, 'Paused campaigns'),
    el('dd', { id: 'buyer-metric-paused' }, String(p.paused)),
    el('dt', null, 'Archived campaigns'),
    el('dd', { id: 'buyer-metric-archived' }, String(p.archived)),
    el('dt', null, 'Impressions (7d)'),
    el('dd', { id: 'buyer-metric-impressions' }, String(p.impressions7d)),
    el('dt', null, 'Clicks (7d)'),
    el('dd', { id: 'buyer-metric-clicks' }, String(p.clicks7d)),
    el('dt', null, 'Campaigns in portfolio'),
    el('dd', null, String(p.sampled)),
  );
  root.appendChild(metrics);

  root.appendChild(el('h3', null, 'Needs attention'));
  if (p.attention.length === 0) {
    root.appendChild(el('p', null, 'No paused campaigns or pacing flags in the current page.'));
  } else {
    const list = el('ul', null);
    for (let i = 0; i < p.attention.length; i++) {
      const row = p.attention[i];
      list.appendChild(el('li', null,
        el('a', { href: `/campaigns/${row.id}` }, row.name),
        ` — ${row.reason}`,
      ));
    }
    root.appendChild(list);
  }

  root.appendChild(el('h3', null, 'Next steps'));
  root.appendChild(el('ul', null,
    el('li', null, el('a', { href: '/campaigns/portfolio' }, 'Portfolio view (drift sort)')),
    el('li', null, el('a', { href: '/campaigns' }, 'Review campaign delivery')),
    el('li', null, el('a', { href: '/reports' }, 'Reports hub')),
    el('li', null, el('a', { href: '/reports/placements' }, 'Check placement report')),
    el('li', null, el('a', { href: '/reports/keywords' }, 'Check keyword report')),
  ));

  if (state.perf && Object.keys(state.perf).length > 0) {
    const perfEl = el('pre', {
      id: 'buyer-perf-metrics',
      'aria-label': 'Critical path probe metrics',
    }, JSON.stringify(state.perf, null, 2));
    root.appendChild(el('h3', null, 'Critical path metrics'));
    root.appendChild(perfEl);
  }

  return root;
}
