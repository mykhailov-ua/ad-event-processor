// L3 traffic optimizer rules CRUD: customer-scoped list + dry-run lane.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  createTrafficOptimizerRule,
  deleteTrafficOptimizerRule,
  dryRunTrafficOptimizerRule,
  listTrafficOptimizerRules,
  updateTrafficOptimizerRule,
} from '@/api/traffic_optimizer_api';
import type { TrafficOptimizerDryRunResult } from '@/api/types';
import {
  type TrafficOptimizerRuleEditDraft,
  trafficOptimizerRuleCreateBody,
  trafficOptimizerRuleEditFromRow,
  trafficOptimizerRuleUpsertBody,
} from '@/domains/automation/automation_rule_forms';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

const EMPTY_CREATE_DRAFT: TrafficOptimizerRuleEditDraft = {
  name: '',
  enabled: true,
};

export function useTrafficOptimizerRulesPageWorkspace() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listTrafficOptimizerRules({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, refreshToken, shouldFetch]
  );

  const [createDraft, setCreateDraft] = useState<TrafficOptimizerRuleEditDraft>(EMPTY_CREATE_DRAFT);
  const [ruleDrafts, setRuleDrafts] = useState<Record<string, TrafficOptimizerRuleEditDraft>>({});
  const [creating, setCreating] = useState(false);
  const [createSuccess, setCreateSuccess] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [updatingRuleId, setUpdatingRuleId] = useState<string | undefined>();
  const [deletingRuleId, setDeletingRuleId] = useState<string | undefined>();

  const [dryRunRuleId, setDryRunRuleId] = useState<string | undefined>(undefined);
  const [dryRunResult, setDryRunResult] = useState<TrafficOptimizerDryRunResult | undefined>(
    undefined
  );
  const [dryRunError, setDryRunError] = useState<Error | undefined>(undefined);
  const [dryRunning, setDryRunning] = useState(false);

  const listBusy =
    fetching || creating || updatingRuleId != null || deletingRuleId != null || dryRunning;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  useEffect(() => {
    if (!data?.length) {
      return;
    }
    setRuleDrafts((prev) => {
      const next = { ...prev };
      for (const row of data) {
        const ruleId = row.id ?? '';
        if (!ruleId || next[ruleId]) {
          continue;
        }
        next[ruleId] = trafficOptimizerRuleEditFromRow(row);
      }
      return next;
    });
  }, [data]);

  const onCreateDraftChange = useCallback((patch: Partial<TrafficOptimizerRuleEditDraft>) => {
    setCreateDraft((prev) => ({ ...prev, ...patch }));
  }, []);

  const onRuleDraftChange = useCallback(
    (ruleId: string, patch: Partial<TrafficOptimizerRuleEditDraft>) => {
      setRuleDrafts((prev) => ({
        ...prev,
        [ruleId]: {
          name: prev[ruleId]?.name ?? '',
          enabled: prev[ruleId]?.enabled ?? false,
          ...patch,
        },
      }));
    },
    []
  );

  const onCreate = useCallback(async () => {
    const customerId = appliedCustomerId.trim();
    if (!customerId || !createDraft.name.trim()) {
      return;
    }
    setCreating(true);
    setActionError(undefined);
    setCreateSuccess(false);
    try {
      await createTrafficOptimizerRule(trafficOptimizerRuleCreateBody(customerId, createDraft));
      setCreateSuccess(true);
      setCreateDraft(EMPTY_CREATE_DRAFT);
      toast.success('Traffic optimizer rule created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(nextError.message);
    } finally {
      setCreating(false);
    }
  }, [appliedCustomerId, bumpRefreshCoalesced, createDraft]);

  const onSaveRule = useCallback(
    async (ruleId: string) => {
      const customerId = appliedCustomerId.trim();
      const row = data?.find((item) => item.id === ruleId);
      const draft = ruleDrafts[ruleId];
      if (!customerId || !row || !draft) {
        return;
      }
      setUpdatingRuleId(ruleId);
      setActionError(undefined);
      try {
        await updateTrafficOptimizerRule(
          ruleId,
          trafficOptimizerRuleUpsertBody(customerId, row, draft)
        );
        toast.success('Rule saved');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      } finally {
        setUpdatingRuleId(undefined);
      }
    },
    [appliedCustomerId, bumpRefreshCoalesced, data, ruleDrafts]
  );

  const onDeleteRule = useCallback(
    async (ruleId: string) => {
      const row = data?.find((item) => item.id === ruleId);
      const label = row?.name?.trim() || ruleDrafts[ruleId]?.name?.trim() || ruleId;
      if (!confirmDestructiveAction(`Delete rule "${label}"?`)) {
        return;
      }
      setDeletingRuleId(ruleId);
      setActionError(undefined);
      try {
        await deleteTrafficOptimizerRule(ruleId);
        setRuleDrafts((prev) => {
          const next = { ...prev };
          delete next[ruleId];
          return next;
        });
        toast.success('Rule deleted');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      } finally {
        setDeletingRuleId(undefined);
      }
    },
    [bumpRefreshCoalesced, data, ruleDrafts]
  );

  const onDryRun = useCallback(async (ruleId: string) => {
    setDryRunning(true);
    setDryRunRuleId(ruleId);
    setDryRunError(undefined);
    setDryRunResult(undefined);
    try {
      const result = await dryRunTrafficOptimizerRule(ruleId);
      setDryRunResult(result);
      toast.success('Dry-run complete');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setDryRunError(nextError);
      toast.error(nextError.message);
    } finally {
      setDryRunning(false);
    }
  }, []);

  return {
    items: data,
    appliedCustomerId,
    draftCustomerId,
    createDraft,
    ruleDrafts,
    fetching,
    creating,
    error,
    actionError,
    createSuccess,
    hasSnapshot: data != null,
    dryRunRuleId,
    dryRunResult,
    dryRunError,
    dryRunning,
    updatingRuleId,
    deletingRuleId,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
    onCreateDraftChange,
    onRuleDraftChange,
    onCreate: () => {
      void onCreate();
    },
    onSaveRule: (ruleId: string) => {
      void onSaveRule(ruleId);
    },
    onDeleteRule: (ruleId: string) => {
      void onDeleteRule(ruleId);
    },
    onDryRun: (ruleId: string) => {
      void onDryRun(ruleId);
    },
  };
}
