import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

/**
 * @typedef {{ label: string, href?: string }} BreadcrumbItem
 */

/**
 * Render a breadcrumb trail for the page header.
 *
 * @param {BreadcrumbItem[]} items
 * @returns {HTMLElement|null}
 */
export function renderBreadcrumbs(items) {
  if (!items.length) return null;
  return el('nav', { className: 'page-header__breadcrumbs', 'aria-label': 'Breadcrumb' },
    items.flatMap((item, i) => {
      const nodes = [];
      if (i > 0) {
        nodes.push(renderIcon('chevron-right', { size: 12, className: 'page-header__crumb-sep' }));
      }
      if (item.href) {
        nodes.push(el('a', { href: item.href, className: 'page-header__crumb-link' }, item.label));
      } else {
        nodes.push(el('span', { className: 'page-header__crumb-current' }, item.label));
      }
      return nodes;
    }),
  );
}
