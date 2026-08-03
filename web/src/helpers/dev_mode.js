import * as storage from './storage.js';

/**
 * Whether developer mode is active (raw technical strings visible).
 *
 * @returns {boolean}
 */
export function devModeEnabled() {
  if (typeof window === 'undefined') return false;
  const params = new URLSearchParams(window.location.search);
  if (params.has('dev')) {
    return params.get('dev') !== '0' && params.get('dev') !== 'false';
  }
  return storage.getDevMode();
}

/**
 * Persist developer mode and sync the document attribute.
 *
 * @param {boolean} enabled
 */
export function setDevMode(enabled) {
  storage.setDevMode(enabled);
  document.documentElement.toggleAttribute('data-dev-mode', enabled);
}

/**
 * Apply dev mode from storage / URL on boot.
 */
export function syncDevModeAttribute() {
  document.documentElement.toggleAttribute('data-dev-mode', devModeEnabled());
}
