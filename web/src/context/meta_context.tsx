import { createContext, useMemo, type ReactNode } from 'react';

import type { MetaResponse } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { fetchMetaCached, invalidateMetaCache } from '@/lib/get_meta_cache';
import { licenseNeedsSetup, readBootstrapComplete } from '@/lib/install_meta';

// GET /meta snapshot shared across shell, license gate, and EULA flows (session-level cache).
export type MetaContextValue = {
  meta: MetaResponse | undefined;
  error: Error | undefined;
  loading: boolean;
  bootstrapComplete: boolean;
  licenseNeedsSetup: boolean;
  refreshMeta: () => void;
};

export const MetaContext = createContext<MetaContextValue | undefined>(undefined);

export type MetaProviderProps = {
  children: ReactNode;
};

export function MetaProvider({ children }: MetaProviderProps) {
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource(
    (signal) => fetchMetaCached(signal),
    [refreshToken]
  );

  const refreshMeta = useCoalescedBumpRefresh(() => {
    invalidateMetaCache();
    bumpRefresh();
  }, fetching);

  const value = useMemo(
    (): MetaContextValue => ({
      meta: data,
      error,
      loading: fetching,
      bootstrapComplete: readBootstrapComplete(data),
      licenseNeedsSetup: licenseNeedsSetup(data),
      refreshMeta,
    }),
    [data, error, fetching, refreshMeta]
  );

  return <MetaContext.Provider value={value}>{children}</MetaContext.Provider>;
}
