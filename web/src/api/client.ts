export const RESOURCE_FETCH_TIMEOUT_MS = 15_000;

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

export type ApiRequestInit = Omit<RequestInit, 'signal'> & {
  signal?: AbortSignal;
};

function readCookie(name: string): string | undefined {
  if (typeof document === 'undefined') {
    return undefined;
  }
  const prefix = `${name}=`;
  for (const part of document.cookie.split(';')) {
    const trimmed = part.trim();
    if (trimmed.startsWith(prefix)) {
      return decodeURIComponent(trimmed.slice(prefix.length));
    }
  }
  return undefined;
}

function linkAbortSignals(signals: AbortSignal[]): AbortSignal {
  const linked = new AbortController();
  const abortLinked = () => {
    if (!linked.signal.aborted) {
      linked.abort();
    }
  };

  for (const signal of signals) {
    if (signal.aborted) {
      abortLinked();
      return linked.signal;
    }
    signal.addEventListener('abort', abortLinked, { once: true });
  }

  return linked.signal;
}

function isMutatingMethod(method: string): boolean {
  const upper = method.toUpperCase();
  return upper !== 'GET' && upper !== 'HEAD' && upper !== 'OPTIONS';
}

async function parseApiError(response: Response): Promise<ApiError> {
  let code = 'HTTP_ERROR';
  let message = response.statusText || `HTTP ${response.status}`;

  try {
    const body: unknown = await response.json();
    if (body && typeof body === 'object') {
      const record = body as Record<string, unknown>;
      const errorField = record.error;
      if (errorField && typeof errorField === 'object') {
        const errObj = errorField as Record<string, unknown>;
        if (typeof errObj.code === 'string') {
          code = errObj.code;
        }
        if (typeof errObj.message === 'string') {
          message = errObj.message;
        }
      } else if (typeof errorField === 'string') {
        message = errorField;
      }
    }
  } catch {
    // Non-JSON error body; keep status text.
  }

  return new ApiError(response.status, code, message);
}

export function isAbortError(err: unknown): boolean {
  if (err instanceof DOMException && err.name === 'AbortError') {
    return true;
  }
  return err instanceof Error && err.name === 'AbortError';
}

export async function apiFetch(path: string, init: ApiRequestInit = {}): Promise<Response> {
  const timeoutCtrl = new AbortController();
  const timeoutId = setTimeout(() => {
    timeoutCtrl.abort();
  }, RESOURCE_FETCH_TIMEOUT_MS);

  const signals: AbortSignal[] = [timeoutCtrl.signal];
  if (init.signal) {
    signals.push(init.signal);
  }

  const method = (init.method ?? 'GET').toUpperCase();
  const headers = new Headers(init.headers ?? undefined);

  if (isMutatingMethod(method)) {
    const csrf = readCookie('csrfToken');
    if (csrf) {
      headers.set('X-CSRF-Token', csrf);
    }
  }

  if (init.body != null && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  try {
    return await fetch(path, {
      ...init,
      method,
      headers,
      credentials: 'include',
      signal: linkAbortSignals(signals),
    });
  } catch (err) {
    if (isAbortError(err) && timeoutCtrl.signal.aborted && !init.signal?.aborted) {
      throw new ApiError(0, 'TIMEOUT', `Request timed out after ${RESOURCE_FETCH_TIMEOUT_MS}ms`);
    }
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}

export async function apiJson<T>(path: string, init: ApiRequestInit = {}): Promise<T> {
  const response = await apiFetch(path, init);

  if (!response.ok) {
    throw await parseApiError(response);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const contentLength = response.headers.get('Content-Length');
  if (contentLength === '0') {
    return undefined as T;
  }

  return (await response.json()) as T;
}
