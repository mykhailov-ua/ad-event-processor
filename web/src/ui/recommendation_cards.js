import { el } from '../lib/dom.js';
import { renderStatusBadge } from './status_badge.js';

/**
 * @typedef {{
 *   onAction?: (actionId: string, card: object) => void|Promise<void>,
 *   actionLoading?: boolean,
 * }} RecommendationCardHandlers
 */

/**
 * Render action buttons for a recommendation card.
 *
 * @param {object} card
 * @param {RecommendationCardHandlers} handlers
 * @returns {HTMLElement|null}
 */
function renderCardActions(card, handlers) {
  const actions = card.actions ?? [];
  if (actions.length === 0) {
    if (card.campaign_id) {
      return el('a', {
        href: `/campaigns/${card.campaign_id}`,
        className: 'btn btn--secondary btn--sm',
      }, 'Open campaign');
    }
    return null;
  }
  const row = el('div', { className: 'toolbar-row' });
  for (let i = 0; i < actions.length; i++) {
    const action = actions[i];
    const actionId = action.id ?? '';
    row.appendChild(el('button', {
      type: 'button',
      className: 'btn btn--secondary btn--sm',
      disabled: handlers.actionLoading === true,
      'data-action': actionId,
      onClick: () => {
        if (handlers.onAction) handlers.onAction(actionId, card);
      },
    }, action.label ?? actionId));
  }
  return row;
}

/**
 * Render recommendation cards from buyer portfolio API.
 *
 * @param {Array<{ id: string, type: string, title: string, detail: string, campaign_id?: string, confidence?: number, actions?: object[] }>} cards
 * @param {RecommendationCardHandlers} [handlers]
 * @returns {HTMLElement|null}
 */
export function renderRecommendationCards(cards, handlers = {}) {
  if (!cards?.length) return null;
  const list = el('ul', { className: 'recommendation-list' });
  for (const card of cards) {
    list.appendChild(el('li', { className: 'recommendation-card', 'data-testid': `rec-${card.id}` },
      el('strong', null, card.title),
      el('p', null, card.detail),
      renderCardActions(card, handlers),
      card.confidence != null
        ? el('span', { className: 'text-muted text-xs' }, ` confidence ${Math.round(card.confidence * 100)}%`)
        : null,
    ));
  }
  return el('section', { className: 'section-block', 'data-testid': 'recommendation-cards' },
    el('h3', { className: 'subsection-title' }, 'Recommendations'),
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
  const list = el('ul', { className: 'alert-feed__list' });
  for (const alert of alerts) {
    const tone = alert.level === 'critical' ? 'error' : 'warning';
    list.appendChild(el('li', {
      className: 'alert-feed__item',
      'data-testid': `alert-${alert.id}`,
    },
      renderStatusBadge(tone, { label: alert.title }),
      el('span', null, alert.detail),
      alert.route ? el('a', { href: alert.route, className: 'btn btn--ghost btn--sm' }, 'View') : null,
    ));
  }
  return el('section', { className: 'alert-feed section-block', 'data-testid': 'alert-feed' },
    el('h3', { className: 'alert-feed__title' }, 'Alerts'),
    list,
  );
}
