import { el } from '../lib/dom.js';
import * as storage from '../helpers/storage.js';
import { shortCustomerId } from '../helpers/customer_context.js';

/**
 * Render recent customer quick links from navigation storage.
 *
 * @param {{ tenant?: boolean }} opts
 * @returns {HTMLElement|null}
 */
export function renderRecentCustomers(opts = {}) {
  if (opts.tenant) return null;
  const ids = storage.getRecentCustomerIds();
  if (ids.length === 0) return null;

  return el('div', { className: 'recent-bar' },
    el('span', { className: 'recent-bar__label' }, 'Recent'),
    ids.map((id) =>
      el('a', {
        href: `/customers/${id}`,
        className: 'recent-chip',
        title: id,
      }, shortCustomerId(id)),
    ),
  );
}
