import { createContext, useCallback, useMemo, useState, type ReactNode } from 'react';

import { getSessionBootstrap } from '@/api/auth_api';
import { ApiError } from '@/api/client';
import type { AuthUser, SessionResponse } from '@/api/types';
import { useResource } from '@/api/use_resource';

// Root session snapshot for the SPA shell. Single GET /session bootstrap via useResource.
// authenticated/forbidden/unauthenticated derive from ApiError status; nav hide is not authorization.
type SessionState = {
  me: AuthUser;
  session: SessionResponse;
  eulaRequired: boolean | undefined;
  eulaAccepted: boolean | undefined;
  eulaVersion: string | undefined;
};

async function fetchSessionState(signal: AbortSignal): Promise<SessionState | null> {
  const bootstrap = await getSessionBootstrap(signal);
  if (!bootstrap) {
    return null;
  }
  return {
    me: bootstrap.user,
    session: bootstrap.session,
    eulaRequired: bootstrap.eula_required,
    eulaAccepted: bootstrap.eula_accepted,
    eulaVersion: bootstrap.eula_version,
  };
}

export type SessionContextValue = {
  session: SessionResponse | undefined;
  user: AuthUser | undefined;
  error: Error | undefined;
  loading: boolean;
  revalidating: boolean;
  authenticated: boolean;
  forbidden: boolean;
  unauthenticated: boolean;
  eulaRequired: boolean | undefined;
  eulaAccepted: boolean | undefined;
  eulaVersion: string | undefined;
  refetchSession: () => void;
};

export const SessionContext = createContext<SessionContextValue | undefined>(undefined);

function buildSessionValue(
  data: SessionState | null | undefined,
  error: Error | undefined,
  fetching: boolean
): Omit<SessionContextValue, 'revalidating' | 'refetchSession'> {
  const apiError = error instanceof ApiError ? error : undefined;
  const forbidden = apiError?.status === 403;
  const unauthenticated =
    apiError?.status === 401 || (!fetching && !error && (data === null || data === undefined));

  return {
    session: data?.session,
    user: data?.me,
    error,
    loading: fetching,
    authenticated: Boolean(data?.me) && !unauthenticated && !forbidden,
    forbidden,
    unauthenticated,
    eulaRequired: data?.eulaRequired,
    eulaAccepted: data?.eulaAccepted,
    eulaVersion: data?.eulaVersion,
  };
}

export type SessionProviderProps = {
  children: ReactNode;
};

export function SessionProvider({ children }: SessionProviderProps) {
  const [refreshToken, setRefreshToken] = useState(0);
  const { data, error, fetching, revalidating } = useResource(fetchSessionState, [refreshToken]);
  const refetchSession = useCallback(() => {
    setRefreshToken((value) => value + 1);
  }, []);
  const value = useMemo(
    () => ({
      ...buildSessionValue(data, error, fetching),
      revalidating,
      refetchSession,
    }),
    [data, error, fetching, revalidating, refetchSession]
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}
