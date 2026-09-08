import { type DependencyList, useEffect, useRef, useState } from 'react';

import { isAbortError } from './client.js';

export type UseResourceState<T> = {
  data: T | undefined;
  error: Error | undefined;
  fetching: boolean;
  /** True while refetching after the first snapshot (sort/filter/page). */
  revalidating: boolean;
};

/**
 * Cold-path fetch hook: aborts in-flight requests on dep change/unmount,
 * replaces data snapshot on success, keeps stale rows during revalidate.
 * `fetching` is true only until the first successful or failed load; background
 * refetches do not toggle it (avoids control-bar flicker on sort/filter).
 *
 * generationRef ignores stale responses when deps change faster than the network.
 */
export function useResource<T>(
  fetcher: (signal: AbortSignal) => Promise<T>,
  deps: DependencyList
): UseResourceState<T> {
  const [data, setData] = useState<T | undefined>(undefined);
  const [error, setError] = useState<Error | undefined>(undefined);
  const [fetching, setFetching] = useState(true);
  const [revalidating, setRevalidating] = useState(false);
  const generationRef = useRef(0);
  const snapshotRef = useRef(false);

  useEffect(() => {
    const ctrl = new AbortController();
    const generation = ++generationRef.current;
    const hadSnapshot = snapshotRef.current;

    if (!hadSnapshot) {
      setFetching(true);
      setRevalidating(false);
    } else {
      setRevalidating(true);
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
        setRevalidating(false);
      });

    return () => {
      ctrl.abort();
    };
  }, deps);

  return { data, error, fetching, revalidating };
}
