import { useCallback, useMemo } from 'react';

import { listDisputes } from '@/api/platform_api';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';

function buildListQuery(params: URLSearchParams) {
  return {
    limit: parseListLimit(params.get('limit')),
    offset: parseListOffset(params.get('offset')),
  };
}

export function useDisputesPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const query = useMemo(() => buildListQuery(searchParams), [searchParams]);

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource((signal) => listDisputes(query, signal), [query.limit, query.offset]);

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('offset', String(nextOffset));
      replaceSearchParams(next);
    },
    [replaceSearchParams, searchParams]
  );

  const onLimitChange = useCallback(
    (limit: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(limit));
      next.set('offset', '0');
      replaceSearchParams(next);
    },
    [replaceSearchParams, searchParams]
  );

  const items = data?.disputes;
  const total = data?.total ?? 0;

  return {
    items,
    total,
    limit: query.limit ?? DEFAULT_LIST_LIMIT,
    offset: query.offset ?? 0,
    fetching: fetching || listQueryPending,
    listRevalidating,
    error,
    hasSnapshot: data != null,
    onPageChange,
    onLimitChange,
  };
}

export type DisputesPageWorkspace = ReturnType<typeof useDisputesPageWorkspace>;
