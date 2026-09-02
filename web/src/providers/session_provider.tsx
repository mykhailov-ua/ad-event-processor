import { createContext, useMemo, type ReactNode } from 'react';

import { getSessionBootstrap } from '@/api/auth_api';
import { ApiError } from '@/api/client';
import type { AuthUser, SessionResponse } from '@/api/types';
import { useResource } from '@/api/use_resource';

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
  authenticated: boolean;
  forbidden: boolean;
  unauthenticated: boolean;
  eulaRequired: boolean | undefined;
  eulaAccepted: boolean | undefined;
  eulaVersion: string | undefined;
};

export const SessionContext = createContext<SessionContextValue | undefined>(undefined);

function buildSessionValue(
  data: SessionState | null | undefined,
  error: Error | undefined,
  fetching: boolean,
): SessionContextValue {
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
  const { data, error, fetching } = useResource(fetchSessionState, []);
  const value = useMemo(
    () => buildSessionValue(data, error, fetching),
    [data, error, fetching],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}
