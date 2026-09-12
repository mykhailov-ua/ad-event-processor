// HTTP transport for /api/v1/* (frontend-modular.mdc).
// Runs in the browser main thread on Cold surfaces; pairs with useResource abort on dep change.
//
// Contracts:
// - RESOURCE_FETCH_TIMEOUT_MS hard deadline; timeout abort maps to ApiError(0, TIMEOUT).
// - Caller AbortSignal and timeout are linked; abort from caller is not rewritten to TIMEOUT.
// - Mutating methods attach X-CSRF-Token from csrfToken cookie when present.
// - credentials: include for session cookie on same-origin control plane.
//
// Verify:
// cd web && npm run typecheck
// bash scripts/ci/admin/web.sh
import { ApiError } from '@/api/api_error';

export { ApiError } from '@/api/api_error';

export const RESOURCE_FETCH_TIMEOUT_MS = 15_000;

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

function featureRequiredMessage(record: Record<string, unknown>): string {
  const featureKey =
    typeof record.feature_key === 'string'
      ? record.feature_key
      : typeof record.feature_required === 'string'
        ? record.feature_required
        : '';
  const planCode = typeof record.plan_code === 'string' ? record.plan_code : '';
  if (featureKey && planCode) {
    return `${featureKey} requires ${planCode} plan`;
  }
  if (featureKey) {
    return featureKey;
  }
  return '';
}

function normalizeApiErrorCode(rawCode: string): string {
  if (rawCode === 'feature_required') {
    return 'FEATURE_REQUIRED';
  }
  return rawCode;
}

export async function parseApiError(response: Response): Promise<ApiError> {
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
          code = normalizeApiErrorCode(errObj.code);
        }
        if (typeof errObj.message === 'string') {
          message = errObj.message;
        }
        if (code === 'FEATURE_REQUIRED' && message === response.statusText) {
          const featureMessage = featureRequiredMessage({ ...record, ...errObj });
          if (featureMessage !== '') {
            message = featureMessage;
          }
        }
      } else if (typeof errorField === 'string') {
        if (errorField === 'feature_required') {
          code = 'FEATURE_REQUIRED';
          const featureMessage = featureRequiredMessage(record);
          message = featureMessage !== '' ? featureMessage : errorField;
        } else {
          message = errorField;
        }
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
  } catch (err: unknown) {
    if (isAbortError(err) && timeoutCtrl.signal.aborted && !init.signal?.aborted) {
      throw new ApiError(0, 'TIMEOUT', `Request timed out after ${RESOURCE_FETCH_TIMEOUT_MS}ms`);
    }
    throw err;
  } finally {
    clearTimeout(timeoutId);
  }
}

export type JsonParser<T> = (value: unknown) => T;

export async function apiJsonValidated<T>(
  path: string,
  init: ApiRequestInit,
  parse: JsonParser<T>
): Promise<T> {
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

  const payload: unknown = await response.json();
  return parse(payload);
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

export async function apiJsonArray<T>(path: string, init: ApiRequestInit = {}): Promise<T[]> {
  const payload = await apiJson<unknown>(path, init);
  if (!Array.isArray(payload)) {
    throw new ApiError(502, 'INVALID_RESPONSE', 'Expected JSON array response');
  }
  return payload as T[];
}

export async function apiJsonArrayWithTotalCount<T>(
  path: string,
  init: ApiRequestInit = {},
  options: { totalHeader?: string } = {}
): Promise<{ items: T[]; total: number }> {
  const response = await apiFetch(path, init);

  if (!response.ok) {
    throw await parseApiError(response);
  }

  const payload: unknown = await response.json();
  if (!Array.isArray(payload)) {
    throw new ApiError(502, 'INVALID_RESPONSE', 'Expected JSON array response');
  }

  const headerName = options.totalHeader ?? 'X-Total-Count';
  const totalHeader = response.headers.get(headerName);
  const parsedTotal = totalHeader != null ? Number.parseInt(totalHeader, 10) : payload.length;
  const total = Number.isFinite(parsedTotal) ? parsedTotal : payload.length;

  return { items: payload as T[], total };
}

export async function apiJsonArrayWithTotalCountValidated<T>(
  path: string,
  init: ApiRequestInit,
  parseItem: JsonParser<T>,
  options: { totalHeader?: string } = {}
): Promise<{ items: T[]; total: number }> {
  const response = await apiFetch(path, init);

  if (!response.ok) {
    throw await parseApiError(response);
  }

  const payload: unknown = await response.json();
  if (!Array.isArray(payload)) {
    throw new ApiError(502, 'INVALID_RESPONSE', 'Expected JSON array response');
  }

  const items = payload.map(parseItem);

  const headerName = options.totalHeader ?? 'X-Total-Count';
  const totalHeader = response.headers.get(headerName);
  const parsedTotal = totalHeader != null ? Number.parseInt(totalHeader, 10) : items.length;
  const total = Number.isFinite(parsedTotal) ? parsedTotal : items.length;

  return { items, total };
}
