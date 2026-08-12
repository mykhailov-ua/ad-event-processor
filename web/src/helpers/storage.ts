import {
  clampSidebarWidth,
  SIDEBAR_WIDTH_DEFAULT,
} from './sidebar_layout.js';
import { invalidateChartThemeCache } from '../charts/canvas_util.js';

const ALLOWED_KEYS = new Set([
  'ui.theme',
  'ui.theme-palette',
  'ui.sidebar.collapsed',
  'ui.sidebar.width',
  'ui.reports.range',
  'ui.dev_mode',
  'ui.ops.charts_layout',
  'ui.ops.charts_range',
  'nav.lastCustomerId',
  'nav.recentCustomerIds',
]);
const IDEM_PREFIX = 'idem.pending.';
const QUOTA_LIMIT = 4096;

function withStorageErrorHandler<T>(fn: () => T, fallback: T, _context?: string): T {
  try {
    return fn();
  } catch {
    return fallback;
  }
}

export type ColorTheme = 'dark' | 'light';

/**
 * Read the persisted color theme preference.
 */
export function getTheme(): ColorTheme {
  const v = _get('ui.theme');
  return v === 'light' || v === 'dark' ? v : 'dark';
}

/**
 * Persist the color theme and apply it to the document root.
 */
export function setTheme(theme: ColorTheme): void {
  if (theme !== 'dark' && theme !== 'light') return;
  _set('ui.theme', theme);
  document.documentElement.setAttribute('data-theme', theme);
  invalidateChartThemeCache();
}

/**
 * @deprecated Palette switching removed in operator UI (M11). Always returns `default`.
 */
export function getThemePalette(): 'default' {
  return 'default';
}

/**
 * @deprecated No-op — single canonical operator palette.
 */
export function setThemePalette(_palette: 'default' | 'neutral'): void {
  document.documentElement.removeAttribute('data-theme-palette');
}

/**
 * Read whether the sidebar is collapsed.
 */
export function getSidebarCollapsed(): boolean {
  return _get('ui.sidebar.collapsed') === 'true';
}

/**
 * Persist sidebar collapsed state.
 */
export function setSidebarCollapsed(collapsed: boolean): void {
  _set('ui.sidebar.collapsed', String(collapsed));
}

/**
 * Read the persisted sidebar width in pixels.
 */
export function getSidebarWidth(): number {
  const raw = _get('ui.sidebar.width');
  const n = raw ? Number.parseInt(raw, 10) : SIDEBAR_WIDTH_DEFAULT;
  if (!Number.isFinite(n)) return clampSidebarWidth(SIDEBAR_WIDTH_DEFAULT);
  return clampSidebarWidth(n);
}

/**
 * Persist sidebar width in pixels after clamping.
 */
export function setSidebarWidth(width: number): void {
  _set('ui.sidebar.width', String(clampSidebarWidth(width)));
}

export type OpsChartsLayout = 'grid' | 'stack';

/**
 * Read operations charts layout preference.
 */
export function getOpsChartsLayout(): OpsChartsLayout {
  return _get('ui.ops.charts_layout') === 'stack' ? 'stack' : 'grid';
}

/**
 * Persist operations charts layout preference.
 */
export function setOpsChartsLayout(layout: OpsChartsLayout): void {
  _set('ui.ops.charts_layout', layout === 'stack' ? 'stack' : 'grid');
}

export type OpsChartsRangeHours = 1 | 6 | 12 | 24;

/**
 * Read operations charts time range in hours.
 */
export function getOpsChartsRangeHours(): OpsChartsRangeHours {
  const raw = Number(_get('ui.ops.charts_range'));
  if (raw === 1 || raw === 6 || raw === 12 || raw === 24) return raw;
  return 24;
}

/**
 * Persist operations charts time range in hours.
 */
export function setOpsChartsRangeHours(hours: number): void {
  const h = Number(hours);
  if (h === 1 || h === 6 || h === 12 || h === 24) {
    _set('ui.ops.charts_range', String(h));
  }
}

/**
 * Read whether developer mode (raw technical strings) is enabled.
 */
export function getDevMode(): boolean {
  return _get('ui.dev_mode') === 'true';
}

/**
 * Persist developer mode preference.
 */
export function setDevMode(enabled: boolean): void {
  _set('ui.dev_mode', String(enabled));
}

export { getSidebarWidthBounds, SIDEBAR_COLLAPSED_WIDTH } from './sidebar_layout.js';

export type ReportRange = { from: string; to: string };

/**
 * Read the persisted report date range.
 */
