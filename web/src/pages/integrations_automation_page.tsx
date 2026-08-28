import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  createAutomationRule,
  deleteAutomationRule,
  dryRunAutomationRule,
  fetchAutomationPresets,
  fetchAutomationRules,
  type AutomationDryRunResponse,
  type AutomationPreset,
  type AutomationRule,
} from '../helpers/integrations_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { AutomationPanel } from '../ui/automation/automation_panel.js';

export function IntegrationsAutomationPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';

  const [presets, setPresets] = useState<AutomationPreset[]>([]);
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [dryRun, setDryRun] = useState<AutomationDryRunResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [dryRunRuleId, setDryRunRuleId] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    void (async () => {
      const [result, err] = await to(fetchAutomationPresets(ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setPresets(result ?? []);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  useEffect(() => {
    if (!customerId) {
      setRules([]);
      setLoading(false);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchAutomationRules(customerId, ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setRules(result ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [customerId, reloadToken]);

  const onCustomerApply = useCallback(
    (nextCustomerId: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextCustomerId) next.set('customer_id', nextCustomerId);
      else next.delete('customer_id');
      setSearchParams(next, { replace: true });
      setDryRun(null);
    },
    [searchParams, setSearchParams]
  );

  const onCreateRule = useCallback(
    async (body: {
      customer_id: string;
      name: string;
      metric: string;
      operator: string;
      threshold: number;
      window_minutes: number;
      enabled: boolean;
      preset_key?: string;
    }) => {
      setBusy(true);
      try {
        await createAutomationRule(body);
        pushToastMessage({ title: 'Rule created', message: body.name });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Create failed',
          message: err instanceof Error ? err.message : 'Create failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  const onDeleteRule = useCallback(
    async (id: string) => {
      setBusy(true);
      try {
        await deleteAutomationRule(id);
        pushToastMessage({ title: 'Rule deleted', message: id });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Delete failed',
          message: err instanceof Error ? err.message : 'Delete failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  const onDryRun = useCallback(async (id: string) => {
    setDryRunRuleId(id);
    setBusy(true);
    try {
      const result = await dryRunAutomationRule(id);
      setDryRun(result);
      pushToastMessage({
        title: 'Dry-run complete',
        message: `${result.would_fire?.length ?? 0} match(es)`,
      });
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({
        title: 'Dry-run failed',
        message: err instanceof Error ? err.message : 'Dry-run failed',
      });
    } finally {
      setBusy(false);
      setDryRunRuleId(null);
    }
  }, []);

  return (
    <AutomationPanel
      customerId={customerId}
      presets={presets}
      rules={rules}
      dryRun={dryRun}
      loading={loading}
      error={error}
      busy={busy}
      dryRunRuleId={dryRunRuleId}
      onCustomerApply={onCustomerApply}
      onCreateRule={(body) => {
        void onCreateRule(body);
      }}
      onDeleteRule={(id) => {
        void onDeleteRule(id);
      }}
      onDryRun={(id) => {
        void onDryRun(id);
      }}
    />
  );
}
