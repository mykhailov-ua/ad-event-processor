import { useCallback, useEffect, useMemo, useState } from 'react';
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
import { TrafficOptimizerRulesDirectory } from '@/domains/automation/traffic_optimizer_rules_directory';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

const EMPTY_CREATE_DRAFT: TrafficOptimizerRuleEditDraft = {
  name: '',
  enabled: true,
};

export function TrafficOptimizerRulesPage() {
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
      return listTrafficOptimizerRules({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, reloadKey, shouldFetch],
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
    undefined,
  );
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
        next[ruleId] = trafficOptimizerRuleEditFromRow(row);
      }
      return next;
    });
  }, [items]);

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
      await createTrafficOptimizerRule(trafficOptimizerRuleCreateBody(customerId, createDraft));
      setCreateSuccess(true);
      setCreateDraft(EMPTY_CREATE_DRAFT);
      toast.success('Traffic optimizer rule created');
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
        await updateTrafficOptimizerRule(
          ruleId,
          trafficOptimizerRuleUpsertBody(customerId, row, draft),
        );
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
      await deleteTrafficOptimizerRule(ruleId);
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
      const result = await dryRunTrafficOptimizerRule(ruleId);
      setDryRunResult(result);
    } catch (err) {
      setDryRunError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDryRunning(false);
    }
  }, []);

  return (
    <TrafficOptimizerRulesDirectory
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
