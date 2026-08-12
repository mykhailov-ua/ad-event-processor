import { el } from '../lib/dom.js';

export type BootstrapBannerOpts = {
  bootstrapComplete: boolean;
};

/**
 * Render an incomplete-bootstrap warning banner with a link to bootstrap.
 */
export function renderBootstrapBanner(opts: BootstrapBannerOpts): HTMLElement {
  if (opts.bootstrapComplete) return el('div');
  return el('div', { className: 'stub-banner mb-4' },
    el('span', {}, 'Platform bootstrap is not complete. '),
    el('a', { href: '/bootstrap', style: { color: 'var(--accent)' } }, 'Run bootstrap'),
  );
}
