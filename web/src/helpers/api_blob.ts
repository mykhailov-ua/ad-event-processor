import * as auth from './auth.js';
import { ApiError, type ErrorPayload } from './api_client.js';

/**
 * Fetch a binary response from the API with CSRF on mutations.
 */
export async function apiBlob(path: string, init: RequestInit = {}): Promise<Blob> {
  const headers = new Headers(init.headers || {});
  const method = (init.method || 'GET').toUpperCase();
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    const csrf = auth.getCsrfToken();
    if (csrf) headers.set('X-CSRF-Token', csrf);
  }
  const res = await fetch(path, { ...init, headers, credentials: 'same-origin' });
  if (!res.ok) {
    const ct = res.headers.get('content-type') ?? '';
    let body: ErrorPayload = null;
    if (ct.includes('application/json')) {
      try {
        body = await res.json();
      } catch {
        // ignore
      }
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
