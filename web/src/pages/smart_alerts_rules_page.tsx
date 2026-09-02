import { useCallback, useEffect, useMemo, useState } from 'react';
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
import { SmartAlertsRulesDirectory } from '@/domains/automation/smart_alerts_rules_directory';
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

export function SmartAlertsRulesPage() {
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
      return listSmartAlertRules({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, reloadKey, shouldFetch],
  );

  const [createDraft, setCreateDraft] = useState<SmartAlertRuleEditDraft>(EMPTY_CREATE_DRAFT);
  const [ruleDrafts, setRuleDrafts] = useState<Record<string, SmartAlertRuleEditDraft>>({});
  const [creating, setCreating] = useState(false);
  const [createSuccess, setCreateSuccess] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [updatingRuleId, setUpdatingRuleId] = useState<string | undefined>();
  const [deletingRuleId, setDeletingRuleId] = useState<string | undefined>();

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
        next[ruleId] = smartAlertRuleEditFromRow(row);
      }
      return next;
    });
  }, [items]);

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
      await createSmartAlertRule(smartAlertRuleCreateBody(customerId, createDraft));
      setCreateSuccess(true);
      setCreateDraft(EMPTY_CREATE_DRAFT);
      toast.success('Smart alert rule created');
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
        await updateSmartAlertRule(ruleId, smartAlertRuleUpsertBody(customerId, row, draft));
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
      await deleteSmartAlertRule(ruleId);
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

  return (
    <SmartAlertsRulesDirectory
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
    />
  );
}
