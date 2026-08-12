import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

export type BreadcrumbItem = {
  label: string;
  href?: string;
};

/**
 * Render a breadcrumb trail for the page header.
 */
export function renderBreadcrumbs(items: BreadcrumbItem[]): HTMLElement | null {
  if (!items.length) return null;
  return el('nav', { className: 'page-header__breadcrumbs', 'aria-label': 'Breadcrumb' },
    items.flatMap((item, i) => {
      const nodes: (HTMLElement | SVGElement | null)[] = [];
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
