import { el } from '../lib/dom.js';
import { renderStatusBadge } from './status_badge.js';

/** Render the campaign status legend for list pages. */
export function renderCampaignStatusLegend(): HTMLElement {
  return el('div', { className: 'status-legend', 'aria-label': 'Status legend' },
    el('span', { className: 'status-legend__label' }, 'Status'),
    renderStatusBadge('ACTIVE'),
    renderStatusBadge('PAUSED'),
    renderStatusBadge('ARCHIVED'),
  );
}
