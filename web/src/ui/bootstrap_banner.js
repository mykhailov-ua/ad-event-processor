import { el } from '../lib/dom.js';

/**
 * @param {{ bootstrapComplete: boolean }} opts
 */
export function renderBootstrapBanner(opts) {
  if (opts.bootstrapComplete) return el('div');
  return el('div', { className: 'stub-banner mb-4' },
    el('span', {}, 'Platform bootstrap is not complete. '),
    el('a', { href: '/bootstrap', style: { color: 'var(--accent)' } }, 'Run bootstrap'),
  );
}
