// metrics popover: seeds from list batch metrics; GET /stats on open; pendingRefreshRef bypasses session cache on manual refresh.
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { getCampaignStats } from '@/api/campaigns_api';
import { toError } from '@/lib/admin_error';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignStats, CampaignStatsQuery } from '@/api/types';
import {
  buildCampaignStatsCacheKey,
  campaignStatsFromListMetrics,
  readCachedCampaignStats,
  writeCachedCampaignStats,
} from '@/domains/campaigns/list/campaign_list_stats_cache';

type UseCampaignMetricsPopoverLoadArgs = {
  open: boolean;
  campaignId: string;
  listMetrics?: CampaignListMetrics;
  statsCacheRevision: string;
  statsQuery?: CampaignStatsQuery;
};

export function useCampaignMetricsPopoverLoad({
  open,
  campaignId,
  listMetrics,
  statsCacheRevision,
  statsQuery,
}: UseCampaignMetricsPopoverLoadArgs) {
  const [stats, setStats] = useState<CampaignStats | undefined>();
  const [error, setError] = useState<Error | undefined>();
  const [loading, setLoading] = useState(false);
  const [refreshNonce, setRefreshNonce] = useState(0);
  const pendingRefreshRef = useRef(false);

  const resolvedStatsQuery = useMemo(
    () => statsQuery ?? {},
    [statsQuery?.from, statsQuery?.granularity, statsQuery?.to]
  );
  const cacheKey = useMemo(
    () => buildCampaignStatsCacheKey(campaignId, resolvedStatsQuery, statsCacheRevision),
    [campaignId, resolvedStatsQuery, statsCacheRevision]
  );

  useEffect(() => {
    if (!open) {
      return;
    }
    const forceRefresh = pendingRefreshRef.current;
    pendingRefreshRef.current = false;
    if (!forceRefresh) {
      const cached = readCachedCampaignStats(cacheKey);
      if (cached) {
        setStats(cached);
        setError(undefined);
        setLoading(false);
        return;
      }
      if (listMetrics) {
        const seeded = campaignStatsFromListMetrics(campaignId, listMetrics, resolvedStatsQuery);
        writeCachedCampaignStats(cacheKey, seeded);
        setStats(seeded);
        setError(undefined);
        setLoading(false);
        return;
      }
    }

    const controller = new AbortController();
    const seeded = listMetrics
      ? campaignStatsFromListMetrics(campaignId, listMetrics, resolvedStatsQuery)
      : undefined;
    setStats(seeded);
    setLoading(true);
    setError(undefined);

    void getCampaignStats(campaignId, resolvedStatsQuery, controller.signal)
      .then((next) => {
        writeCachedCampaignStats(cacheKey, next);
        setStats(next);
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) {
          return;
        }
        setError(toError(err));
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });

    return () => {
      controller.abort();
    };
  }, [cacheKey, campaignId, listMetrics, open, refreshNonce, resolvedStatsQuery]);

  const onRefresh = useCallback(() => {
    pendingRefreshRef.current = true;
    setRefreshNonce((value) => value + 1);
  }, []);

  return {
    stats,
    error,
    loading,
    onRefresh,
  };
}
