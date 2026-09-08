import { useEffect, useState } from 'react';

import { displayRelativeTimestamp } from '@/lib/display';

/** Recomputes displayRelativeTimestamp every second while mounted. */
export function useLiveRelativeTimestamp(iso?: string | null, display?: string | null): string {
  const [, setTick] = useState(0);

  useEffect(() => {
    if (!iso?.trim()) {
      return;
    }
    const timerId = window.setInterval(() => {
      setTick((value) => value + 1);
    }, 1000);
    return () => {
      window.clearInterval(timerId);
    };
  }, [iso]);

  return displayRelativeTimestamp(iso, display);
}
