// L3 smart alert rules CRUD: customer-scoped list + inline edit rows.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  createSmartAlertRule,
  deleteSmartAlertRule,
  listSmartAlertRules,
  updateSmartAlertRule,
} from '@/api/smart_alerts_api';
import {
  type SmartAlertRuleEditDraft,
  smartAlertRuleCreateBody,
  smartAlertRuleEditFromRow,
  smartAlertRuleUpsertBody,
} from '@/domains/automation/automation_rule_forms';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

const EMPTY_CREATE_DRAFT: SmartAlertRuleEditDraft = {
  name: '',
  metric: 'spend_micro',
  operator: 'gt',
  threshold: '',
  window_minutes: '60',
  webhook_url: '',
  enabled: true,
};

export function useSmartAlertsRulesPageWorkspace() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listSmartAlertRules({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, refreshToken, shouldFetch]
  );

  const [createDraft, setCreateDraft] = useState<SmartAlertRuleEditDraft>(EMPTY_CREATE_DRAFT);
  const [ruleDrafts, setRuleDrafts] = useState<Record<string, SmartAlertRuleEditDraft>>({});
  const [creating, setCreating] = useState(false);
  const [createSuccess, setCreateSuccess] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [updatingRuleId, setUpdatingRuleId] = useState<string | undefined>();
  const [deletingRuleId, setDeletingRuleId] = useState<string | undefined>();

  const listBusy = fetching || creating || updatingRuleId != null || deletingRuleId != null;
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
        next[ruleId] = smartAlertRuleEditFromRow(row);
      }
      return next;
    });
  }, [data]);

  const onCreateDraftChange = useCallback((patch: Partial<SmartAlertRuleEditDraft>) => {
    setCreateDraft((prev) => ({ ...prev, ...patch }));
  }, []);

  const onRuleDraftChange = useCallback(
    (ruleId: string, patch: Partial<SmartAlertRuleEditDraft>) => {
      setRuleDrafts((prev) => ({
        ...prev,
        [ruleId]: {
          name: prev[ruleId]?.name ?? '',
          metric: prev[ruleId]?.metric ?? '',
          operator: prev[ruleId]?.operator ?? '',
          threshold: prev[ruleId]?.threshold ?? '',
          window_minutes: prev[ruleId]?.window_minutes ?? '',
          webhook_url: prev[ruleId]?.webhook_url ?? '',
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
      await createSmartAlertRule(smartAlertRuleCreateBody(customerId, createDraft));
      setCreateSuccess(true);
      setCreateDraft(EMPTY_CREATE_DRAFT);
      toast.success('Smart alert rule created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
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
        await updateSmartAlertRule(ruleId, smartAlertRuleUpsertBody(customerId, row, draft));
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        setActionError(err instanceof Error ? err : new Error(String(err)));
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
        await deleteSmartAlertRule(ruleId);
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
  };
}
