import * as storage from './storage.js';

export function devModeEnabled(): boolean {
  if (typeof window === 'undefined') return false;
  const params = new URLSearchParams(window.location.search);
  if (params.has('dev')) {
    return params.get('dev') !== '0' && params.get('dev') !== 'false';
  }
  return storage.getDevMode();
}

export function setDevMode(enabled: boolean): void {
  storage.setDevMode(enabled);
  document.documentElement.toggleAttribute('data-dev-mode', enabled);
}

export function syncDevModeAttribute(): void {
  document.documentElement.toggleAttribute('data-dev-mode', devModeEnabled());
}
