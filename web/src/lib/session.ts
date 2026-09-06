import { getSession } from '@/api/auth_api';
import type { SessionResponse } from '@/api/types';

// Post-login landing path from session.role wire field; does not replace server route guards.
export function defaultHomePath(session: SessionResponse): string {
  const role = session.role?.toLowerCase() ?? '';
  if (role === 'ops' || role === 'admin' || role === 'operator') {
    return '/ops';
  }
  return '/customers';
}

export async function fetchSession(signal?: AbortSignal): Promise<SessionResponse> {
  return getSession(signal);
}
