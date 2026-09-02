import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createMarginGuardPolicy, listMarginGuardPolicies } from '@/api/margin_guard_api';
import type { MarginGuardPolicy } from '@/api/types';
import {
  MarginGuardPoliciesDirectory,
  type PolicyCreateDraft,
} from '@/domains/automation/margin_guard_policies_directory';
import { useCampaignScope } from '@/hooks/use_campaign_scope';
import { useResource } from '@/api/use_resource';

const EMPTY_CREATE_DRAFT: PolicyCreateDraft = {
  name: '',
  roi_floor_pct: '0.05',
  min_clicks: '50',
  zero_conv_streak: '3',
  cost_over_revenue_threshold_bps: '1500',
  is_active: true,
};

export function MarginGuardPoliciesPage() {
  const {
    appliedCampaignId,
    draftCampaignId,
    setDraftCampaignId,
    applyCampaignScope,
  } = useCampaignScope();

  const [reloadKey, setReloadKey] = useState(0);
  const shouldFetch = Boolean(appliedCampaignId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve([]);
      }
      return listMarginGuardPolicies({ campaign_id: appliedCampaignId }, signal);
    },
    [appliedCampaignId, reloadKey, shouldFetch],
  );

  const [createDraft, setCreateDraft] = useState<PolicyCreateDraft>(EMPTY_CREATE_DRAFT);
  const [creating, setCreating] = useState(false);
  const [createSuccess, setCreateSuccess] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();

  const items = data ?? [];

  const onCreateDraftChange = useCallback((patch: Partial<PolicyCreateDraft>) => {
    setCreateDraft((prev) => ({ ...prev, ...patch }));
  }, []);

  const onCreate = useCallback(async () => {
    const campaignId = appliedCampaignId.trim();
    const name = createDraft.name.trim();
    if (!campaignId || !name) {
      return;
    }
    const roiFloor = Number.parseFloat(createDraft.roi_floor_pct.trim());
    const minClicks = Number.parseInt(createDraft.min_clicks.trim(), 10);
    const zeroStreak = Number.parseInt(createDraft.zero_conv_streak.trim(), 10);
    const costBps = Number.parseInt(createDraft.cost_over_revenue_threshold_bps.trim(), 10);
    if (
      !Number.isFinite(roiFloor) ||
      !Number.isFinite(minClicks) ||
      !Number.isFinite(zeroStreak) ||
      !Number.isFinite(costBps)
    ) {
      setActionError(new Error('Numeric policy fields must be valid numbers'));
      return;
    }

    setCreating(true);
    setActionError(undefined);
    setCreateSuccess(false);
    try {
      const body: MarginGuardPolicy = {
        campaign_id: campaignId,
        name,
        roi_floor_pct: roiFloor,
        min_clicks: minClicks,
        zero_conv_streak: zeroStreak,
        cost_over_revenue_threshold_bps: costBps,
        is_active: createDraft.is_active,
      };
      await createMarginGuardPolicy(body);
      setCreateSuccess(true);
      setCreateDraft(EMPTY_CREATE_DRAFT);
      toast.success('Margin guard policy created');
      setReloadKey((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [appliedCampaignId, createDraft]);

  return (
    <MarginGuardPoliciesDirectory
      items={items}
      appliedCampaignId={appliedCampaignId}
      draftCampaignId={draftCampaignId}
      createDraft={createDraft}
      fetching={fetching}
      creating={creating}
      error={error}
      actionError={actionError}
      createSuccess={createSuccess}
      hasSnapshot={!shouldFetch || data != null}
      onDraftCampaignIdChange={setDraftCampaignId}
      onApplyCampaignScope={applyCampaignScope}
      onCreateDraftChange={onCreateDraftChange}
      onCreate={() => {
        void onCreate();
      }}
    />
  );
}
