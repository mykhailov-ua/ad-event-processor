import * as auth from './auth.js';
import { to } from '../lib/to.js';

const LOGIN_PATH = '/login';

export function redirectToLogin(reason = 'session'): void {
  auth.logoutLocal();
  if (typeof window === 'undefined') return;
  const suffix = reason ? `?reason=${encodeURIComponent(reason)}` : '';
  window.location.assign(`${LOGIN_PATH}${suffix}`);
}

export async function tryRefreshSession(): Promise<boolean> {
  const [refreshRes, refreshErr] = await to(
    fetch('/api/v1/auth/refresh', {
      method: 'POST',
      credentials: 'same-origin',
    })
  );
  if (refreshErr || !refreshRes?.ok) return false;

  const refreshCsrf = refreshRes.headers.get('X-CSRF-Token');
  if (refreshCsrf) auth.setCsrfFromLoginResponse(refreshCsrf);

  const [meRes, meErr] = await to(fetch('/api/v1/auth/me', { credentials: 'same-origin' }));
  if (meErr || !meRes?.ok) return false;

  const csrf = meRes.headers.get('X-CSRF-Token');
  if (csrf) auth.setCsrfFromLoginResponse(csrf);

  const [text, textErr] = await to(meRes.text());
  if (textErr) return false;
  if (!text) return true;

  try {
    const data = JSON.parse(text) as {
      id: string;
      email: string;
      role: string;
      customer_id?: string;
      permissions?: string[];
    };
    auth.setUser({
      id: data.id,
      email: data.email,
      role: data.role,
      customer_id: data.customer_id ?? '',
      permissions: data.permissions ?? [],
    });
  } catch {
    return false;
  }
  return true;
}

export function isAuthMutationPath(path: string): boolean {
  return (
    path.includes('/auth/login') || path.includes('/auth/register') || path.includes('/auth/logout')
  );
}
