import { getSession } from '@/api/auth_api';
import type { SessionResponse } from '@/api/types';

function roleHomeEnabled(): boolean {
  return import.meta.env.VITE_CP_ROLE_HOME === '1';
}

// Post-login landing: export hub by default; optional role dashboards when VITE_CP_ROLE_HOME=1.
export function defaultHomePath(session: SessionResponse): string {
  if (roleHomeEnabled()) {
    if (session.role === 'TL') {
      return '/dashboards/adops';
    }
    if (session.role === 'MB' || session.role === 'B') {
      return '/dashboards/buyer';
    }
  }
  return '/exports';
}

export async function fetchSession(signal?: AbortSignal): Promise<SessionResponse> {
  return getSession(signal);
}
