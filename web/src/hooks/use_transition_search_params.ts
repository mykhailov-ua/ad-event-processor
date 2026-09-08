import { useCallback, useTransition } from 'react';
import { useSearchParams } from 'react-router-dom';

export type TransitionSearchParams = {
  /** True while a replace navigation is scheduled (sort/filter/pagination). */
  isPending: boolean;
  /** setSearchParams(next, { replace: true }) inside startTransition. */
  replaceSearchParams: (next: URLSearchParams) => void;
};

/**
 * URL search params for server-paginated directories. Replace navigations use
 * startTransition so tables and filter chrome stay interactive during refetch.
 */
export function useTransitionSearchParams(): [URLSearchParams, TransitionSearchParams] {
  const [searchParams, setSearchParams] = useSearchParams();
  const [isPending, startTransition] = useTransition();

  const replaceSearchParams = useCallback(
    (next: URLSearchParams) => {
      startTransition(() => {
        setSearchParams(next, { replace: true });
      });
    },
    [setSearchParams]
  );

  return [searchParams, { isPending, replaceSearchParams }];
}
