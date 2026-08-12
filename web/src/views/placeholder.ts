import { el, replaceChildren } from '../lib/dom.js';
import type { ViewHandle } from '../lib/router_types.js';

const TITLE_MAP: Record<string, string> = {
  '/rtb/deals': 'RTB: PMP deals',
  '/rtb/integration': 'RTB: integration',
  '/ops/blacklist': 'Blacklist',
  '/audit': 'Audit',
};

/**
 * Mount a placeholder page for routes under development.
 */
export function mount(container: HTMLElement): ViewHandle {
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
