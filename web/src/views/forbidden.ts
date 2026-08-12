import { el, replaceChildren } from '../lib/dom.js';
import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import { buyerEmptyCopy } from '../models/empty_state.js';

/**
 * Mount a 403 forbidden page for routes the user cannot access.
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
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
