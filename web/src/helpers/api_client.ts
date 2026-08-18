import * as auth from './auth.js';
import { clearScope, getOrCreate } from './idempotency.js';
import { removeIdempotencyPending, setIdempotencyPending } from './storage.js';
import { isAuthMutationPath, redirectToLogin, tryRefreshSession } from './session.js';
import { recordApiTiming } from './api_timing.js';

const LARGE_BODY_THRESHOLD = 32_000;
const MUTATION_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

export type ApiInit = RequestInit & {
  idempotencyScope?: string;
  _authRetry?: boolean;
};

export type ApiResult<T = unknown> = {
  data: T;
  status: number;
  headers: Headers;
};

export type ErrorPayload = {
  error?: { code?: string; message?: string };
  errors?: string[];
  [key: string]: unknown;
} | null;

export class ApiError extends Error {
  status: number;
  code: string;
  stub: boolean;
  payload: ErrorPayload;
  responseHeaders: Headers | null;

  constructor(
    status: number,
    code: string,
    message: string,
    stub = false,
    payload: ErrorPayload = null,
    responseHeaders: Headers | null = null
  ) {
    super(message || code);
    this.status = status;
    this.code = code;
    this.stub = stub;
    this.payload = payload;
    this.responseHeaders = responseHeaders;
  }
}

export class AuthError extends Error {
  constructor() {
    super('unauthorized');
  }
}

export class NetworkError extends Error {
  constructor(message = 'network request failed') {
    super(message);
    this.name = 'NetworkError';
  }
}

export async function api<T = unknown>(path: string, init: ApiInit = {}): Promise<ApiResult<T>> {
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
  let res: Response;
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
        return api<T>(path, { ...init, _authRetry: true });
      }
    }
    redirectToLogin('session');
    throw new AuthError();
  }

  const stub = res.headers.get('X-API-Stub') === 'true';

  let body: unknown = null;
  const ct = res.headers.get('content-type') ?? '';
  if (ct.includes('application/json')) {
    const text = await res.text();
    if (text.length > LARGE_BODY_THRESHOLD) {
      body = await parseInWorker(text);
    } else if (text) {
      try {
        body = JSON.parse(text);
      } catch {}
    }
  }

  if (res.ok && init.idempotencyScope) {
    removeIdempotencyPending(init.idempotencyScope);
    clearScope(init.idempotencyScope);
  }

  const csrfHeader = res.headers.get('X-CSRF-Token');
  if (csrfHeader) auth.setCsrfFromLoginResponse(csrfHeader);

  if (!res.ok) {
    const errBody = body as ErrorPayload;
    const code = errBody?.error?.code ?? 'UNKNOWN';
    const msg = errBody?.error?.message ?? res.statusText;
    if (res.status === 429 || code === 'RATE_LIMITED' || code === 'TOO_MANY_REQUESTS') {
      const retryRaw = res.headers.get('Retry-After');
      const retryAfterSec = retryRaw ? Number.parseInt(retryRaw, 10) : 0;
      window.dispatchEvent(
        new CustomEvent('admin:rate-limited', {
          detail: { retryAfterSec: Number.isFinite(retryAfterSec) ? retryAfterSec : 0 },
        })
      );
    }
    throw new ApiError(res.status, code, msg, stub, errBody, res.headers);
  }

  return { data: body as T, status: res.status, headers: res.headers };
}

let parseJsonWorker: Worker | null = null;

function getParseJsonWorker(): Worker | null {
  if (typeof Worker === 'undefined') return null;
  if (!parseJsonWorker) {
    parseJsonWorker = new Worker('/src/workers/parse_json.worker.js', { type: 'module' });
  }
  return parseJsonWorker;
}

async function parseInWorker(text: string): Promise<unknown> {
  if (typeof Worker === 'undefined') {
    try {
      return JSON.parse(text);
    } catch {
      return null;
    }
  }
  const worker = getParseJsonWorker();
  if (!worker) {
    try {
      return JSON.parse(text);
    } catch {
      return null;
    }
  }
  return new Promise((resolve) => {
    const onMessage = (e: MessageEvent) => {
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

function isMutation(method: string): boolean {
  return MUTATION_METHODS.has(method.toUpperCase());
}

export { isMutation };

async function hashRequestBody(body: BodyInit | null | undefined): Promise<string> {
  const text = body == null ? '' : typeof body === 'string' ? body : String(body);
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
