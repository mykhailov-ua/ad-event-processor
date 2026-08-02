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
 * @returns {'dark'|'light'}
 */
export function getTheme() {
  return _get('ui.theme') ?? 'dark';
}

/**
 * @param {'dark'|'light'} theme
 */
export function setTheme(theme) {
  if (theme !== 'dark' && theme !== 'light') return;
  _set('ui.theme', theme);
  document.documentElement.setAttribute('data-theme', theme);
}

/**
 * @returns {'default'|'neutral'}
 */
export function getThemePalette() {
  return _get('ui.theme-palette') ?? 'neutral';
}

/**
 * @param {'default'|'neutral'} palette
 */
export function setThemePalette(palette) {
  if (palette !== 'default' && palette !== 'neutral') return;
  _set('ui.theme-palette', palette);
  document.documentElement.setAttribute('data-theme-palette', palette);
}

/**
 * @returns {boolean}
 */
export function getSidebarCollapsed() {
  return _get('ui.sidebar.collapsed') === 'true';
}

/**
 * @param {boolean} collapsed
 */
export function setSidebarCollapsed(collapsed) {
  _set('ui.sidebar.collapsed', String(collapsed));
}

/**
 * @returns {number}
 */
export function getSidebarWidth() {
  const raw = _get('ui.sidebar.width');
  const n = raw ? Number.parseInt(raw, 10) : SIDEBAR_WIDTH_DEFAULT;
  if (!Number.isFinite(n)) return clampSidebarWidth(SIDEBAR_WIDTH_DEFAULT);
  return clampSidebarWidth(n);
}

/**
 * @param {number} width
 */
export function setSidebarWidth(width) {
  _set('ui.sidebar.width', String(clampSidebarWidth(width)));
}

export { getSidebarWidthBounds, SIDEBAR_COLLAPSED_WIDTH } from './sidebar_layout.js';

/**
 * @returns {{from: string, to: string}|null}
 */
export function getReportRange() {
  const v = _get('ui.reports.range');
  if (!v) return null;
  try { return JSON.parse(v); } catch { return null; }
}

/**
 * @param {{from: string, to: string}} range
 */
export function setReportRange(range) {
  _set('ui.reports.range', JSON.stringify(range));
}

/**
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
 * @param {string} scope
 * @param {{key: string, bodyHash: string, ts: number}} data
 */
export function setIdempotencyPending(scope, data) {
  _setRaw(IDEM_PREFIX + scope, JSON.stringify(data));
}

/**
 * @param {string} scope
 */
export function removeIdempotencyPending(scope) {
  try { localStorage.removeItem(IDEM_PREFIX + scope); } catch {}
}

/**
 * @returns {string|null}
 */
export function getLastCustomerId() {
  return _get('nav.lastCustomerId');
}

/**
 * @param {string} id
 */
export function setLastCustomerId(id) {
  _set('nav.lastCustomerId', id);
}

const RECENT_CUSTOMERS_MAX = 8;

/**
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
 * @param {string} id
 */
export function pushRecentCustomerId(id) {
  const trimmed = id.trim();
  if (!trimmed) return;
  const next = [trimmed, ...getRecentCustomerIds().filter((x) => x !== trimmed)]
    .slice(0, RECENT_CUSTOMERS_MAX);
  _set('nav.recentCustomerIds', JSON.stringify(next));
  setLastCustomerId(trimmed);
}

function _get(key) {
  if (!ALLOWED_KEYS.has(key)) return null;
  try { return localStorage.getItem(key); } catch { return null; }
}

function _set(key, value) {
  if (!ALLOWED_KEYS.has(key)) return;
  try {
    localStorage.setItem(key, value);
    _enforceQuota();
  } catch {}
}

function _getRaw(key) {
  try { return localStorage.getItem(key); } catch { return null; }
}

function _setRaw(key, value) {
  try {
    localStorage.setItem(key, value);
    _enforceQuota();
  } catch {}
}

/**
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
