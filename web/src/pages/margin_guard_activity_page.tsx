import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { listMarginGuardActivity, removeMarginGuardOverride } from '@/api/margin_guard_api';
import { MarginGuardActivityDirectory } from '@/domains/automation/margin_guard_activity_directory';
import { useCampaignScope } from '@/hooks/use_campaign_scope';
import { useResource } from '@/hooks/use_resource';

export function MarginGuardActivityPage() {
  const {
    appliedCampaignId,
    draftCampaignId,
    setDraftCampaignId,
    applyCampaignScope,
  } = useCampaignScope();

  const [reloadKey, setReloadKey] = useState(0);
  const [draftPlacementId, setDraftPlacementId] = useState('');
  const [removing, setRemoving] = useState(false);
  const [removeSuccess, setRemoveSuccess] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();

  const shouldFetch = Boolean(appliedCampaignId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve([]);
      }
      return listMarginGuardActivity({ campaign_id: appliedCampaignId }, signal);
    },
    [appliedCampaignId, reloadKey, shouldFetch],
  );

  const items = data ?? [];

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
      setReloadKey((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setRemoving(false);
    }
  }, [appliedCampaignId, draftPlacementId]);

  return (
    <MarginGuardActivityDirectory
      items={items}
      appliedCampaignId={appliedCampaignId}
      draftCampaignId={draftCampaignId}
      draftPlacementId={draftPlacementId}
      fetching={fetching}
      removing={removing}
      removeSuccess={removeSuccess}
      error={error}
      actionError={actionError}
      hasSnapshot={!shouldFetch || data != null}
      onDraftCampaignIdChange={setDraftCampaignId}
      onApplyCampaignScope={applyCampaignScope}
      onDraftPlacementIdChange={setDraftPlacementId}
      onRemoveOverride={() => {
        void onRemoveOverride();
      }}
    />
  );
}
