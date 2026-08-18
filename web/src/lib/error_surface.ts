import { el } from './dom.js';

let mounted = false;

export function installErrorSurface(root: HTMLElement): void {
  if (mounted) return;
  mounted = true;

  let banner: HTMLElement | null = null;

  function show(message: string): void {
    if (!banner) {
      banner = el('div', {
        className: 'stub-banner cluster cluster--sm items-center',
        style: { borderColor: 'var(--error)', margin: '8px 16px' },
      });
      root.prepend(banner);
    }
    const reloadBtn = el(
      'button',
      {
        type: 'button',
        className: 'btn btn--primary btn--sm',
        onClick: () => window.location.reload(),
      },
      'Reload page'
    );
    banner.replaceChildren(el('span', {}, message), reloadBtn);
  }

  window.addEventListener('error', (e) => {
    show(e.message || 'Unexpected error');
  });
  window.addEventListener('unhandledrejection', (e) => {
    const reason = e.reason;
    const msg = reason instanceof Error ? reason.message : String(reason ?? 'Unhandled rejection');
    show(msg);
  });
}
