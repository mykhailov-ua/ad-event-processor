import { el, replaceChildren } from '../lib/dom.js';
import type { ViewHandle } from '../lib/router_types.js';
import { renderButtonLink } from '../ui/button.js';

/**
 * Mount the 404 not-found page.
 */
export function mount(container: HTMLElement): ViewHandle {
  replaceChildren(container,
    el('div', { className: 'error-page' },
      el('div', { className: 'error-page__code' }, '404'),
      el('div', { className: 'error-page__title' }, 'Page not found'),
      el('div', { className: 'error-page__desc text-muted mb-4' },
        'The requested route does not exist or is not implemented yet.',
      ),
      renderButtonLink({ href: '/', label: 'Home', variant: 'secondary', size: 'sm' }),
    ),
  );
  return {};
}
