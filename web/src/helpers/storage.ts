import { clampSidebarWidth, SIDEBAR_WIDTH_DEFAULT } from './sidebar_layout.js';

const ALLOWED_KEYS = new Set([
  'ui.theme',
  'ui.sidebar.collapsed',
  'ui.sidebar.width',
  'ui.dev_mode',
  'nav.lastCustomerId',
  'nav.recentCustomerIds',
]);
const IDEM_PREFIX = 'idem.pending.';

export type ColorTheme = 'dark' | 'light';

export function getTheme(): ColorTheme {
  const v = localStorageGet('ui.theme');
  return v === 'light' || v === 'dark' ? v : 'dark';
}

export function setTheme(theme: ColorTheme): void {
  if (theme !== 'dark' && theme !== 'light') return;
  localStorageSet('ui.theme', theme);
  document.documentElement.setAttribute('data-theme', theme);
}

export function getSidebarCollapsed(): boolean {
  return localStorageGet('ui.sidebar.collapsed') === 'true';
}

export function setSidebarCollapsed(collapsed: boolean): void {
  localStorageSet('ui.sidebar.collapsed', String(collapsed));
}

export function getSidebarWidth(): number {
  const raw = localStorageGet('ui.sidebar.width');
  const n = raw ? Number.parseInt(raw, 10) : SIDEBAR_WIDTH_DEFAULT;
  if (!Number.isFinite(n)) return clampSidebarWidth(SIDEBAR_WIDTH_DEFAULT);
  return clampSidebarWidth(n);
}

export function setSidebarWidth(width: number): void {
  localStorageSet('ui.sidebar.width', String(clampSidebarWidth(width)));
}

export function getDevMode(): boolean {
  return localStorageGet('ui.dev_mode') === 'true';
}

export function setDevMode(enabled: boolean): void {
  localStorageSet('ui.dev_mode', String(enabled));
}

export { getSidebarWidthBounds, SIDEBAR_COLLAPSED_WIDTH } from './sidebar_layout.js';

export type IdempotencyPending = {
  key: string;
  bodyHash: string;
  ts: number;
};

export function getIdempotencyPending(scope: string): IdempotencyPending | null {
  const v = localStorageGetRaw(IDEM_PREFIX + scope);
  if (!v) return null;
  try {
    const parsed = JSON.parse(v) as IdempotencyPending;
    if (Date.now() - (parsed.ts ?? 0) > 24 * 60 * 60 * 1000) {
      removeIdempotencyPending(scope);
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function setIdempotencyPending(scope: string, data: IdempotencyPending): void {
  localStorageSetRaw(IDEM_PREFIX + scope, JSON.stringify(data));
}

export function removeIdempotencyPending(scope: string): void {
  try {
    localStorage.removeItem(IDEM_PREFIX + scope);
  } catch {
    /* ignore */
  }
}

export function getLastCustomerId(): string | null {
  return localStorageGet('nav.lastCustomerId');
}

export function setLastCustomerId(id: string): void {
  localStorageSet('nav.lastCustomerId', id);
}

const RECENT_CUSTOMERS_MAX = 8;

export function getRecentCustomerIds(): string[] {
  const v = localStorageGet('nav.recentCustomerIds');
  if (!v) return [];
  try {
    const parsed = JSON.parse(v) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter((x): x is string => typeof x === 'string' && x.length > 0)
      .slice(0, RECENT_CUSTOMERS_MAX);
  } catch {
    return [];
  }
}

export function pushRecentCustomerId(id: string): void {
  const trimmed = id.trim();
  if (!trimmed) return;
  const next = [trimmed, ...getRecentCustomerIds().filter((x) => x !== trimmed)].slice(
    0,
    RECENT_CUSTOMERS_MAX
  );
  localStorageSet('nav.recentCustomerIds', JSON.stringify(next));
  setLastCustomerId(trimmed);
}

export function clearIdempotencyPendingAll(): void {
  try {
    const keys: string[] = [];
    for (let i = 0; i < localStorage.length; i += 1) {
      const k = localStorage.key(i);
      if (k?.startsWith(IDEM_PREFIX)) keys.push(k);
    }
    for (const k of keys) localStorage.removeItem(k);
  } catch {
    /* ignore */
  }
}

function localStorageGet(key: string): string | null {
  if (!ALLOWED_KEYS.has(key)) return null;
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function localStorageSet(key: string, value: string): void {
  if (!ALLOWED_KEYS.has(key)) return;
  try {
    localStorage.setItem(key, value);
  } catch {
    /* ignore */
  }
}

function localStorageGetRaw(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function localStorageSetRaw(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    /* ignore */
  }
}
