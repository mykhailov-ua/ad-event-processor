import { getSession } from '@/api/auth_api';
import type { SessionResponse } from '@/api/types';

// Post-login landing: export hub is the primary Control Plane surface.
export function defaultHomePath(_session: SessionResponse): string {
  return '/exports';
}

export async function fetchSession(signal?: AbortSignal): Promise<SessionResponse> {
  return getSession(signal);
}
