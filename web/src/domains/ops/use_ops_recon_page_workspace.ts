// recon runs directory: service filter in URL; draft resets on Apply only.
import { useCallback, useMemo, useState } from 'react';

import { listReconRuns } from '@/api/ops_api';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';

export function useOpsReconPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const appliedService = searchParams.get('service') ?? '';
  const limit = parseListLimit(searchParams.get('limit'));
  const offset = parseListOffset(searchParams.get('offset'));
  const [draftService, setDraftService] = useState(appliedService);

  const query = useMemo(
    () => ({
      service: appliedService || undefined,
      limit,
      offset,
    }),
    [appliedService, limit, offset]
  );

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    (signal) => listReconRuns(query, signal),
    [query.limit, query.offset, query.service]
  );

  const onApplyFilters = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    const trimmed = draftService.trim();
    if (trimmed) {
      next.set('service', trimmed);
    } else {
      next.delete('service');
    }
    next.set('limit', String(limit));
    next.set('offset', '0');
    replaceSearchParams(next);
  }, [draftService, limit, replaceSearchParams, searchParams]);

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('offset', String(Math.max(0, nextOffset)));
      next.set('limit', String(limit ?? DEFAULT_LIST_LIMIT));
      replaceSearchParams(next);
    },
    [limit, replaceSearchParams, searchParams]
  );

  return {
    items: data,
    draftService,
    limit: limit ?? DEFAULT_LIST_LIMIT,
    offset,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error,
    hasSnapshot: data != null,
    onDraftServiceChange: setDraftService,
    onApplyFilters,
    onPageChange,
  };
}
