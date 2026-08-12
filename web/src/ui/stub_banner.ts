import { el } from '../lib/dom.js';

export type StubBannerOpts = {
  message?: string;
  linkTo?: string;
  linkLabel?: string;
};

/**
 * Render a 501 stub endpoint banner with an optional link.
 */
export function renderStubBanner(opts: StubBannerOpts): HTMLElement {
  return el('div', { className: 'stub-banner' },
    el('p', { className: 'stub-banner__message' }, opts.message ?? 'Endpoint not implemented (501).'),
    opts.linkTo
      ? el('a', {
        href: opts.linkTo,
        className: 'stub-banner__link',
      }, opts.linkLabel ?? 'Open placements report')
      : null,
  );
}
