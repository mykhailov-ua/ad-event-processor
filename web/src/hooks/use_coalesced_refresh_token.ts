import { useCallback, useState } from 'react';

import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';

// refreshToken dep for useResource/effects; useCoalescedBumpRefresh drops double-clicks while fetching.
export function useRefreshToken() {
  const [refreshToken, setRefreshToken] = useState(0);

  const bumpRefresh = useCallback(() => {
    setRefreshToken((value) => value + 1);
  }, []);

  return { refreshToken, bumpRefresh };
}

export function useCoalescedBumpRefresh(bumpRefresh: () => void, inFlight: boolean) {
  return useCoalescedCallback(bumpRefresh, {
    inFlightGuard: true,
    inFlight,
  });
}
