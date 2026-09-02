const STORAGE_KEY = 'adminDevMode';

// Dev-only: show raw error text in ErrorBlock. Boot via ?admin_dev=1|0; persisted in localStorage.
let active = false;

function readStored(): boolean {
  if (typeof window === 'undefined') {
    return false;
  }
  try {
    return window.localStorage.getItem(STORAGE_KEY) === '1';
  } catch {
    return false;
  }
}

function writeStored(enabled: boolean): void {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    if (enabled) {
      window.localStorage.setItem(STORAGE_KEY, '1');
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // Private browsing or blocked storage.
  }
}

/** Call once before the React tree mounts (main.tsx). */
export function initAdminDevModeFromUrl(): void {
  if (typeof window === 'undefined') {
    return;
  }
  const params = new URLSearchParams(window.location.search);
  const flag = params.get('admin_dev');
  if (flag === '1') {
    active = true;
    writeStored(true);
    params.delete('admin_dev');
    const nextQuery = params.toString();
    const nextUrl = `${window.location.pathname}${nextQuery ? `?${nextQuery}` : ''}${window.location.hash}`;
    window.history.replaceState(null, '', nextUrl);
    return;
  }
  if (flag === '0') {
    active = false;
    writeStored(false);
    params.delete('admin_dev');
    const nextQuery = params.toString();
    const nextUrl = `${window.location.pathname}${nextQuery ? `?${nextQuery}` : ''}${window.location.hash}`;
    window.history.replaceState(null, '', nextUrl);
    return;
  }
  active = readStored();
}

export function isAdminDevMode(): boolean {
  return active;
}

export function enableAdminDevMode(): void {
  active = true;
  writeStored(true);
}

export function disableAdminDevMode(): void {
  active = false;
  writeStored(false);
}
