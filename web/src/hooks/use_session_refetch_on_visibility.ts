import { useEffect, useRef } from 'react';

import { refreshSession } from '@/api/auth_api';
import { DEFAULT_COALESCE_WINDOW_MS, coalesceUserAction } from '@/lib/coalesced_user_action';

// Refetch JWT + bootstrap when tab becomes visible so nav permissions catch admin grants without re-login.
export function useSessionRefetchOnVisibility(refetchSession: () => void) {
  const lastFiredAtMsRef = useRef(0);
  const inFlightRef = useRef(false);

  useEffect(() => {
    const syncSession = async () => {
      if (document.visibilityState !== 'visible') {
        return;
      }
      const verdict = coalesceUserAction({
        lastFiredAtMs: lastFiredAtMsRef.current,
        nowMs: Date.now(),
        windowMs: DEFAULT_COALESCE_WINDOW_MS,
        inFlight: inFlightRef.current,
        inFlightGuard: true,
      });
      if (verdict !== 'allow') {
        return;
      }
      lastFiredAtMsRef.current = Date.now();
      inFlightRef.current = true;
      try {
        await refreshSession();
      } catch {
        // refresh may fail when session expired; bootstrap refetch still reconciles policy snapshot
      } finally {
        inFlightRef.current = false;
        refetchSession();
      }
    };

    const onVisibilityChange = () => {
      void syncSession();
    };
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => document.removeEventListener('visibilitychange', onVisibilityChange);
  }, [refetchSession]);
}
