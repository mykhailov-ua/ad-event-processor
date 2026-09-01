import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

export function useCampaignScope() {
  const [searchParams, setSearchParams] = useSearchParams();

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
    setSearchParams(next, { replace: true });
  }, [draftCampaignId, searchParams, setSearchParams]);

  return {
    appliedCampaignId,
    draftCampaignId,
    setDraftCampaignId,
    applyCampaignScope,
  };
}
