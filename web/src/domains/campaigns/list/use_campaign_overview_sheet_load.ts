// overview sheet: stats+margins when sheet open; session stats cache; skips fetch when campaign id not UUID-like.
import { useEffect, useMemo, useState } from 'react';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import { getCampaignMargin, getCampaignStats } from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type { CampaignMargin, CampaignStats, CampaignStatsQuery } from '@/api/types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import {
  buildCampaignStatsCacheKey,
  campaignStatsFromListMetrics,
  readCachedCampaignStats,
  writeCachedCampaignStats,
} from '@/domains/campaigns/list/campaign_list_stats_cache';
import { isUuidLike } from '@/lib/customer_label';

type UseCampaignOverviewSheetLoadArgs = {
  open: boolean;
  campaign: CampaignWithMoneyDisplay | null;
  listMargin?: CampaignMargin;
  listMetrics?: CampaignListMetrics;
  statsCacheRevision: string;
  statsQuery?: CampaignStatsQuery;
};

export function useCampaignOverviewSheetLoad({
  open,
  campaign,
  listMargin,
  listMetrics,
  statsCacheRevision,
  statsQuery,
}: UseCampaignOverviewSheetLoadArgs) {
  const [stats, setStats] = useState<CampaignStats | undefined>();
  const [margin, setMargin] = useState<CampaignMargin | undefined>();
  const [statsError, setStatsError] = useState<Error | undefined>();
  const [marginError, setMarginError] = useState<Error | undefined>();
  const [loading, setLoading] = useState(false);

  const resolvedStatsQuery = useMemo(
    () => statsQuery ?? {},
    [statsQuery?.from, statsQuery?.granularity, statsQuery?.to]
  );
  const cacheKey = useMemo(
    () =>
      campaign && isUuidLike(campaign.id)
        ? buildCampaignStatsCacheKey(campaign.id, resolvedStatsQuery, statsCacheRevision)
        : '',
    [campaign, resolvedStatsQuery, statsCacheRevision]
  );

  useEffect(() => {
    if (!open || !campaign || !isUuidLike(campaign.id)) {
      return;
    }

    const cached = readCachedCampaignStats(cacheKey);
    if (cached) {
      setStats(cached);
      setStatsError(undefined);
      setMargin(listMargin);
      setMarginError(undefined);
      setLoading(false);
      return;
    }

    const controller = new AbortController();
    const seededStats = listMetrics
      ? campaignStatsFromListMetrics(campaign.id, listMetrics, resolvedStatsQuery)
      : undefined;
    setStats(seededStats);
    setMargin(listMargin);
    setStatsError(undefined);
    setMarginError(undefined);
    setLoading(true);

    const statsPromise = getCampaignStats(campaign.id, resolvedStatsQuery, controller.signal);
    const marginPromise = listMargin
      ? Promise.resolve(listMargin)
      : getCampaignMargin(campaign.id, controller.signal);

    void Promise.allSettled([statsPromise, marginPromise])
      .then(([statsResult, marginResult]) => {
        if (controller.signal.aborted) {
          return;
        }

        if (statsResult.status === 'fulfilled') {
          writeCachedCampaignStats(cacheKey, statsResult.value);
          setStats(statsResult.value);
          setStatsError(undefined);
        } else {
          const reason = statsResult.reason;
          setStats(seededStats);
          setStatsError(reason instanceof Error ? reason : new Error(String(reason)));
        }

        if (marginResult.status === 'fulfilled') {
          setMargin(marginResult.value);
          setMarginError(undefined);
        } else {
          const reason = marginResult.reason;
          setMargin(listMargin);
          if (!(reason instanceof ApiError && reason.status === 501)) {
            setMarginError(reason instanceof Error ? reason : new Error(String(reason)));
          }
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setLoading(false);
        }
      });

    return () => {
      controller.abort();
    };
  }, [cacheKey, campaign, listMargin, listMetrics, open, resolvedStatsQuery]);

  const reset = () => {
    setStats(undefined);
    setMargin(undefined);
    setStatsError(undefined);
    setMarginError(undefined);
    setLoading(false);
  };

  return {
    stats,
    margin,
    statsError,
    marginError,
    loading,
    reset,
  };
}
