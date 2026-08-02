import * as auth from './auth.js';
import { ApiError } from './api_client.js';

/**
 * Fetch a binary response from the API with CSRF on mutations.
 *
 * @param {string} path
 * @param {RequestInit} [init]
 * @returns {Promise<Blob>}
 */
export async function apiBlob(path, init = {}) {
  const headers = new Headers(init.headers || {});
  const method = (init.method || 'GET').toUpperCase();
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    const csrf = auth.getCsrfToken();
    if (csrf) headers.set('X-CSRF-Token', csrf);
  }
  const res = await fetch(path, { ...init, headers, credentials: 'same-origin' });
  if (!res.ok) {
    const ct = res.headers.get('content-type') ?? '';
    let body = null;
    if (ct.includes('application/json')) {
      try { body = await res.json(); } catch {}
    }
    const code = body?.error?.code ?? 'UNKNOWN';
    const msg = body?.error?.message ?? res.statusText;
    throw new ApiError(
      res.status,
      code,
      msg,
      res.headers.get('X-API-Stub') === 'true',
      body,
    );
  }
  return res.blob();
}
