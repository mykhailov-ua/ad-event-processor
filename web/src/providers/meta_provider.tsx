import { createContext, useCallback, useMemo, useState, type ReactNode } from 'react';

import { getMeta } from '@/api/platform_api';
import type { MetaResponse } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { licenseNeedsSetup, readBootstrapComplete } from '@/lib/install_meta';

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
  const [refreshToken, setRefreshToken] = useState(0);

  const { data, error, fetching } = useResource(
    (signal) => getMeta(signal),
    [refreshToken],
  );

  const refreshMeta = useCallback(() => {
    setRefreshToken((value) => value + 1);
  }, []);

  const value = useMemo(
    (): MetaContextValue => ({
      meta: data,
      error,
      loading: fetching,
      bootstrapComplete: readBootstrapComplete(data),
      licenseNeedsSetup: licenseNeedsSetup(data),
      refreshMeta,
    }),
    [data, error, fetching, refreshMeta],
  );

  return <MetaContext.Provider value={value}>{children}</MetaContext.Provider>;
}
