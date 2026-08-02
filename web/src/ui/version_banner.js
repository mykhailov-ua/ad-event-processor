import { el } from '../lib/dom.js';
import { BUILD_LABEL } from '../lib/build_label.js';

const STORAGE_KEY = 'adminServerVersion';

/**
 * @param {{ serverVersion: string|null }} opts
 */
export function renderVersionBanner(opts) {
  const serverVersion = opts.serverVersion?.trim() ?? '';
  if (!serverVersion) return el('div');

  let prev = null;
  try {
    prev = sessionStorage.getItem(STORAGE_KEY);
  } catch {
    prev = null;
  }

  try {
    sessionStorage.setItem(STORAGE_KEY, serverVersion);
  } catch {
    /* ignore quota */
  }

  if (!prev || prev === serverVersion) return el('div');

  const buildHint = BUILD_LABEL ? ` UI bundle ${BUILD_LABEL}.` : '';
  return el('div', {
    className: 'stub-banner mb-4',
    style: { borderColor: 'var(--warning)' },
  },
    `Server updated (${prev} → ${serverVersion}).${buildHint} Reload to pick up changes.`,
    el('button', {
      type: 'button',
      className: 'btn btn--secondary btn--sm',
      style: { marginLeft: 12 },
      onClick: () => window.location.reload(),
    }, 'Reload'),
  );
}
