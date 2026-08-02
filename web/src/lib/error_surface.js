import { el } from './dom.js';

let mounted = false;

/**
 * @param {HTMLElement} root
 */
export function installErrorSurface(root) {
  if (mounted) return;
  mounted = true;

  /** @type {HTMLElement|null} */
  let banner = null;

  function show(message) {
    if (!banner) {
      banner = el('div', {
        className: 'stub-banner',
        style: { borderColor: 'var(--error)', margin: '8px 16px' },
      });
      root.prepend(banner);
    }
    const reloadBtn = el('button', {
      type: 'button',
      className: 'btn btn--primary btn--sm',
      style: { marginLeft: 12 },
      onClick: () => window.location.reload(),
    }, 'Reload page');
    banner.replaceChildren(
      el('span', {}, message),
      reloadBtn,
    );
  }

  window.addEventListener('error', (e) => {
    show(e.message || 'Unexpected error');
  });
  window.addEventListener('unhandledrejection', (e) => {
    const msg = e.reason?.message ?? String(e.reason ?? 'Unhandled rejection');
    show(msg);
  });
}
