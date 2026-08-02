import { el, replaceChildren } from '../lib/dom.js';

/**
 * Mount the 404 not-found page.
 *
 * @param {HTMLElement} container
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container) {
  replaceChildren(container,
    el('div', { className: 'error-page' },
      el('div', { className: 'error-page__code' }, '404'),
      el('div', { className: 'error-page__title' }, 'Page not found'),
      el('div', { className: 'error-page__desc text-muted mb-4' },
        'The requested route does not exist or is not implemented yet.',
      ),
      el('a', { href: '/', className: 'btn btn--secondary btn--sm' }, 'Home'),
    ),
  );
  return {};
}
