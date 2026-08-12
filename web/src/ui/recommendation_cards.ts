import { el } from '../lib/dom.js';
import { renderStatusBadge } from './status_badge.js';
import { dismissAlert, isAlertDismissed } from '../helpers/alert_dismiss.js';
import { renderButton, renderButtonLink } from './button.js';

export type RecommendationAction = {
  id?: string;
  label?: string;
};

export type RecommendationCard = {
  id: string;
  type: string;
  title: string;
  detail: string;
  campaign_id?: string;
  confidence?: number;
  actions?: RecommendationAction[];
};

export type RecommendationCardHandlers = {
  onAction?: ((actionId: string, card: RecommendationCard) => void | Promise<void>) | null;
  actionLoading?: boolean;
};

/**
 * Render action buttons for a recommendation card.
 */
function renderCardActions(
  card: RecommendationCard,
  handlers: RecommendationCardHandlers,
): HTMLElement | null {
  const actions = card.actions ?? [];
  if (actions.length === 0) {
    if (card.campaign_id) {
      return renderButtonLink({
        href: `/campaigns/${card.campaign_id}`,
        label: 'Open campaign',
        variant: 'secondary',
        size: 'sm',
      });
    }
    return null;
  }
  const row = el('div', { className: 'toolbar-row' });
  for (let i = 0; i < actions.length; i++) {
    const action = actions[i];
    const actionId = action.id ?? '';
    row.appendChild(renderButton({
      label: action.label ?? actionId,
      variant: 'secondary',
      size: 'sm',
      action: actionId,
      disabled: handlers.actionLoading === true,
      onClick: () => {
        if (handlers.onAction) handlers.onAction(actionId, card);
      },
    }));
  }
  return row;
}

/**
 * Render recommendation cards from buyer portfolio API.
 */
export function renderRecommendationCards(
  cards: RecommendationCard[] | null | undefined,
  handlers: RecommendationCardHandlers = {},
): HTMLElement | null {
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

export type AlertFeedItem = {
  id: string;
  level: string;
  title: string;
  detail: string;
  route?: string;
};

export type AlertFeedOpts = {
  onDismiss?: (() => void) | null;
};

/**
 * Render alert feed cards (session-dismissible via alert_dismiss.js).
 */
export function renderAlertFeed(
  alerts: AlertFeedItem[] | null | undefined,
  opts: AlertFeedOpts = {},
): HTMLElement | null {
  const visible = (alerts ?? []).filter((a) => a?.id && !isAlertDismissed(`home.${a.id}`));
  if (!visible.length) return null;
  const list = el('ul', { className: 'alert-feed__list' });
  for (const alert of visible) {
    const tone = alert.level === 'critical' ? 'error' : 'warning';
    list.appendChild(el('li', {
      className: 'alert-feed__item',
      'data-testid': `alert-${alert.id}`,
    },
      renderStatusBadge(tone, { label: alert.title }),
      el('span', null, alert.detail),
      el('div', { className: 'alert-feed__item-actions' },
        alert.route
          ? renderButtonLink({ href: alert.route, label: 'View', variant: 'ghost', size: 'sm' })
          : null,
        renderButton({
          label: 'Dismiss',
          variant: 'ghost',
          size: 'sm',
          testId: `alert-dismiss-${alert.id}`,
          onClick: () => {
            dismissAlert(`home.${alert.id}`);
            if (opts.onDismiss) opts.onDismiss();
          },
        }),
      ),
    ));
  }
  return el('section', { className: 'alert-feed section-block', 'data-testid': 'alert-feed' },
    el('h3', { className: 'alert-feed__title' }, 'Alerts'),
    list,
  );
}
