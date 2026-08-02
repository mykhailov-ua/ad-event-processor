import {
  clampSidebarWidth,
  SIDEBAR_COLLAPSED_WIDTH,
  SIDEBAR_WIDTH_DEFAULT,
} from './sidebar_layout.js';

const ALLOWED_KEYS = new Set([
  'ui.theme',
  'ui.theme-palette',
  'ui.sidebar.collapsed',
  'ui.sidebar.width',
  'ui.reports.range',
  'nav.lastCustomerId',
  'nav.recentCustomerIds',
]);
const IDEM_PREFIX = 'idem.pending.';
const QUOTA_LIMIT = 4096;

/**
 * Read the persisted color theme preference.
 *
 * @returns {'dark'|'light'}
 */
export function getTheme() {
  return _get('ui.theme') ?? 'dark';
}

/**
 * Persist the color theme and apply it to the document root.
 *
 * @param {'dark'|'light'} theme
 * @returns {void}
 */
export function setTheme(theme) {
  if (theme !== 'dark' && theme !== 'light') return;
  _set('ui.theme', theme);
  document.documentElement.setAttribute('data-theme', theme);
}

/**
 * @deprecated Palette switching removed in operator UI (M11). Always returns `default`.
 * @returns {'default'}
 */
export function getThemePalette() {
  return 'default';
}

/**
 * @deprecated No-op — single canonical operator palette.
 * @param {'default'|'neutral'} _palette
 * @returns {void}
 */
export function setThemePalette(_palette) {
  document.documentElement.removeAttribute('data-theme-palette');
}

/**
 * Read whether the sidebar is collapsed.
 *
 * @returns {boolean}
 */
export function getSidebarCollapsed() {
  return _get('ui.sidebar.collapsed') === 'true';
}

/**
 * Persist sidebar collapsed state.
 *
 * @param {boolean} collapsed
 * @returns {void}
 */
export function setSidebarCollapsed(collapsed) {
  _set('ui.sidebar.collapsed', String(collapsed));
}

/**
 * Read the persisted sidebar width in pixels.
 *
 * @returns {number}
 */
export function getSidebarWidth() {
  const raw = _get('ui.sidebar.width');
  const n = raw ? Number.parseInt(raw, 10) : SIDEBAR_WIDTH_DEFAULT;
  if (!Number.isFinite(n)) return clampSidebarWidth(SIDEBAR_WIDTH_DEFAULT);
  return clampSidebarWidth(n);
}

/**
 * Persist sidebar width in pixels after clamping.
 *
 * @param {number} width
 * @returns {void}
 */
export function setSidebarWidth(width) {
  _set('ui.sidebar.width', String(clampSidebarWidth(width)));
}

export { getSidebarWidthBounds, SIDEBAR_COLLAPSED_WIDTH } from './sidebar_layout.js';

/**
 * Read the persisted report date range.
 *
 * @returns {{from: string, to: string}|null}
 */
export function getReportRange() {
  const v = _get('ui.reports.range');
  if (!v) return null;
  try { return JSON.parse(v); } catch { return null; }
}

/**
 * Persist a report date range.
 *
 * @param {{from: string, to: string}} range
 * @returns {void}
 */
export function setReportRange(range) {
  _set('ui.reports.range', JSON.stringify(range));
}

/**
 * Read a pending idempotency record for a scope.
 *
 * @param {string} scope
 * @returns {{key: string, bodyHash: string, ts: number}|null}
 */
export function getIdempotencyPending(scope) {
  const v = _getRaw(IDEM_PREFIX + scope);
  if (!v) return null;
  try {
    const parsed = JSON.parse(v);
    const ageMs = Date.now() - (parsed.ts ?? 0);
    if (ageMs > 24 * 60 * 60 * 1000) { removeIdempotencyPending(scope); return null; }
    return parsed;
  } catch { return null; }
}

