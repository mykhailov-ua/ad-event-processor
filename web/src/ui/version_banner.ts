import { el } from '../lib/dom.js';
import { BUILD_LABEL } from '../lib/build_label.js';
import { renderButton } from './button.js';

const STORAGE_KEY = 'adminServerVersion';

export type VersionBannerOpts = {
  serverVersion: string | null;
};

/**
 * Render a reload prompt when the server version changes mid-session.
 */
export function renderVersionBanner(opts: VersionBannerOpts): HTMLElement {
  const serverVersion = opts.serverVersion?.trim() ?? '';
  if (!serverVersion) return el('div');

  let prev: string | null = null;
  try {
    prev = sessionStorage.getItem(STORAGE_KEY);
  } catch {
    prev = null;
  }

  try {
    sessionStorage.setItem(STORAGE_KEY, serverVersion);
  } catch {
    // ignore quota / private mode
  }

  if (!prev || prev === serverVersion) return el('div');

  const buildHint = BUILD_LABEL ? ` UI bundle ${BUILD_LABEL}.` : '';
  return el('div', {
    className: 'stub-banner mb-4 cluster cluster--sm items-center',
    style: { borderColor: 'var(--warning)' },
  },
    el('span', {}, `Server updated (${prev} → ${serverVersion}).${buildHint} Reload to pick up changes.`),
    renderButton({
      label: 'Reload',
      variant: 'secondary',
      size: 'sm',
      onClick: () => window.location.reload(),
    }),
  );
}
