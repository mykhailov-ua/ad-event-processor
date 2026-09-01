import { type DependencyList, useEffect, useRef, useState } from 'react';

import { isAbortError } from '../api/client.js';

export type UseResourceState<T> = {
  data: T | undefined;
  error: Error | undefined;
  fetching: boolean;
};

/**
 * Cold-path fetch hook: aborts in-flight requests on dep change/unmount,
 * replaces data snapshot on success, keeps stale rows during revalidate.
 * `fetching` is true only until the first successful or failed load; background
 * refetches do not toggle it (avoids control-bar flicker on sort/filter).
 */
export function useResource<T>(
  fetcher: (signal: AbortSignal) => Promise<T>,
  deps: DependencyList,
): UseResourceState<T> {
  const [data, setData] = useState<T | undefined>(undefined);
  const [error, setError] = useState<Error | undefined>(undefined);
  const [fetching, setFetching] = useState(true);
  const generationRef = useRef(0);
  const snapshotRef = useRef(false);

  useEffect(() => {
    const ctrl = new AbortController();
    const generation = ++generationRef.current;

    if (!snapshotRef.current) {
      setFetching(true);
    }

    void fetcher(ctrl.signal)
      .then((next) => {
        if (generation !== generationRef.current) {
          return;
        }
        snapshotRef.current = true;
        setData(next);
        setError(undefined);
      })
      .catch((err: unknown) => {
        if (generation !== generationRef.current) {
          return;
        }
        if (isAbortError(err)) {
          return;
        }
        snapshotRef.current = true;
        setError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        if (generation !== generationRef.current) {
          return;
        }
        setFetching(false);
      });

    return () => {
      ctrl.abort();
    };
  }, deps);

  return { data, error, fetching };
}
