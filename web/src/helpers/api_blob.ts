import * as auth from './auth.js';
import { ApiError, type ErrorPayload } from './api_client.js';

export type ApiBlobResult = {
  blob: Blob;
  truncated: boolean;
  nextCursor: string | null;
  bytes: number | null;
};

export async function apiBlob(path: string, init: RequestInit = {}): Promise<Blob> {
  const result = await apiBlobResult(path, init);
  return result.blob;
}

export async function apiBlobResult(path: string, init: RequestInit = {}): Promise<ApiBlobResult> {
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
      } catch {}
    }
    const code = body?.error?.code ?? 'UNKNOWN';
    const msg = body?.error?.message ?? res.statusText;
    throw new ApiError(res.status, code, msg, res.headers.get('X-API-Stub') === 'true', body);
  }
  const bytesHdr = res.headers.get('X-Export-Bytes');
  return {
    blob: await res.blob(),
    truncated: res.headers.get('X-Export-Truncated') === 'true',
    nextCursor: res.headers.get('X-Next-Cursor'),
    bytes: bytesHdr ? Number(bytesHdr) : null,
  };
}
