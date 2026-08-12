import { el } from '../lib/dom.js';
import { buyerEmptyCopy } from '../models/empty_state.js';
import { renderCommercialMetrics, type MetricsBlockDTO } from './commercial_metrics.js';
import { renderStatusBadge } from './status_badge.js';
import {
  renderRecommendationCards,
  renderAlertFeed,
  type RecommendationCard,
  type RecommendationCardHandlers,
  type AlertFeedItem,
} from './recommendation_cards.js';
import { renderFraudKpiTiles, type FraudKpiPayload } from './fraud_kpi_tiles.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId } from '../helpers/buyer_session.js';

export type BuyerAttentionRow = {
  id: string;
  name: string;
  reason: string;
};

export type BuyerPortfolio = {
  active: number;
  paused: number;
  archived: number;
  impressions7d: number;
  clicks7d: number;
  sampled: number;
  overspendCount?: number;
  kpis?: MetricsBlockDTO | null;
  recommendations?: RecommendationCard[];
  alerts?: AlertFeedItem[];
  attention: BuyerAttentionRow[];
};

export type BuyerPerfMetric = {
  count: number;
  nsPerOp: number;
  allocPerOp: number;
  bytesPerOp: number;
};

export type BuyerOverviewState = {
  loading?: boolean;
  portfolio?: BuyerPortfolio | null;
  perf?: Record<string, BuyerPerfMetric>;
  error?: string | null;
  fraud?: FraudKpiPayload | null;
  recActionLoading?: boolean;
};

/**
 * Render the buyer portfolio skeleton panel.
 */
export function renderBuyerOverview(
  state: BuyerOverviewState,
  recHandlers: RecommendationCardHandlers = {},
): HTMLElement {
  const children: HTMLElement[] = [
    el('h2', { className: 'subsection-title' }, 'Buyer portfolio'),
  ];

  if (state.loading) {
    children.push(el('p', { className: 'loading-hint' }, 'Loading portfolio metrics…'));
    return el('section', { className: 'buyer-overview', 'data-testid': 'buyer-overview' }, children);
  }

  if (state.error) {
    const copy = buyerEmptyCopy('campaigns_blocked');
    children.push(
      el('div', { className: 'stack stack--sm' },
        el('p', { className: 'empty-hint' }, copy.title),
        el('p', { className: 'text-muted text-sm' }, copy.description),
        el('p', { className: 'text-sm' }, state.error),
      ),
    );
    return el('section', { className: 'buyer-overview', 'data-testid': 'buyer-overview' }, children);
  }

  const p = state.portfolio;
  if (!p) {
    children.push(el('p', { className: 'empty-hint' }, 'Portfolio metrics unavailable.'));
    return el('section', { className: 'buyer-overview', 'data-testid': 'buyer-overview' }, children);
  }

  const commercial = renderCommercialMetrics(p.kpis, { masked: true });
  if (commercial) children.push(commercial);

  const perms = auth.getUser()?.permissions ?? [];
  if (can(perms, 'audit:read')) {
    children.push(renderFraudKpiTiles({
      loading: false,
      fraud: state.fraud,
      customerId: boundCustomerId(auth.getUser()),
    }));
  }

  if ((p.overspendCount ?? 0) > 0) {
    children.push(el('p', { 'data-testid': 'buyer-overspend-alert' },
      renderStatusBadge('warning', { label: `${p.overspendCount} campaign(s) at overspend risk` }),
    ));
  }

  const recs = renderRecommendationCards(p.recommendations ?? [], {
    ...recHandlers,
    actionLoading: state.recActionLoading === true,
  });
  if (recs) children.push(recs);
  const alerts = renderAlertFeed(p.alerts ?? []);
  if (alerts) children.push(alerts);

  children.push(el('dl', { className: 'definition-list' },
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
  ));

  const attentionItems: HTMLElement[] = [];
  if (p.attention.length === 0) {
    attentionItems.push(el('p', { className: 'text-muted text-sm' },
      'No paused campaigns or pacing flags in the current page.',
    ));
  } else {
    const list = el('ul', { className: 'plain-list' });
    for (let i = 0; i < p.attention.length; i++) {
      const row = p.attention[i];
      list.appendChild(el('li', { className: 'plain-list__item' },
        el('a', { href: `/campaigns/${row.id}` }, row.name),
        ` — ${row.reason}`,
      ));
    }
    attentionItems.push(list);
  }

  children.push(el('div', { className: 'stack stack--sm' },
    el('h3', { className: 'subsection-title' }, 'Needs attention'),
    ...attentionItems,
  ));

  children.push(el('div', { className: 'stack stack--sm' },
    el('h3', { className: 'subsection-title' }, 'Next steps'),
    el('ul', { className: 'plain-list' },
      el('li', { className: 'plain-list__item' }, el('a', { href: '/campaigns/portfolio' }, 'Portfolio view (drift sort)')),
      el('li', { className: 'plain-list__item' }, el('a', { href: '/campaigns' }, 'Review campaign delivery')),
      el('li', { className: 'plain-list__item' }, el('a', { href: '/reports' }, 'Reports')),
      el('li', { className: 'plain-list__item' }, el('a', { href: '/reports/placements' }, 'Check placement report')),
      el('li', { className: 'plain-list__item' }, el('a', { href: '/reports/keywords' }, 'Check keyword report')),
      can(perms, 'audit:read')
        ? el('li', { className: 'plain-list__item' }, el('a', {
          href: `/reports/ivt-by-source?customer_id=${encodeURIComponent(boundCustomerId(auth.getUser()) ?? '')}`,
        }, 'IVT by source'))
        : null,
      can(perms, 'audit:read')
        ? el('li', { className: 'plain-list__item' }, el('a', {
          href: `/dashboards/fraud?customer_id=${encodeURIComponent(boundCustomerId(auth.getUser()) ?? '')}`,
        }, 'Fraud dashboard'))
        : null,
    ),
  ));

  if (state.perf && Object.keys(state.perf).length > 0) {
    children.push(el('div', { className: 'stack stack--sm' },
      el('h3', { className: 'subsection-title' }, 'Critical path metrics'),
      el('pre', {
        id: 'buyer-perf-metrics',
        className: 'code-block',
        'aria-label': 'Critical path probe metrics',
      }, JSON.stringify(state.perf, null, 2)),
    ));
  }

  return el('section', { className: 'buyer-overview', 'data-testid': 'buyer-overview' }, children);
}
