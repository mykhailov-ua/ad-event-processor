import * as idempotency from './idempotency.js';
import * as storage from './storage.js';

/**
 * @typedef {Object} AuthUser
 * @property {string} id
 * @property {string} email
 * @property {string} role
 * @property {string} customer_id
 * @property {string[]} permissions
 */

let _user = null;
let _csrf = null;

/**
 * @param {string} scriptId
 * @returns {AuthUser|null}
 */
export function hydrateFromBoot(scriptId = '__BOOT__') {
  const el = document.getElementById(scriptId);
  if (!el) return hydrateCsrfFromCookie();
  try {
    const data = JSON.parse(el.textContent);
    _user = data.user ?? null;
    if (_user && Array.isArray(data.permissions)) {
      _user.permissions = data.permissions;
    }
    hydrateCsrfFromCookie();
    return _user;
  } catch {
    return hydrateCsrfFromCookie();
  }
}

/**
 * @returns {AuthUser|null}
 */
function hydrateCsrfFromCookie() {
  const fromCookie = readCsrfCookie();
  if (fromCookie) _csrf = fromCookie;
  return _user;
}

/**
 * @returns {string|null}
 */
function readCsrfCookie() {
  if (typeof document === 'undefined') return null;
  const parts = document.cookie.split(';');
  for (const part of parts) {
    const [rawKey, ...rest] = part.trim().split('=');
    if (rawKey === 'csrfToken' && rest.length > 0) {
      return decodeURIComponent(rest.join('='));
    }
  }
  return null;
}

/**
 * @param {AuthUser} user
 */
export function setUser(user) {
  _user = user;
}

/**
 * @returns {AuthUser|null}
 */
export function getUser() {
  return _user;
}

/**
 * @param {string} header
 */
export function setCsrfFromLoginResponse(header) {
  _csrf = header;
}

/**
 * @returns {string|null}
 */
export function getCsrfToken() {
  if (_csrf) return _csrf;
  return readCsrfCookie();
}

/**
 * @returns {void}
 */
export function logoutLocal() {
  _user = null;
  _csrf = null;
  idempotency.clearAll();
  storage.clearIdempotencyPendingAll();
}
