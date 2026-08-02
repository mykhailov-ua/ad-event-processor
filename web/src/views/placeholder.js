import { el, replaceChildren } from '../lib/dom.js';

/** @type {Record<string, string>} */
const TITLE_MAP = {
  '/rtb/deals': 'RTB: deals',
  '/ops/blacklist': 'Blacklist',
  '/audit': 'Audit',
};

/**
 * Mount a placeholder page for routes under development.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container) {
  const path = window.location.pathname.replace(/\/$/, '') || '/';
  const title = TITLE_MAP[path] ?? 'Section';

  replaceChildren(container,
    el('div', { className: 'page-header' },
      el('div', { className: 'page-header__row' },
        el('h1', { className: 'page-header__title' }, title),
      ),
    ),
    el('div', { className: 'stub-banner' },
      'This section is under development. API and UI will ship in a future release.',
    ),
  );
  return {};
}
