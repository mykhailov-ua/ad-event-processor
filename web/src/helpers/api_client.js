import * as auth from './auth.js';
import { clearScope, getOrCreate } from './idempotency.js';
import { removeIdempotencyPending, setIdempotencyPending } from './storage.js';
import {
  isAuthMutationPath,
  redirectToLogin,
  tryRefreshSession,
} from './session.js';
import { recordApiTiming } from './api_timing.js';

const LARGE_BODY_THRESHOLD = 32_000;
const MUTATION_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

/**
 * @typedef {RequestInit & { idempotencyScope?: string, _authRetry?: boolean }} ApiInit
 */

/** HTTP API error with status, code, and optional stub flag. */
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

/** Session expired or missing credentials. */
export class AuthError extends Error {
  constructor() { super('unauthorized'); }
}

/** Network or fetch-layer failure. */
export class NetworkError extends Error {
  /**
   * @param {string} [message]
   */
  constructor(message = 'network request failed') {
    super(message);
    this.name = 'NetworkError';
  }
}

/**
 * Perform a same-origin API request with CSRF, idempotency, and session recovery.
 *
 * @param {string} path
 * @param {ApiInit} [init]
 * @returns {Promise<{data: any, status: number, headers: Headers}>}
 * @throws {AuthError|NetworkError|ApiError}
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
      const key = getOrCreate(init.idempotencyScope);
      headers.set('Idempotency-Key', key);
      const bodyHash = await hashRequestBody(init.body);
      setIdempotencyPending(init.idempotencyScope, { key, bodyHash, ts: Date.now() });
    }
  }

  const t0 = performance.now();
  let res;
  try {
    res = await fetch(path, { ...init, headers, credentials: 'same-origin', signal: init.signal });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    const msg = err instanceof Error ? err.message : 'fetch failed';
    window.dispatchEvent(new CustomEvent('admin:network-error', { detail: { message: msg } }));
    throw new NetworkError(msg.includes('fetch') ? msg : 'Failed to fetch');
  } finally {
    recordApiTiming(path, performance.now() - t0);
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

  if (res.ok && init.idempotencyScope) {
    removeIdempotencyPending(init.idempotencyScope);
    clearScope(init.idempotencyScope);
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

/** @type {Worker|null} */
let parseJsonWorker = null;

/**
 * Return a shared JSON parse worker instance.
 *
 * @returns {Worker|null}
 */
function getParseJsonWorker() {
  if (typeof Worker === 'undefined') return null;
  if (!parseJsonWorker) {
    parseJsonWorker = new Worker(
      new URL('../workers/parse_json.worker.js', import.meta.url),
      { type: 'module' },
    );
  }
  return parseJsonWorker;
}

/**
 * Parse a large JSON payload in a Web Worker when available.
 *
 * @param {string} text
 * @returns {Promise<any|null>}
 */
async function parseInWorker(text) {
  if (typeof Worker === 'undefined') {
    try { return JSON.parse(text); } catch { return null; }
  }
  const worker = getParseJsonWorker();
  if (!worker) {
    try { return JSON.parse(text); } catch { return null; }
  }
  return new Promise((resolve) => {
    const onMessage = (e) => {
      worker.removeEventListener('message', onMessage);
      worker.removeEventListener('error', onError);
      resolve(e.data);
    };
    const onError = () => {
      worker.removeEventListener('message', onMessage);
      worker.removeEventListener('error', onError);
      resolve(null);
    };
    worker.addEventListener('message', onMessage);
    worker.addEventListener('error', onError);
    worker.postMessage(text);
  });
}

/**
 * Test whether an HTTP method is a mutation.
 *
 * @param {string} method
 * @returns {boolean}
 */
function isMutation(method) {
  return MUTATION_METHODS.has(method.toUpperCase());
}

export { isMutation };

/**
 * Hash a request body for idempotency crash recovery.
 *
 * @param {BodyInit|null|undefined} body
 * @returns {Promise<string>}
 */
async function hashRequestBody(body) {
  const text = body == null ? '' : (typeof body === 'string' ? body : String(body));
  if (typeof crypto !== 'undefined' && crypto.subtle) {
    const data = new TextEncoder().encode(text);
    const buf = await crypto.subtle.digest('SHA-256', data);
    return [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, '0')).join('');
  }
  let h = 5381;
  for (let i = 0; i < text.length; i++) {
    h = ((h << 5) + h) ^ text.charCodeAt(i);
  }
  return (h >>> 0).toString(16);
}
