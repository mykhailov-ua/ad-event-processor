import type { AuthUser } from '../types/auth.js';
import * as storage from './storage.js';

export type { AuthUser };

let user: AuthUser | null = null;
let csrf: string | null = null;

export function hydrateFromBoot(scriptId = '__BOOT__'): AuthUser | null {
  const el = document.getElementById(scriptId);
  if (!el) return hydrateCsrfFromCookie();
  try {
    const data = JSON.parse(el.textContent ?? '') as {
      user?: AuthUser | null;
      permissions?: string[];
    };
    user = data.user ?? null;
    if (user && Array.isArray(data.permissions)) {
      user.permissions = data.permissions;
    }
    hydrateCsrfFromCookie();
    return user;
  } catch {
    return hydrateCsrfFromCookie();
  }
}

function hydrateCsrfFromCookie(): AuthUser | null {
  const fromCookie = readCsrfCookie();
  if (fromCookie) csrf = fromCookie;
  return user;
}

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

export function setUser(next: AuthUser): void {
  user = next;
}

export function getUser(): AuthUser | null {
  return user;
}

export function setCsrfFromLoginResponse(header: string): void {
  csrf = header;
}

export function getCsrfToken(): string | null {
  if (csrf) return csrf;
  return readCsrfCookie();
}

export function logoutLocal(): void {
  user = null;
  csrf = null;
  storage.clearIdempotencyPendingAll();
}
