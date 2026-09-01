import { apiJson, ApiError } from './client.js';
import { getMeta } from './platform_api.js';
import type {
  AuthLoginRequest,
  AuthLoginResponse,
  AuthRefreshResponse,
  AuthUser,
  PublicAcceptInviteRequest,
  PublicActivateRequest,
  PublicLoginResponse,
  SessionResponse,
} from './types.js';

export async function login(
  body: AuthLoginRequest,
  signal?: AbortSignal,
): Promise<AuthLoginResponse> {
  return apiJson<AuthLoginResponse>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function logout(signal?: AbortSignal): Promise<void> {
  await apiJson<void>('/api/v1/auth/logout', {
    method: 'POST',
    signal,
  });
}

export async function refreshSession(signal?: AbortSignal): Promise<AuthRefreshResponse> {
  return apiJson<AuthRefreshResponse>('/api/v1/auth/refresh', {
    method: 'POST',
    signal,
  });
}

export async function getAuthMe(signal?: AbortSignal): Promise<AuthUser> {
  return apiJson<AuthUser>('/api/v1/auth/me', { signal });
}

export async function getSession(signal?: AbortSignal): Promise<SessionResponse> {
  return apiJson<SessionResponse>('/api/v1/session', { signal });
}

export type SessionBootstrap = {
  user: AuthUser;
  session: SessionResponse;
  eula_required?: boolean;
  eula_accepted?: boolean;
  eula_version?: string;
};

async function composeSessionBootstrap(signal?: AbortSignal): Promise<SessionBootstrap> {
  const [user, session, meta] = await Promise.all([
    getAuthMe(signal),
    getSession(signal),
    getMeta(signal).catch(() => undefined),
  ]);
  return {
    user,
    session,
    eula_required: meta?.eula_required,
    eula_accepted: meta?.eula_accepted,
    eula_version: meta?.eula_version,
  };
}

export async function getSessionBootstrap(signal?: AbortSignal): Promise<SessionBootstrap | null> {
  try {
    return await apiJson<SessionBootstrap>('/api/v1/session/bootstrap', { signal });
  } catch (err) {
    if (!(err instanceof ApiError) || (err.status !== 401 && err.status !== 404)) {
      throw err;
    }
    try {
      return await composeSessionBootstrap(signal);
    } catch (fallbackErr) {
      if (fallbackErr instanceof ApiError && fallbackErr.status === 401) {
        return null;
      }
      throw fallbackErr;
    }
  }
}

export async function publicActivate(
  body: PublicActivateRequest,
  signal?: AbortSignal,
): Promise<PublicLoginResponse> {
  return apiJson<PublicLoginResponse>('/api/v1/public/activate', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function publicAcceptInvite(
  body: PublicAcceptInviteRequest,
  signal?: AbortSignal,
): Promise<PublicLoginResponse> {
  return apiJson<PublicLoginResponse>('/api/v1/public/invite/accept', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