export function getReportRange(): ReportRange | null {
  const v = _get('ui.reports.range');
  if (!v) return null;
  return withStorageErrorHandler(() => JSON.parse(v) as ReportRange, null, 'getReportRange');
}

/**
 * Persist a report date range.
 */
export function setReportRange(range: ReportRange): void {
  _set('ui.reports.range', JSON.stringify(range));
}

export type IdempotencyPending = {
  key: string;
  bodyHash: string;
  ts: number;
};

/**
 * Read a pending idempotency record for a scope.
 */
export function getIdempotencyPending(scope: string): IdempotencyPending | null {
  const v = _getRaw(IDEM_PREFIX + scope);
  if (!v) return null;
  return withStorageErrorHandler(() => {
    const parsed = JSON.parse(v) as IdempotencyPending;
    const ageMs = Date.now() - (parsed.ts ?? 0);
    if (ageMs > 24 * 60 * 60 * 1000) {
      removeIdempotencyPending(scope);
      return null;
    }
    return parsed;
  }, null, 'getIdempotencyPending');
}

/**
 * Persist a pending idempotency record for a scope.
 */
export function setIdempotencyPending(scope: string, data: IdempotencyPending): void {
  _setRaw(IDEM_PREFIX + scope, JSON.stringify(data));
}

/**
 * Remove a pending idempotency record for a scope.
 */
export function removeIdempotencyPending(scope: string): void {
  withStorageErrorHandler(() => {
    localStorage.removeItem(IDEM_PREFIX + scope);
  }, undefined, 'removeIdempotencyPending');
}

/**
 * Read the last visited customer id.
 */
export function getLastCustomerId(): string | null {
  return _get('nav.lastCustomerId');
}

/**
 * Persist the last visited customer id.
 */
export function setLastCustomerId(id: string): void {
  _set('nav.lastCustomerId', id);
}

const RECENT_CUSTOMERS_MAX = 8;

/**
 * Read recent customer ids from navigation storage.
 */
export function getRecentCustomerIds(): string[] {
  const v = _get('nav.recentCustomerIds');
  if (!v) return [];
  return withStorageErrorHandler(() => {
    const parsed = JSON.parse(v) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((x): x is string => typeof x === 'string' && x.length > 0)
      .slice(0, RECENT_CUSTOMERS_MAX);
  }, [], 'getRecentCustomerIds');
}

/**
 * Push a customer id to the front of recent navigation storage.
 */
export function pushRecentCustomerId(id: string): void {
  const trimmed = id.trim();
  if (!trimmed) return;
  const next = [trimmed, ...getRecentCustomerIds().filter((x) => x !== trimmed)]
    .slice(0, RECENT_CUSTOMERS_MAX);
  _set('nav.recentCustomerIds', JSON.stringify(next));
  setLastCustomerId(trimmed);
}

function _get(key: string): string | null {
  if (!ALLOWED_KEYS.has(key)) return null;
  return withStorageErrorHandler(() => localStorage.getItem(key), null, '_get');
}

function _set(key: string, value: string): void {
  if (!ALLOWED_KEYS.has(key)) return;
  withStorageErrorHandler(() => {
    localStorage.setItem(key, value);
    _enforceQuota();
  }, undefined, '_set');
}

function _getRaw(key: string): string | null {
  return withStorageErrorHandler(() => localStorage.getItem(key), null, '_getRaw');
}

function _setRaw(key: string, value: string): void {
  withStorageErrorHandler(() => {
    localStorage.setItem(key, value);
    _enforceQuota();
  }, undefined, '_setRaw');
}

/**
 * Remove all pending idempotency keys from localStorage.
 */
export function clearIdempotencyPendingAll(): void {
  withStorageErrorHandler(() => {
    const keys: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (k?.startsWith(IDEM_PREFIX)) keys.push(k);
    }
    for (const k of keys) localStorage.removeItem(k);
  }, undefined, 'clearIdempotencyPendingAll');
}

/**
 * Evict idempotency pending keys when localStorage exceeds the quota budget.
 */
function _enforceQuota(): void {
  withStorageErrorHandler(() => {
    let total = 0;
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (!k) continue;
      total += k.length + (localStorage.getItem(k) ?? '').length;
    }
    if (total > QUOTA_LIMIT) {
      const idemKeys: string[] = [];
      for (let i = 0; i < localStorage.length; i++) {
        const k = localStorage.key(i);
        if (k?.startsWith(IDEM_PREFIX)) idemKeys.push(k);
      }
      for (const k of idemKeys) localStorage.removeItem(k);
    }
  }, undefined, '_enforceQuota');
}
