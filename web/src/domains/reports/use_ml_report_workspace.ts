import { useCallback, useEffect, useMemo, useState } from 'react';

import type { DataFreshness } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { defaultReportRange } from '@/lib/report_paths';
import type { MlReportFetcher } from '@/domains/reports/ml_report_meta';

export type MlReportFetchResult<Row> = {
  rows: Row[];
  freshness?: DataFreshness;
  next_cursor?: string;
};

export function useMlReportWorkspace<Row>(fetchReport: MlReportFetcher<Row>) {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedLimit = parseListLimit(searchParams.get('limit'), 1000);
  const appliedOffset = parseListOffset(searchParams.get('offset'));

  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));

  useEffect(() => {
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
  }, [appliedFrom, appliedTo]);

  const shouldFetch = Boolean(appliedFrom && appliedTo);

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return fetchReport(
        {
          from: appliedFrom,
          to: appliedTo,
          limit: appliedLimit,
          offset: appliedOffset,
        },
        signal
      );
    },
    [appliedFrom, appliedLimit, appliedOffset, appliedTo, fetchReport, shouldFetch]
  );

  const rows = data?.rows ?? [];
  const canGoPrev = appliedOffset > 0;
  const canGoNext = Boolean(data?.next_cursor) || rows.length >= appliedLimit;

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
      next.set('limit', String(appliedLimit));
      next.set('offset', '0');
      replaceSearchParams(next);
    },
    [appliedLimit, draftFrom, draftTo, replaceSearchParams]
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('offset', String(Math.max(0, nextOffset)));
      replaceSearchParams(next);
    },
    [replaceSearchParams, searchParams]
  );

  return {
    rows,
    freshness: data?.freshness,
    nextCursor: data?.next_cursor,
    draftFrom,
    draftTo,
    fetching: fetching || listQueryPending,
    listRevalidating,
    error,
    hasSnapshot: data != null || !shouldFetch,
    rangeReady: shouldFetch,
    canGoPrev,
    canGoNext,
    limit: appliedLimit,
    offset: appliedOffset,
    rangeLabel:
      rows.length > 0
        ? `${appliedOffset + 1} - ${appliedOffset + rows.length}${data?.next_cursor ? '+' : ''}`
        : '0 of 0',
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onApplyFilters,
    onPageChange,
  };
}
