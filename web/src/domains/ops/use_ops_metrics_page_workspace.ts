// L3 ops metrics: GET dashboard by URL range + optional SSE liveSummary overlay (regime F leaf).
import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { getOpsDashboardMetrics, subscribeOpsDashboardStream } from '@/api/ops_api';
import type { DashboardSummary } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh } from '@/hooks/use_coalesced_refresh_token';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useOpsMetricsPageWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const appliedRange = searchParams.get('range') ?? '1h';
  const [draftRange, setDraftRange] = useState(appliedRange);
  const [metricsLoadToken, setMetricsLoadToken] = useState(0);
  const [pendingRange, setPendingRange] = useState('');
  const [liveSummary, setLiveSummary] = useState<DashboardSummary | undefined>();
  const [liveEnabled, setLiveEnabled] = useState(false);
  const [streamError, setStreamError] = useState<Error | undefined>();

  useEffect(() => {
    setDraftRange(appliedRange);
  }, [appliedRange]);

  const metricsResource = useResource(
    async (signal) => {
      if (metricsLoadToken === 0) {
        return skipLazyFetch();
      }
      const range = pendingRange.trim() || '1h';
      const result = await getOpsDashboardMetrics({ range }, signal);
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set('range', range);
          return next;
        },
        { replace: true }
      );
      return result;
    },
    [metricsLoadToken, pendingRange, setSearchParams]
  );

  useEffect(() => {
    if (!liveEnabled) {
      setLiveSummary(undefined);
      return undefined;
    }
    setStreamError(undefined);
    return subscribeOpsDashboardStream(
      (summary) => {
        setLiveSummary(summary);
      },
      (error) => {
        setStreamError(error);
        setLiveEnabled(false);
      }
    );
  }, [liveEnabled]);

  const onLoad = useCoalescedBumpRefresh(() => {
    setPendingRange(draftRange.trim() || '1h');
    setMetricsLoadToken((value) => value + 1);
  }, metricsResource.fetching);

  const hasSnapshot = metricsResource.data != null || liveSummary != null;

  return {
    metrics: metricsResource.data,
    liveSummary,
    liveEnabled,
    draftRange,
    fetching: metricsResource.fetching,
    error: streamError ?? metricsResource.error,
    hasSnapshot,
    onDraftRangeChange: setDraftRange,
    onLoad,
    onLiveEnabledChange: setLiveEnabled,
  };
}
