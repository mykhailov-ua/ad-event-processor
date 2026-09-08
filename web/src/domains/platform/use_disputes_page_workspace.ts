// L3 disputes directory: customer scope + paginated list (limit/offset in URL).
import { useCallback, useMemo } from 'react';

import { listDisputes } from '@/api/platform_api';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';

function readListQuery(searchParams: URLSearchParams) {
  return {
    limit: parseListLimit(searchParams.get('limit')),
    offset: parseListOffset(searchParams.get('offset')),
  };
}

export function useDisputesPageWorkspace() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const query = useMemo(() => readListQuery(searchParams), [searchParams]);

  const { data, error, fetching, revalidating: listRevalidating } = useResource(
    (signal) =>
      listDisputes(
        {
          customer_id: appliedCustomerId || undefined,
          limit: query.limit,
          offset: query.offset,
        },
        signal
      ),
    [appliedCustomerId, query.limit, query.offset]
  );

  const disputes = useMemo(() => data?.disputes ?? [], [data]);
  const total = data?.total ?? 0;
  const limit = query.limit ?? DEFAULT_LIST_LIMIT;
  const offset = query.offset ?? 0;

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(limit));
      next.set('offset', String(Math.max(0, nextOffset)));
      replaceSearchParams(next);
    },
    [limit, replaceSearchParams, searchParams]
  );

  const onApplyCustomerScope = useCallback(() => {
    applyCustomerScope();
    const next = new URLSearchParams(searchParams);
    next.set('offset', '0');
    replaceSearchParams(next);
  }, [applyCustomerScope, replaceSearchParams, searchParams]);

  return {
    disputes,
    total,
    limit,
    offset,
    appliedCustomerId,
    draftCustomerId,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error,
    hasSnapshot: data != null,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope,
    onPageChange,
  };
}
