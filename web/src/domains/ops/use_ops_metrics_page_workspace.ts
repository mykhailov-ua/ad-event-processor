// ops metrics: GET dashboard by URL range + optional SSE liveSummary overlay (regime F leaf).
import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import { getOpsDashboardMetrics, subscribeOpsDashboardStream } from '@/api/ops_api';
import type { DashboardSummary } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh } from '@/hooks/use_coalesced_refresh_token';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';

export function useOpsMetricsPageWorkspace() {
  const location = useLocation();
  const [searchParams, { replaceSearchParams }] = useTransitionSearchParams();
  const appliedRange = searchParams.get('range') ?? '1h';
  const [draftRange, setDraftRange] = useState(appliedRange);
  const [metricsLoadToken, setMetricsLoadToken] = useState(0);
  const [pendingRange, setPendingRange] = useState(appliedRange);
  const [liveSummary, setLiveSummary] = useState<DashboardSummary | undefined>();
  const [liveEnabled, setLiveEnabled] = useState(false);
  const [streamError, setStreamError] = useState<Error | undefined>();

  useEffect(() => {
    setDraftRange(appliedRange);
    setPendingRange(appliedRange);
    setMetricsLoadToken((value) => value + 1);
  }, [appliedRange]);

  const metricsResource = useResource(
    async (signal) => {
      if (metricsLoadToken === 0) {
        return Promise.reject(new DOMException('Skipped', 'AbortError'));
      }
      const range = pendingRange.trim() || '1h';
      const result = await getOpsDashboardMetrics({ range }, signal);
      const next = new URLSearchParams(searchParams);
      next.set('range', range);
      replaceSearchParams(next);
      return result;
    },
    [metricsLoadToken, pendingRange, replaceSearchParams, searchParams]
  );

  useEffect(() => {
    if (!liveEnabled) {
      return undefined;
    }
    setStreamError(undefined);
    const close = subscribeOpsDashboardStream(
      (summary) => {
        setLiveSummary(summary);
      },
      (error) => {
        setStreamError(error);
        setLiveEnabled(false);
      }
    );
    return () => {
      close();
      setLiveSummary(undefined);
    };
  }, [liveEnabled, location.pathname]);

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
    error: metricsResource.error,
    streamError,
    hasSnapshot,
    onDraftRangeChange: setDraftRange,
    onLoad,
    onLiveEnabledChange: setLiveEnabled,
  };
}
