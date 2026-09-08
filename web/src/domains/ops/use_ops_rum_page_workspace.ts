// RUM snapshot: lazy load (loadToken 0 skips fetch until operator clicks Load).
import { useState } from 'react';

import { getOpsRum } from '@/api/ops_api';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh } from '@/hooks/use_coalesced_refresh_token';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useOpsRumPageWorkspace() {
  const [loadToken, setLoadToken] = useState(0);

  const rumResource = useResource(
    (signal) => {
      if (loadToken === 0) {
        return skipLazyFetch();
      }
      return getOpsRum(signal);
    },
    [loadToken]
  );

  const onLoad = useCoalescedBumpRefresh(() => {
    setLoadToken((value) => value + 1);
  }, rumResource.fetching);

  return {
    payload: rumResource.data,
    fetching: rumResource.fetching,
    error: rumResource.error,
    hasSnapshot: rumResource.data != null,
    onLoad,
  };
}
