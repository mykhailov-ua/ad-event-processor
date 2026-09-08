// edge parity report: URL date range drives GET /api/v1/reports/edge-parity (max 15 minute window).
import { useCallback, useEffect, useMemo, useState } from 'react';

import { getEdgeParityReport } from '@/api/reports_api';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';

function defaultEdgeParityRange(): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to.getTime() - 15 * 60 * 1000);
  return { from: from.toISOString(), to: to.toISOString() };
}

export function useEdgeParityPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const defaultRange = useMemo(() => defaultEdgeParityRange(), []);

  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;

  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));

  useEffect(() => {
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
  }, [appliedFrom, appliedTo]);

  const shouldFetch = Boolean(appliedFrom && appliedTo);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getEdgeParityReport({ from: appliedFrom, to: appliedTo }, signal);
    },
    [appliedFrom, appliedTo, shouldFetch]
  );

  const onApplyFilters = useCallback(
    (event?: { preventDefault?: () => void }) => {
      event?.preventDefault?.();
      const next = new URLSearchParams();
      const from = fromDatetimeLocalValue(draftFrom);
      const to = fromDatetimeLocalValue(draftTo);
      if (from) {
        next.set('from', from);
      }
      if (to) {
        next.set('to', to);
      }
      replaceSearchParams(next);
    },
    [draftFrom, draftTo, replaceSearchParams]
  );

  return {
    data,
    error,
    fetching: fetching || listQueryPending,
    hasSnapshot: data != null,
    shouldFetch,
    draftFrom,
    draftTo,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onApplyFilters,
  };
}
