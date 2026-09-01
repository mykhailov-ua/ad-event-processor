import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { listReconRuns } from '@/api/ops_api';
import { OpsRecon } from '@/domains/ops/ops_recon';
import { useResource } from '@/hooks/use_resource';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';

export function OpsReconPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const appliedService = searchParams.get('service') ?? '';
  const limit = parseListLimit(searchParams.get('limit'));
  const offset = parseListOffset(searchParams.get('offset'));
  const [draftService, setDraftService] = useState(appliedService);

  useEffect(() => {
    setDraftService(appliedService);
  }, [appliedService]);

  const query = useMemo(
    () => ({
      service: appliedService || undefined,
      limit,
      offset,
    }),
    [appliedService, limit, offset],
  );

  const { data, error, fetching } = useResource(
    (signal) => listReconRuns(query, signal),
    [query.limit, query.offset, query.service],
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
    setSearchParams(next, { replace: true });
  }, [draftService, limit, searchParams, setSearchParams]);

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('offset', String(Math.max(0, nextOffset)));
      next.set('limit', String(limit ?? DEFAULT_LIST_LIMIT));
      setSearchParams(next, { replace: true });
    },
    [limit, searchParams, setSearchParams],
  );

  return (
    <OpsRecon
      items={data ?? []}
      draftService={draftService}
      limit={limit ?? DEFAULT_LIST_LIMIT}
      offset={offset}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      onDraftServiceChange={setDraftService}
      onApplyFilters={onApplyFilters}
      onPageChange={onPageChange}
    />
  );
}
