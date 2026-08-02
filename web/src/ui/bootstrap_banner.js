import { el } from '../lib/dom.js';

/**
 * Render an incomplete-bootstrap warning banner with a link to bootstrap.
 *
 * @param {{ bootstrapComplete: boolean }} opts
 * @returns {HTMLElement}
 */
export function renderBootstrapBanner(opts) {
  if (opts.bootstrapComplete) return el('div');
  return el('div', { className: 'stub-banner mb-4' },
    el('span', {}, 'Platform bootstrap is not complete. '),
    el('a', { href: '/bootstrap', style: { color: 'var(--accent)' } }, 'Run bootstrap'),
  );
}
