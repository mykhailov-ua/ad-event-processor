import * as auth from './auth.js';
import { AuthError, NetworkError } from './api_client.js';
import { redirectToLogin, tryRefreshSession } from './session.js';

export type BlobFetchResult = {
  blob: Blob;
  truncated: boolean;
  nextCursor: string | null;
};

export async function fetchBlob(
  path: string,
  init: RequestInit = {},
  authRetry = false
): Promise<BlobFetchResult> {
  const headers = new Headers(init.headers || {});
  const csrf = auth.getCsrfToken();
  if (csrf) headers.set('X-CSRF-Token', csrf);

  let res: Response;
  try {
    res = await fetch(path, {
      ...init,
      headers,
      credentials: 'same-origin',
    });
  } catch (err) {
    const msg = err instanceof Error ? err.message : 'fetch failed';
    throw new NetworkError(msg);
  }

  if (res.status === 401) {
    if (!authRetry) {
      const recovered = await tryRefreshSession();
      if (recovered) return fetchBlob(path, init, true);
    }
    redirectToLogin('session');
    throw new AuthError();
  }

  if (!res.ok) {
    throw new Error(res.statusText || 'download failed');
  }

  const truncated = res.headers.get('X-Export-Truncated') === 'true';
  const nextCursor = res.headers.get('X-Next-Cursor');
  const blob = await res.blob();
  return { blob, truncated, nextCursor };
}

export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}
