import * as storage from './storage.js';

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * Test whether a value is a well-formed customer UUID.
 *
 * @param {string} value
 * @returns {boolean}
 */
export function isCustomerUuid(value) {
  return UUID_RE.test(String(value).trim());
}

/**
 * Truncate a customer id for compact display.
 *
 * @param {string} id
 * @param {number} [len]
 * @returns {string}
 */
export function shortCustomerId(id, len = 8) {
  const s = String(id);
  return s.length > len + 1 ? `${s.slice(0, len)}…` : s;
}

/**
 * Record a customer visit in recent-navigation storage.
 *
 * @param {string} id
 * @returns {void}
 */
export function touchCustomerContext(id) {
  const trimmed = String(id).trim();
  if (!isCustomerUuid(trimmed)) return;
  storage.pushRecentCustomerId(trimmed);
}
