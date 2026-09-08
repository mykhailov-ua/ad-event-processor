import { useCallback, useEffect, useState } from 'react';

import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';

export function useCampaignScope() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();

  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);

  useEffect(() => {
    setDraftCampaignId(appliedCampaignId);
  }, [appliedCampaignId]);

  const applyCampaignScope = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    const trimmed = draftCampaignId.trim();
    if (trimmed) {
      next.set('campaign_id', trimmed);
    } else {
      next.delete('campaign_id');
    }
    replaceSearchParams(next);
  }, [draftCampaignId, replaceSearchParams, searchParams]);

  return {
    appliedCampaignId,
    draftCampaignId,
    setDraftCampaignId,
    applyCampaignScope,
    listQueryPending,
  };
}
