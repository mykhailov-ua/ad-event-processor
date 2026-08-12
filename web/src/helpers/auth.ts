import * as idempotency from './idempotency.js';
import * as storage from './storage.js';
import type { AuthUser } from '../types/api/auth.js';

export type { AuthUser } from '../types/api/auth.js';

let _user: AuthUser | null = null;
let _csrf: string | null = null;

/**
 * Hydrate auth state from the boot script and CSRF cookie.
 */
export function hydrateFromBoot(scriptId = '__BOOT__'): AuthUser | null {
  const el = document.getElementById(scriptId);
  if (!el) return hydrateCsrfFromCookie();
  try {
    const data = JSON.parse(el.textContent ?? '') as {
      user?: AuthUser | null;
      permissions?: string[];
    };
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
 * Load CSRF from cookie when boot JSON is absent or invalid.
 */
function hydrateCsrfFromCookie(): AuthUser | null {
  const fromCookie = readCsrfCookie();
  if (fromCookie) _csrf = fromCookie;
  return _user;
}

/**
 * Parse the csrfToken cookie value.
 */
function readCsrfCookie(): string | null {
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
 * Replace the in-memory authenticated user.
 */
export function setUser(user: AuthUser): void {
  _user = user;
}

/**
 * Return the current authenticated user, if any.
 */
export function getUser(): AuthUser | null {
  return _user;
}

/**
 * Store CSRF token from a login response header.
 */
export function setCsrfFromLoginResponse(header: string): void {
  _csrf = header;
}

/**
 * Return the in-memory or cookie-backed CSRF token.
 */
export function getCsrfToken(): string | null {
  if (_csrf) return _csrf;
  return readCsrfCookie();
}

/**
 * Clear auth, CSRF, and idempotency state on logout.
 */
export function logoutLocal(): void {
  _user = null;
  _csrf = null;
  idempotency.clearAll();
  storage.clearIdempotencyPendingAll();
}
