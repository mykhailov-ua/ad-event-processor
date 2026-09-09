import { useEffect, useState } from 'react';
import { isValid, parseISO } from 'date-fns';

import { displayRelativeTimestamp } from '@/lib/display';

const LIVE_TIMESTAMP_TICK_MS = 30_000;

function liveRelativeTimestampTickMs(iso: string): number {
  const date = parseISO(iso);
  if (!isValid(date)) {
    return LIVE_TIMESTAMP_TICK_MS;
  }
  const ageMs = Date.now() - date.getTime();
  if (ageMs < 60_000) {
    return 15_000;
  }
  if (ageMs < 3_600_000) {
    return 60_000;
  }
  return LIVE_TIMESTAMP_TICK_MS;
}

/** Recomputes displayRelativeTimestamp on an adaptive interval while mounted and tab visible. */
export function useLiveRelativeTimestamp(iso?: string | null, display?: string | null): string {
  const [, setTick] = useState(0);

  useEffect(() => {
    if (!iso?.trim()) {
      return;
    }

    let timerId: number | undefined;

    const scheduleTick = () => {
      if (document.visibilityState === 'hidden') {
        return;
      }
      setTick((value) => value + 1);
      timerId = window.setTimeout(scheduleTick, liveRelativeTimestampTickMs(iso));
    };

    const onVisibilityChange = () => {
      if (document.visibilityState !== 'visible') {
        if (timerId !== undefined) {
          window.clearTimeout(timerId);
          timerId = undefined;
        }
        return;
      }
      setTick((value) => value + 1);
      if (timerId !== undefined) {
        window.clearTimeout(timerId);
      }
      timerId = window.setTimeout(scheduleTick, liveRelativeTimestampTickMs(iso));
    };

    timerId = window.setTimeout(scheduleTick, liveRelativeTimestampTickMs(iso));
    document.addEventListener('visibilitychange', onVisibilityChange);

    return () => {
      if (timerId !== undefined) {
        window.clearTimeout(timerId);
      }
      document.removeEventListener('visibilitychange', onVisibilityChange);
    };
  }, [iso]);

  return displayRelativeTimestamp(iso, display);
}
