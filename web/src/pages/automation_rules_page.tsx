import { useCallback, useEffect, useMemo, useState } from 'react';
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
import { AutomationRulesDirectory } from '@/domains/automation/automation_rules_directory';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/hooks/use_resource';

const EMPTY_CREATE_DRAFT: AutomationRuleEditDraft = {
  name: '',
  metric: 'spend_micro',
  operator: 'gt',
  threshold: '',
  enabled: true,
};

export function AutomationRulesPage() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const [reloadKey, setReloadKey] = useState(0);
  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve([]);
      }
      return listAutomationRules({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, reloadKey, shouldFetch],
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

  const items = useMemo(() => data ?? [], [data]);

  useEffect(() => {
    if (items.length === 0) {
      return;
    }
    setRuleDrafts((prev) => {
      const next = { ...prev };
      for (const row of items) {
        const ruleId = row.id ?? '';
        if (!ruleId || next[ruleId]) {
          continue;
        }
        next[ruleId] = automationRuleEditFromRow(row);
      }
      return next;
    });
  }, [items]);

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
    [],
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
      setReloadKey((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [appliedCustomerId, createDraft]);

  const onSaveRule = useCallback(
    async (ruleId: string) => {
      const customerId = appliedCustomerId.trim();
      const row = items.find((item) => item.id === ruleId);
      const draft = ruleDrafts[ruleId];
      if (!customerId || !row || !draft) {
        return;
      }
      setUpdatingRuleId(ruleId);
      setActionError(undefined);
      try {
        await updateAutomationRule(ruleId, automationRuleUpsertBody(customerId, row, draft));
        setReloadKey((value) => value + 1);
      } catch (err) {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setUpdatingRuleId(undefined);
      }
    },
    [appliedCustomerId, items, ruleDrafts],
  );

  const onDeleteRule = useCallback(async (ruleId: string) => {
    setDeletingRuleId(ruleId);
    setActionError(undefined);
    try {
      await deleteAutomationRule(ruleId);
      setRuleDrafts((prev) => {
        const next = { ...prev };
        delete next[ruleId];
        return next;
      });
      setReloadKey((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDeletingRuleId(undefined);
    }
  }, []);

  const onDryRun = useCallback(async (ruleId: string) => {
    setDryRunning(true);
    setDryRunRuleId(ruleId);
    setDryRunError(undefined);
    setDryRunResult(undefined);
    try {
      const result = await dryRunAutomationRule(ruleId);
      setDryRunResult(result);
    } catch (err) {
      setDryRunError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDryRunning(false);
    }
  }, []);

  return (
    <AutomationRulesDirectory
      items={items}
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      createDraft={createDraft}
      ruleDrafts={ruleDrafts}
      fetching={fetching}
      creating={creating}
      error={error}
      actionError={actionError}
      createSuccess={createSuccess}
      hasSnapshot={!shouldFetch || data != null}
      dryRunRuleId={dryRunRuleId}
      dryRunResult={dryRunResult}
      dryRunError={dryRunError}
      dryRunning={dryRunning}
      updatingRuleId={updatingRuleId}
      deletingRuleId={deletingRuleId}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
      onCreateDraftChange={onCreateDraftChange}
      onRuleDraftChange={onRuleDraftChange}
      onCreate={() => {
        void onCreate();
      }}
      onSaveRule={(ruleId) => {
        void onSaveRule(ruleId);
      }}
      onDeleteRule={(ruleId) => {
        void onDeleteRule(ruleId);
      }}
      onDryRun={(ruleId) => {
        void onDryRun(ruleId);
      }}
    />
  );
}
