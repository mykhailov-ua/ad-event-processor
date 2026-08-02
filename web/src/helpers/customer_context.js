import * as storage from './storage.js';

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * @param {string} value
 */
export function isCustomerUuid(value) {
  return UUID_RE.test(String(value).trim());
}

/**
 * @param {string} id
 * @param {number} [len]
 */
export function shortCustomerId(id, len = 8) {
  const s = String(id);
  return s.length > len + 1 ? `${s.slice(0, len)}…` : s;
}

/**
 * @param {string} id
 */
export function touchCustomerContext(id) {
  const trimmed = String(id).trim();
  if (!isCustomerUuid(trimmed)) return;
  storage.pushRecentCustomerId(trimmed);
}
