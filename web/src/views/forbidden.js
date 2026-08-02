import { el, replaceChildren } from '../lib/dom.js';
import { buyerEmptyCopy } from '../models/empty_state.js';

/**
 * Mount a 403 forbidden page for routes the user cannot access.
 *
 * @param {HTMLElement} container
 * @param {{ navigate: (path: string) => void }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  const copy = buyerEmptyCopy('forbidden');
  replaceChildren(container,
    el('main', null,
      el('h1', null, copy.title),
      el('p', null, copy.description),
      el('button', {
        type: 'button',
        onClick: () => ctx.navigate(copy.actionHref ?? '/campaigns'),
      }, copy.actionLabel ?? 'Continue'),
    ),
  );
  return {};
}
