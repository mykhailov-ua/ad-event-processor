import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { getOpsDashboardMetrics, subscribeOpsDashboardStream } from '@/api/ops_api';
import type { DashboardSummary } from '@/api/types';
import { OpsMetrics } from '@/domains/ops/ops_metrics';

export function OpsMetricsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const appliedRange = searchParams.get('range') ?? '1h';
  const [draftRange, setDraftRange] = useState(appliedRange);
  const [metrics, setMetrics] = useState<Awaited<ReturnType<typeof getOpsDashboardMetrics>>>();
  const [liveSummary, setLiveSummary] = useState<DashboardSummary | undefined>();
  const [liveEnabled, setLiveEnabled] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState<Error | undefined>();
  const [hasSnapshot, setHasSnapshot] = useState(false);

  useEffect(() => {
    setDraftRange(appliedRange);
  }, [appliedRange]);

  useEffect(() => {
    if (!liveEnabled) {
      setLiveSummary(undefined);
      return undefined;
    }
    return subscribeOpsDashboardStream(
      (summary) => {
        setLiveSummary(summary);
        setHasSnapshot(true);
      },
      (streamError) => {
        setError(streamError);
        setLiveEnabled(false);
      },
    );
  }, [liveEnabled]);

  const onLoad = useCallback(async () => {
    const range = draftRange.trim() || '1h';
    setFetching(true);
    setError(undefined);
    try {
      const result = await getOpsDashboardMetrics({ range });
      setMetrics(result);
      setHasSnapshot(true);
      const next = new URLSearchParams(searchParams);
      next.set('range', range);
      setSearchParams(next, { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setFetching(false);
    }
  }, [draftRange, searchParams, setSearchParams]);

  return (
    <OpsMetrics
      metrics={metrics}
      liveSummary={liveSummary}
      liveEnabled={liveEnabled}
      draftRange={draftRange}
      fetching={fetching}
      error={error}
      hasSnapshot={hasSnapshot}
      onDraftRangeChange={setDraftRange}
      onLoad={onLoad}
      onLiveEnabledChange={setLiveEnabled}
    />
  );
}
