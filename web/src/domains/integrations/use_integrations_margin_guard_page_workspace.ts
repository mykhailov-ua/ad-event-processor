import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { isAbortError } from '@/api/client';
import {
  createMarginGuardPolicy,
  listMarginGuardActivity,
  listMarginGuardPolicies,
  removeMarginGuardOverride,
} from '@/api/margin_guard_api';
import { useCampaignScope } from '@/hooks/use_campaign_scope';
import { useSession } from '@/hooks/use_session';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import {
  requireInteger,
  requireNonEmpty,
  toastValidationError,
  type AdminValidationError,
} from '@/lib/admin_validation_error';
import { mutationError } from '@/lib/mutation_audit';
import { useResource } from '@/api/use_resource';

export function useIntegrationsMarginGuardPageWorkspace() {
  const { user } = useSession();
  const canWrite = user?.permissions?.includes('campaigns:write') ?? false;
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { appliedCampaignId, draftCampaignId, setDraftCampaignId, applyCampaignScope } =
    useCampaignScope();

  const shouldFetch = Boolean(appliedCampaignId);

  const policiesResource = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listMarginGuardPolicies({ campaign_id: appliedCampaignId }, signal);
    },
    [appliedCampaignId, shouldFetch, refreshToken]
  );

  const activityResource = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listMarginGuardActivity({ campaign_id: appliedCampaignId }, signal);
    },
    [appliedCampaignId, shouldFetch, refreshToken]
  );

  const [draftName, setDraftName] = useState('Default margin guard');
  const [draftMinClicks, setDraftMinClicks] = useState('100');
  const [draftRoiFloorPct, setDraftRoiFloorPct] = useState('0');
  const [draftZeroConvStreak, setDraftZeroConvStreak] = useState('3');
  const [draftCostOverRevenueBps, setDraftCostOverRevenueBps] = useState('1000');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createValidationError, setCreateValidationError] = useState<
    AdminValidationError | undefined
  >();

  const [overridePlacementId, setOverridePlacementId] = useState('');
  const [removingOverride, setRemovingOverride] = useState(false);
  const [overrideError, setOverrideError] = useState<Error | undefined>();

  const listBusy = creating || removingOverride;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const onCreatePolicy = useCallback(async () => {
    if (!canWrite || creating) {
      return;
    }
    const campaignCheck = requireNonEmpty(appliedCampaignId, 'Campaign ID', 'campaign_id');
    if (!campaignCheck.ok) {
      setCreateValidationError(campaignCheck.error);
      toastValidationError(campaignCheck.error);
      return;
    }
    const nameCheck = requireNonEmpty(draftName, 'Policy name', 'name');
    if (!nameCheck.ok) {
      setCreateValidationError(nameCheck.error);
      toastValidationError(nameCheck.error);
      return;
    }
    const minClicksCheck = requireInteger(draftMinClicks, 'Min clicks', {
      min: 1,
      field: 'min_clicks',
    });
    if (!minClicksCheck.ok) {
      setCreateValidationError(minClicksCheck.error);
      toastValidationError(minClicksCheck.error);
      return;
    }
    const roiFloorCheck = requireInteger(draftRoiFloorPct, 'ROI floor %', {
      field: 'roi_floor_pct',
    });
    if (!roiFloorCheck.ok) {
      setCreateValidationError(roiFloorCheck.error);
      toastValidationError(roiFloorCheck.error);
      return;
    }
    const zeroConvCheck = requireInteger(draftZeroConvStreak, 'Zero conversion streak', {
      min: 0,
      field: 'zero_conv_streak',
    });
    if (!zeroConvCheck.ok) {
      setCreateValidationError(zeroConvCheck.error);
      toastValidationError(zeroConvCheck.error);
      return;
    }
    const costOverCheck = requireInteger(draftCostOverRevenueBps, 'Cost over revenue bps', {
      min: 0,
      field: 'cost_over_revenue_threshold_bps',
    });
    if (!costOverCheck.ok) {
      setCreateValidationError(costOverCheck.error);
      toastValidationError(costOverCheck.error);
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateValidationError(undefined);
    try {
      await createMarginGuardPolicy({
        campaign_id: campaignCheck.value,
        name: nameCheck.value,
        min_clicks: Math.round(minClicksCheck.value),
        roi_floor_pct: Number(roiFloorCheck.value),
        zero_conv_streak: Math.round(zeroConvCheck.value),
        cost_over_revenue_threshold_bps: Math.round(costOverCheck.value),
        is_active: true,
      });
      toast.success('Margin guard policy created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setCreateError(mutationError(err));
    } finally {
      setCreating(false);
    }
  }, [
    appliedCampaignId,
    bumpRefreshCoalesced,
    canWrite,
    creating,
    draftCostOverRevenueBps,
    draftMinClicks,
    draftName,
    draftRoiFloorPct,
    draftZeroConvStreak,
  ]);

  const onRemoveOverride = useCallback(async () => {
    if (!canWrite || removingOverride) {
      return;
    }
    const campaignCheck = requireNonEmpty(appliedCampaignId, 'Campaign ID', 'campaign_id');
    if (!campaignCheck.ok) {
      toastValidationError(campaignCheck.error);
      return;
    }
    const placementCheck = requireNonEmpty(overridePlacementId, 'Placement ID', 'placement_id');
    if (!placementCheck.ok) {
      toastValidationError(placementCheck.error);
      return;
    }
    setRemovingOverride(true);
    setOverrideError(undefined);
    try {
      await removeMarginGuardOverride({
        campaign_id: campaignCheck.value,
        placement_id: placementCheck.value,
      });
      toast.success('Placement override removed');
      setOverridePlacementId('');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setOverrideError(mutationError(err));
    } finally {
      setRemovingOverride(false);
    }
  }, [appliedCampaignId, bumpRefreshCoalesced, canWrite, overridePlacementId, removingOverride]);

  const fetching = policiesResource.fetching || activityResource.fetching;
  const error = policiesResource.error ?? activityResource.error;
  const hasSnapshot =
    (policiesResource.data != null || !shouldFetch) &&
    (activityResource.data != null || !shouldFetch);

  return {
    canWrite,
    appliedCampaignId,
    draftCampaignId,
    onDraftCampaignIdChange: setDraftCampaignId,
    onApplyCampaignScope: applyCampaignScope,
    policies: policiesResource.data ?? [],
    activity: activityResource.data ?? [],
    fetching,
    error,
    hasSnapshot,
    draftName,
    onDraftNameChange: setDraftName,
    draftMinClicks,
    onDraftMinClicksChange: setDraftMinClicks,
    draftRoiFloorPct,
    onDraftRoiFloorPctChange: setDraftRoiFloorPct,
    draftZeroConvStreak,
    onDraftZeroConvStreakChange: setDraftZeroConvStreak,
    draftCostOverRevenueBps,
    onDraftCostOverRevenueBpsChange: setDraftCostOverRevenueBps,
    creating,
    createError,
    createValidationError,
    onCreatePolicy,
    overridePlacementId,
    onOverridePlacementIdChange: setOverridePlacementId,
    removingOverride,
    overrideError,
    onRemoveOverride,
  };
}
