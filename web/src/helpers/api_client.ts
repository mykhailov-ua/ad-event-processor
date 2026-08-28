import * as auth from './auth.js';
import { isAuthMutationPath, redirectToLogin, tryRefreshSession } from './session.js';
import { parseJsonText } from './parse_json_client.js';

const MUTATION_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

export type ApiInit = RequestInit & {
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
  }

  let res: Response;
  try {
    res = await fetch(path, { ...init, headers, credentials: 'same-origin', signal: init.signal });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    const msg = err instanceof Error ? err.message : 'fetch failed';
    throw new NetworkError(msg.includes('fetch') ? msg : 'Failed to fetch');
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
    if (text) {
      try {
        body = await parseJsonText(text);
      } catch {
        body = null;
      }
    }
  }

  const csrfHeader = res.headers.get('X-CSRF-Token');
  if (csrfHeader) auth.setCsrfFromLoginResponse(csrfHeader);

  if (!res.ok) {
    const errBody = body as ErrorPayload;
    const code = errBody?.error?.code ?? 'UNKNOWN';
    const msg = errBody?.error?.message ?? res.statusText;
    throw new ApiError(res.status, code, msg, stub, errBody, res.headers);
  }

  return { data: body as T, status: res.status, headers: res.headers };
}

export function isMutation(method: string): boolean {
  return MUTATION_METHODS.has(method.toUpperCase());
}
