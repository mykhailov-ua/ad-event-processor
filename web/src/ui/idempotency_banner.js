import { el } from '../lib/dom.js';
import { renderAlertBanner } from './alert_banner.js';
import * as storage from '../helpers/storage.js';

/**
 * Scan localStorage for pending idempotency scopes and render recovery banner.
 *
 * @param {{ onCleared?: () => void }} [opts]
 * @returns {HTMLElement|null}
 */
export function renderIdempotencyRecoveryBanner(opts = {}) {
  const pending = listPendingIdempotencyScopes();
  if (pending.length === 0) return null;
  const banner = renderAlertBanner({
    variant: 'warning',
    message: `Unfinished write operations (${pending.length}). Retry the action or clear pending keys.`,
    dismissKey: 'idem.recovery',
  });
  if (!banner) return null;
  banner.appendChild(el('button', {
    type: 'button',
    className: 'btn btn--secondary btn--sm',
    style: { marginLeft: 8 },
    onClick: () => {
      storage.clearIdempotencyPendingAll();
      opts.onCleared?.();
      window.location.reload();
    },
  }, 'Clear pending'));
  return banner;
}

/**
 * List idempotency pending scope suffixes from localStorage.
 *
 * @returns {string[]}
 */
function listPendingIdempotencyScopes() {
  const out = [];
  try {
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith('idem.pending.')) {
        out.push(key.slice('idem.pending.'.length));
      }
    }
  } catch {
    return [];
  }
  return out;
}
