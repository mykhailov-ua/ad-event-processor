import { useCallback, useEffect, useMemo, useState } from 'react';

import { getCampaignStats } from '@/api/campaigns_api';
import type { CampaignStats } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { defaultReportRange } from '@/lib/report_paths';

export type CampaignStatsGranularity = 'hour' | 'day';

export function useCampaignStatsReportWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedGranularity: CampaignStatsGranularity =
    searchParams.get('granularity') === 'day' ? 'day' : 'hour';

  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [draftGranularity, setDraftGranularity] =
    useState<CampaignStatsGranularity>(appliedGranularity);

  useEffect(() => {
    setDraftCampaignId(appliedCampaignId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
    setDraftGranularity(appliedGranularity);
  }, [appliedCampaignId, appliedFrom, appliedGranularity, appliedTo]);

  const shouldFetch = Boolean(appliedCampaignId.trim());

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
      return getCampaignStats(
        appliedCampaignId,
        {
          from: appliedFrom,
          to: appliedTo,
          granularity: appliedGranularity,
        },
        signal
      );
    },
    [appliedCampaignId, appliedFrom, appliedGranularity, appliedTo, shouldFetch]
  );

  const onApplyFilters = useCallback(
    (event?: { preventDefault?: () => void }) => {
      event?.preventDefault?.();
      const next = new URLSearchParams();
      const campaignId = draftCampaignId.trim();
      if (campaignId) {
        next.set('campaign_id', campaignId);
      }
      const from = fromDatetimeLocalValue(draftFrom);
      const to = fromDatetimeLocalValue(draftTo);
      if (from) {
        next.set('from', from);
      }
      if (to) {
        next.set('to', to);
      }
      if (draftGranularity === 'day') {
        next.set('granularity', 'day');
      }
      replaceSearchParams(next);
    },
    [draftCampaignId, draftFrom, draftGranularity, draftTo, replaceSearchParams]
  );

  return {
    stats: data as CampaignStats | undefined,
    draftCampaignId,
    draftFrom,
    draftTo,
    draftGranularity,
    fetching: fetching || listQueryPending,
    listRevalidating,
    error,
    hasSnapshot: data != null || !shouldFetch,
    campaignReady: shouldFetch,
    onDraftCampaignIdChange: setDraftCampaignId,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onDraftGranularityChange: setDraftGranularity,
    onApplyFilters,
  };
}
