// traffic optimizer: presets + rules catalog; dry-run suggestions; apply via flow PUT.
import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { isAbortError } from '@/api/client';
import {
  applyTrafficOptimizerRule,
  createTrafficOptimizerRule,
  deleteTrafficOptimizerRule,
  dryRunTrafficOptimizerRule,
  listTrafficOptimizerPresets,
  listTrafficOptimizerRules,
} from '@/api/traffic_optimizer_api';
import type {
  TrafficOptimizerDryRunResult,
  TrafficOptimizerRule,
  UpsertTrafficOptimizerRuleRequest,
} from '@/api/types';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useSession } from '@/hooks/use_session';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import {
  actionGuardError,
  requireNonEmpty,
  toastValidationError,
  type AdminValidationError,
} from '@/lib/admin_validation_error';
import { mutationError } from '@/lib/mutation_audit';
import { useResource } from '@/api/use_resource';

export function useIntegrationsTrafficOptimizerPageWorkspace() {
  const { user } = useSession();
  const canWrite = user?.permissions?.includes('campaigns:write') ?? false;
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const shouldFetchRules = Boolean(appliedCustomerId);

  const presetsResource = useResource((signal) => listTrafficOptimizerPresets(signal), []);
  const rulesResource = useResource(
    (signal) => {
      if (!shouldFetchRules) {
        return Promise.resolve(undefined);
      }
      return listTrafficOptimizerRules({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, shouldFetchRules, refreshToken]
  );

  const [draftPresetKey, setDraftPresetKey] = useState('cr_best_performer');
  const [draftFlowId, setDraftFlowId] = useState('');
  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createValidationError, setCreateValidationError] = useState<
    AdminValidationError | undefined
  >();

  const [activeRuleId, setActiveRuleId] = useState<string | undefined>();
  const [dryRunResult, setDryRunResult] = useState<TrafficOptimizerDryRunResult | undefined>();
  const [dryRunningRuleId, setDryRunningRuleId] = useState<string | undefined>();
  const [dryRunError, setDryRunError] = useState<Error | undefined>();

  const [applyingRuleId, setApplyingRuleId] = useState<string | undefined>();
  const [applyError, setApplyError] = useState<Error | undefined>();

  const [deletingRuleId, setDeletingRuleId] = useState<string | undefined>();
  const [deleteError, setDeleteError] = useState<Error | undefined>();

  const presets = presetsResource.data ?? [];
  const rules = rulesResource.data ?? [];
  const listBusy =
    creating || dryRunningRuleId != null || applyingRuleId != null || deletingRuleId != null;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const activeRule = useMemo(
    () => rules.find((rule) => rule.id === activeRuleId),
    [activeRuleId, rules]
  );

  const onCreateRule = useCallback(async () => {
    if (!canWrite || creating) {
      return;
    }
    if (!appliedCustomerId) {
      const err = actionGuardError('Apply a customer ID before creating a rule.');
      setCreateValidationError(err);
      toastValidationError(err);
      return;
    }
    const presetCheck = requireNonEmpty(draftPresetKey, 'Preset', 'preset_key');
    if (!presetCheck.ok) {
      setCreateValidationError(presetCheck.error);
      toastValidationError(presetCheck.error);
      return;
    }
    const flowCheck = requireNonEmpty(draftFlowId, 'Flow ID', 'flow_id');
    if (!flowCheck.ok) {
      setCreateValidationError(flowCheck.error);
      toastValidationError(flowCheck.error);
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateValidationError(undefined);
    try {
      const rule = await createTrafficOptimizerRule({
        customer_id: appliedCustomerId,
        preset_key: presetCheck.value as UpsertTrafficOptimizerRuleRequest['preset_key'],
        flow_id: flowCheck.value,
        campaign_id: draftCampaignId.trim() || undefined,
        enabled: true,
      });
      toast.success(`Rule created: ${rule.name ?? rule.id}`);
      setDraftFlowId('');
      setDraftCampaignId('');
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
    appliedCustomerId,
    bumpRefreshCoalesced,
    canWrite,
    creating,
    draftCampaignId,
    draftFlowId,
    draftPresetKey,
  ]);

  const onDryRunRule = useCallback(
    async (rule: TrafficOptimizerRule) => {
      if (!rule.id || dryRunningRuleId) {
        return;
      }
      setActiveRuleId(rule.id);
      setDryRunningRuleId(rule.id);
      setDryRunError(undefined);
      setDryRunResult(undefined);
      setApplyError(undefined);
      try {
        const result = await dryRunTrafficOptimizerRule(rule.id);
        setDryRunResult(result);
        if ((result.arms ?? []).length === 0) {
          toast.message('Dry-run finished with no weight changes');
        } else {
          toast.success(`Dry-run: ${result.arms?.length ?? 0} arm(s) proposed`);
        }
      } catch (err: unknown) {
        if (isAbortError(err)) {
          return;
        }
        setDryRunError(mutationError(err));
      } finally {
        setDryRunningRuleId(undefined);
      }
    },
    [dryRunningRuleId]
  );

  const onApplyRule = useCallback(
    async (rule: TrafficOptimizerRule) => {
      if (!canWrite || applyingRuleId || !rule.id) {
        return;
      }
      setApplyingRuleId(rule.id);
      setApplyError(undefined);
      try {
        const result = await applyTrafficOptimizerRule(rule.id);
        if (result.applied) {
          const count = result.campaign_ids?.length ?? 0;
          toast.success(
            count > 0 ? `Optimizer applied to ${count} campaign(s)` : 'Optimizer apply completed'
          );
        } else {
          toast.message('Optimizer apply finished with no weight changes');
        }
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        if (isAbortError(err)) {
          return;
        }
        setApplyError(mutationError(err));
      } finally {
        setApplyingRuleId(undefined);
      }
    },
    [applyingRuleId, bumpRefreshCoalesced, canWrite]
  );

  const onDeleteRule = useCallback(
    async (rule: TrafficOptimizerRule) => {
      if (!canWrite || !rule.id || deletingRuleId) {
        return;
      }
      setDeletingRuleId(rule.id);
      setDeleteError(undefined);
      try {
        await deleteTrafficOptimizerRule(rule.id);
        toast.success('Rule deleted');
        if (activeRuleId === rule.id) {
          setActiveRuleId(undefined);
          setDryRunResult(undefined);
        }
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        if (isAbortError(err)) {
          return;
        }
        setDeleteError(mutationError(err));
      } finally {
        setDeletingRuleId(undefined);
      }
    },
    [activeRuleId, bumpRefreshCoalesced, canWrite, deletingRuleId]
  );

  const fetching = presetsResource.fetching || rulesResource.fetching;
  const error = presetsResource.error ?? rulesResource.error;
  const hasSnapshot =
    presetsResource.data != null && (rulesResource.data != null || !shouldFetchRules);

  return {
    canWrite,
    appliedCustomerId,
    draftCustomerId,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
    presets,
    rules,
    fetching,
    error,
    hasSnapshot,
    draftPresetKey,
    onDraftPresetKeyChange: setDraftPresetKey,
    draftFlowId,
    onDraftFlowIdChange: setDraftFlowId,
    draftCampaignId,
    onDraftCampaignIdChange: setDraftCampaignId,
    creating,
    createError,
    createValidationError,
    onCreateRule,
    activeRule,
    dryRunResult,
    dryRunningRuleId,
    dryRunError,
    onDryRunRule,
    applyingRuleId,
    applyError,
    onApplyRule,
    deletingRuleId,
    deleteError,
    onDeleteRule,
  };
}
