import { renderAlertBanner } from './alert_banner.js';
import { renderButton } from './button.js';
import * as storage from '../helpers/storage.js';

export type IdempotencyBannerOpts = {
  onCleared?: (() => void) | null;
};

/**
 * Scan localStorage for pending idempotency scopes and render recovery banner.
 */
export function renderIdempotencyRecoveryBanner(opts: IdempotencyBannerOpts = {}): HTMLElement | null {
  const pending = listPendingIdempotencyScopes();
  if (pending.length === 0) return null;
  const banner = renderAlertBanner({
    variant: 'warning',
    message: `Unfinished write operations (${pending.length}). Retry the action or clear pending keys.`,
    dismissKey: 'idem.recovery',
  });
  if (!banner) return null;
  banner.appendChild(renderButton({
    label: 'Clear pending',
    variant: 'secondary',
    size: 'sm',
    onClick: () => {
      storage.clearIdempotencyPendingAll();
      opts.onCleared?.();
      window.location.reload();
    },
  }));
  return banner;
}

/**
 * List idempotency pending scope suffixes from localStorage.
 */
function listPendingIdempotencyScopes(): string[] {
  const out: string[] = [];
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
