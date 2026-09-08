// automation rules CRUD: customer-scoped list + inline edit rows + dry-run lane.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  createAutomationRule,
  deleteAutomationRule,
  dryRunAutomationRule,
  listAutomationRules,
  updateAutomationRule,
} from '@/api/automation_api';
import type { AutomationDryRunResult } from '@/api/types';
import {
  type AutomationRuleEditDraft,
  automationRuleCreateBody,
  automationRuleEditFromRow,
  automationRuleUpsertBody,
} from '@/domains/automation/automation_rule_forms';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

const EMPTY_CREATE_DRAFT: AutomationRuleEditDraft = {
  name: '',
  metric: 'spend_micro',
  operator: 'gt',
  threshold: '',
  enabled: true,
};

export function useAutomationRulesPageWorkspace() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listAutomationRules({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, refreshToken, shouldFetch]
  );

  const [createDraft, setCreateDraft] = useState<AutomationRuleEditDraft>(EMPTY_CREATE_DRAFT);
  const [ruleDrafts, setRuleDrafts] = useState<Record<string, AutomationRuleEditDraft>>({});
  const [creating, setCreating] = useState(false);
  const [createSuccess, setCreateSuccess] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [updatingRuleId, setUpdatingRuleId] = useState<string | undefined>();
  const [deletingRuleId, setDeletingRuleId] = useState<string | undefined>();

  const [dryRunRuleId, setDryRunRuleId] = useState<string | undefined>(undefined);
  const [dryRunResult, setDryRunResult] = useState<AutomationDryRunResult | undefined>(undefined);
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
        next[ruleId] = automationRuleEditFromRow(row);
      }
      return next;
    });
  }, [data]);

  const onCreateDraftChange = useCallback((patch: Partial<AutomationRuleEditDraft>) => {
    setCreateDraft((prev) => ({ ...prev, ...patch }));
  }, []);

  const onRuleDraftChange = useCallback(
    (ruleId: string, patch: Partial<AutomationRuleEditDraft>) => {
      setRuleDrafts((prev) => ({
        ...prev,
        [ruleId]: {
          name: prev[ruleId]?.name ?? '',
          metric: prev[ruleId]?.metric ?? '',
          operator: prev[ruleId]?.operator ?? '',
          threshold: prev[ruleId]?.threshold ?? '',
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
      await createAutomationRule(automationRuleCreateBody(customerId, createDraft));
      setCreateSuccess(true);
      setCreateDraft(EMPTY_CREATE_DRAFT);
      toast.success('Automation rule created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
      toast.error(err instanceof Error ? err.message : String(err));
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
        await updateAutomationRule(ruleId, automationRuleUpsertBody(customerId, row, draft));
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
        await deleteAutomationRule(ruleId);
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
      const result = await dryRunAutomationRule(ruleId);
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
