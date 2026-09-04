import { useCallback, useRef } from 'react';

import {
  coalesceUserAction,
  DEFAULT_COALESCE_WINDOW_MS,
  type CoalescedCallbackOptions,
} from '@/lib/coalesced_user_action';

export function useCoalescedCallback(
  callback: () => void,
  options: CoalescedCallbackOptions = {},
): () => void {
  const {
    windowMs = DEFAULT_COALESCE_WINDOW_MS,
    inFlightGuard = false,
    inFlight = false,
  } = options;
  const lastFiredAtRef = useRef(0);

  return useCallback(() => {
    const verdict = coalesceUserAction({
      lastFiredAtMs: lastFiredAtRef.current,
      nowMs: Date.now(),
      windowMs,
      inFlight,
      inFlightGuard,
    });
    if (verdict !== 'allow') {
      return;
    }
    lastFiredAtRef.current = Date.now();
    callback();
  }, [callback, inFlight, inFlightGuard, windowMs]);
}
