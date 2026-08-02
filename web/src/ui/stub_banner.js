import { el } from '../lib/dom.js';

/**
 * @param {{ message?: string, linkTo?: string, linkLabel?: string }} opts
 */
export function renderStubBanner(opts) {
  const children = [
    el('span', {}, opts.message ?? 'Endpoint not implemented (501).'),
  ];
  if (opts.linkTo) {
    children.push(
      el('a', {
        href: opts.linkTo,
        style: { marginLeft: 12, color: 'var(--accent)' },
      }, opts.linkLabel ?? 'Open placements report'),
    );
  }
  return el('div', { className: 'stub-banner' }, children);
}
