// L3 margin guard activity: campaign-scoped overrides list + remove placement override.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { listMarginGuardActivity, removeMarginGuardOverride } from '@/api/margin_guard_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useCampaignScope } from '@/hooks/use_campaign_scope';
import { useResource } from '@/api/use_resource';

export function useMarginGuardActivityPageWorkspace() {
  const { appliedCampaignId, draftCampaignId, setDraftCampaignId, applyCampaignScope } =
    useCampaignScope();

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [draftPlacementId, setDraftPlacementId] = useState('');
  const [removing, setRemoving] = useState(false);
  const [removeSuccess, setRemoveSuccess] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();

  const shouldFetch = Boolean(appliedCampaignId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listMarginGuardActivity({ campaign_id: appliedCampaignId }, signal);
    },
    [appliedCampaignId, refreshToken, shouldFetch]
  );

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching || removing);

  const onRemoveOverride = useCallback(async () => {
    const campaignId = appliedCampaignId.trim();
    const placementId = draftPlacementId.trim();
    if (!campaignId || !placementId) {
      return;
    }
    setRemoving(true);
    setActionError(undefined);
    setRemoveSuccess(false);
    try {
      await removeMarginGuardOverride({ campaign_id: campaignId, placement_id: placementId });
      setRemoveSuccess(true);
      setDraftPlacementId('');
      toast.success('Placement override cleared');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setRemoving(false);
    }
  }, [appliedCampaignId, bumpRefreshCoalesced, draftPlacementId]);

  return {
    items: data,
    appliedCampaignId,
    draftCampaignId,
    draftPlacementId,
    fetching,
    removing,
    removeSuccess,
    error,
    actionError,
    hasSnapshot: data != null,
    onDraftCampaignIdChange: setDraftCampaignId,
    onApplyCampaignScope: applyCampaignScope,
    onDraftPlacementIdChange: setDraftPlacementId,
    onRemoveOverride: () => {
      void onRemoveOverride();
    },
  };
}
