import * as storage from './storage.js';

/**
 * Whether developer mode is active (raw technical strings visible).
 */
export function devModeEnabled(): boolean {
  if (typeof window === 'undefined') return false;
  const params = new URLSearchParams(window.location.search);
  if (params.has('dev')) {
    return params.get('dev') !== '0' && params.get('dev') !== 'false';
  }
  return storage.getDevMode();
}

/**
 * Persist developer mode and sync the document attribute.
 */
export function setDevMode(enabled: boolean): void {
  storage.setDevMode(enabled);
  document.documentElement.toggleAttribute('data-dev-mode', enabled);
}

/**
 * Apply dev mode from storage / URL on boot.
 */
export function syncDevModeAttribute(): void {
  document.documentElement.toggleAttribute('data-dev-mode', devModeEnabled());
}
