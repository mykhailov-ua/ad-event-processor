import { el } from '../lib/dom.js';
import { renderStatusBadge } from './status_badge.js';

/**
 * Render recommendation cards from buyer portfolio API.
 *
 * @param {Array<{ id: string, type: string, title: string, detail: string, campaign_id?: string, confidence?: number }>} cards
 * @returns {HTMLElement|null}
 */
export function renderRecommendationCards(cards) {
  if (!cards?.length) return null;
  const list = el('ul', { className: 'recommendation-list' });
  for (const card of cards) {
    list.appendChild(el('li', { className: 'recommendation-card', 'data-testid': `rec-${card.id}` },
      el('strong', null, card.title),
      el('p', null, card.detail),
      card.campaign_id
        ? el('a', { href: `/campaigns/${card.campaign_id}`, className: 'btn btn--secondary btn--sm' }, 'Open campaign')
        : null,
      card.confidence != null
        ? el('span', { className: 'text-muted', style: { fontSize: 12 } }, ` confidence ${Math.round(card.confidence * 100)}%`)
        : null,
    ));
  }
  return el('section', { 'data-testid': 'recommendation-cards' },
    el('h3', null, 'Recommendations'),
    list,
  );
}

/**
 * Render alert feed cards.
 *
 * @param {Array<{ id: string, level: string, title: string, detail: string, route?: string }>} alerts
 * @returns {HTMLElement|null}
 */
export function renderAlertFeed(alerts) {
  if (!alerts?.length) return null;
  const list = el('ul', null);
  for (const alert of alerts) {
    const tone = alert.level === 'critical' ? 'error' : 'warning';
    list.appendChild(el('li', { 'data-testid': `alert-${alert.id}` },
      renderStatusBadge(tone, { label: alert.title }),
      ' ',
      alert.detail,
      alert.route ? [' ', el('a', { href: alert.route }, 'View')] : null,
    ));
  }
  return el('section', { 'data-testid': 'alert-feed' },
    el('h3', null, 'Alerts'),
    list,
  );
}