/**
 * Persist a pending idempotency record for a scope.
 *
 * @param {string} scope
 * @param {{key: string, bodyHash: string, ts: number}} data
 * @returns {void}
 */
export function setIdempotencyPending(scope, data) {
  _setRaw(IDEM_PREFIX + scope, JSON.stringify(data));
}

/**
 * Remove a pending idempotency record for a scope.
 *
 * @param {string} scope
 * @returns {void}
 */
export function removeIdempotencyPending(scope) {
  try { localStorage.removeItem(IDEM_PREFIX + scope); } catch {}
}

/**
 * Read the last visited customer id.
 *
 * @returns {string|null}
 */
export function getLastCustomerId() {
  return _get('nav.lastCustomerId');
}

/**
 * Persist the last visited customer id.
 *
 * @param {string} id
 * @returns {void}
 */
export function setLastCustomerId(id) {
  _set('nav.lastCustomerId', id);
}

const RECENT_CUSTOMERS_MAX = 8;

/**
 * Read recent customer ids from navigation storage.
 *
 * @returns {string[]}
 */
export function getRecentCustomerIds() {
  const v = _get('nav.recentCustomerIds');
  if (!v) return [];
  try {
    const parsed = JSON.parse(v);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((x) => typeof x === 'string' && x.length > 0).slice(0, RECENT_CUSTOMERS_MAX);
  } catch {
    return [];
  }
}

/**
 * Push a customer id to the front of recent navigation storage.
 *
 * @param {string} id
 * @returns {void}
 */
export function pushRecentCustomerId(id) {
  const trimmed = id.trim();
  if (!trimmed) return;
  const next = [trimmed, ...getRecentCustomerIds().filter((x) => x !== trimmed)]
    .slice(0, RECENT_CUSTOMERS_MAX);
  _set('nav.recentCustomerIds', JSON.stringify(next));
  setLastCustomerId(trimmed);
}

/**
 * Read a whitelisted localStorage key.
 *
 * @param {string} key
 * @returns {string|null}
 */
function _get(key) {
  if (!ALLOWED_KEYS.has(key)) return null;
  try { return localStorage.getItem(key); } catch { return null; }
}

/**
 * Write a whitelisted localStorage key and enforce quota.
 *
 * @param {string} key
 * @param {string} value
 * @returns {void}
 */
function _set(key, value) {
  if (!ALLOWED_KEYS.has(key)) return;
  try {
    localStorage.setItem(key, value);
    _enforceQuota();
  } catch {}
}

/**
 * Read a localStorage key without the whitelist guard.
 *
 * @param {string} key
 * @returns {string|null}
 */
function _getRaw(key) {
  try { return localStorage.getItem(key); } catch { return null; }
}

/**
 * Write a localStorage key without the whitelist guard and enforce quota.
 *
 * @param {string} key
 * @param {string} value
 * @returns {void}
 */
function _setRaw(key, value) {
  try {
    localStorage.setItem(key, value);
    _enforceQuota();
  } catch {}
}

/**
 * Remove all pending idempotency keys from localStorage.
 *
 * @returns {void}
 */
export function clearIdempotencyPendingAll() {
  try {
    const keys = [];
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (k?.startsWith(IDEM_PREFIX)) keys.push(k);
    }
    for (const k of keys) localStorage.removeItem(k);
  } catch {}
}

/**
 * Evict idempotency pending keys when localStorage exceeds the quota budget.
 *
 * @returns {void}
 */
function _enforceQuota() {
  let total = 0;
  try {
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      total += (k?.length ?? 0) + (localStorage.getItem(k) ?? '').length;
    }
    if (total > QUOTA_LIMIT) {
      const idemKeys = [];
      for (let i = 0; i < localStorage.length; i++) {
        const k = localStorage.key(i);
        if (k?.startsWith(IDEM_PREFIX)) idemKeys.push(k);
      }
      for (const k of idemKeys) localStorage.removeItem(k);
    }
  } catch {}
}
