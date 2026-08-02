import * as auth from './auth.js';
import { getOrCreate } from './idempotency.js';
import {
  isAuthMutationPath,
  redirectToLogin,
  tryRefreshSession,
} from './session.js';

const LARGE_BODY_THRESHOLD = 100_000;
const MUTATION_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

/**
 * @typedef {RequestInit & { idempotencyScope?: string, _authRetry?: boolean }} ApiInit
 */

export class ApiError extends Error {
  /**
   * @param {number} status
   * @param {string} code
   * @param {string} message
   * @param {boolean} stub
   * @param {object|null} [payload]
   * @param {Headers|null} [responseHeaders]
   */
  constructor(status, code, message, stub = false, payload = null, responseHeaders = null) {
    super(message || code);
    this.status = status;
    this.code = code;
    this.stub = stub;
    this.payload = payload;
    this.responseHeaders = responseHeaders;
  }
}

export class AuthError extends Error {
  constructor() { super('unauthorized'); }
}

export class NetworkError extends Error {
  constructor(message = 'network request failed') {
    super(message);
    this.name = 'NetworkError';
  }
}

/**
 * @param {string} path
 * @param {ApiInit} [init]
 * @returns {Promise<{data: any, status: number, headers: Headers}>}
 */
export async function api(path, init = {}) {
  const method = (init.method || 'GET').toUpperCase();
  const headers = new Headers(init.headers || {});

  if (!headers.has('Content-Type') && init.body) {
    headers.set('Content-Type', 'application/json');
  }

  if (MUTATION_METHODS.has(method)) {
    const csrf = auth.getCsrfToken();
    if (csrf) headers.set('X-CSRF-Token', csrf);
    if (init.idempotencyScope) {
      headers.set('Idempotency-Key', getOrCreate(init.idempotencyScope));
    }
  }

  let res;
  try {
    res = await fetch(path, { ...init, headers, credentials: 'same-origin', signal: init.signal });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    const msg = err instanceof Error ? err.message : 'fetch failed';
    window.dispatchEvent(new CustomEvent('admin:network-error', { detail: { message: msg } }));
    throw new NetworkError(msg.includes('fetch') ? msg : 'Failed to fetch');
  }

  if (res.status === 401) {
    if (!init._authRetry && !isAuthMutationPath(path)) {
      const recovered = await tryRefreshSession();
      if (recovered) {
        return api(path, { ...init, _authRetry: true });
      }
    }
    redirectToLogin('session');
    throw new AuthError();
  }

  const stub = res.headers.get('X-API-Stub') === 'true';

  let body = null;
  const ct = res.headers.get('content-type') ?? '';
  if (ct.includes('application/json')) {
    const text = await res.text();
    if (text.length > LARGE_BODY_THRESHOLD) {
      body = await parseInWorker(text);
    } else if (text) {
      try { body = JSON.parse(text); } catch {}
    }
  }

  if (!res.ok) {
    const code = body?.error?.code ?? 'UNKNOWN';
    const msg = body?.error?.message ?? res.statusText;
    if (res.status === 429 || code === 'RATE_LIMITED' || code === 'TOO_MANY_REQUESTS') {
      const retryRaw = res.headers.get('Retry-After');
      const retryAfterSec = retryRaw ? Number.parseInt(retryRaw, 10) : 0;
      window.dispatchEvent(new CustomEvent('admin:rate-limited', {
        detail: { retryAfterSec: Number.isFinite(retryAfterSec) ? retryAfterSec : 0 },
      }));
    }
    throw new ApiError(res.status, code, msg, stub, body, res.headers);
  }

  return { data: body, status: res.status, headers: res.headers };
}

async function parseInWorker(text) {
  if (typeof Worker === 'undefined') {
    try { return JSON.parse(text); } catch { return null; }
  }
  return new Promise((resolve) => {
    const w = new Worker(new URL('../workers/parse_json.worker.js', import.meta.url), { type: 'module' });
    w.onmessage = (e) => { resolve(e.data); w.terminate(); };
    w.onerror = () => { resolve(null); w.terminate(); };
    w.postMessage(text);
  });
}

/**
 * @param {string} method
 * @returns {boolean}
 */
function isMutation(method) {
  return MUTATION_METHODS.has(method.toUpperCase());
}

export { isMutation };
