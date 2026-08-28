import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  ackSmartAlertEvent,
  createSmartAlertRule,
  deleteSmartAlertRule,
  fetchSmartAlertHistory,
  fetchSmartAlertRules,
  type SmartAlertEvent,
  type SmartAlertRule,
} from '../helpers/integrations_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { SmartAlertsPanel } from '../ui/smart_alerts/smart_alerts_panel.js';

const DEFAULT_HISTORY_LIMIT = 50;

function parseHistoryLimit(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value <= 0) return DEFAULT_HISTORY_LIMIT;
  return Math.min(value, 200);
}

export function IntegrationsSmartAlertsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';
  const historyLimit = parseHistoryLimit(searchParams.get('limit'));

  const [rules, setRules] = useState<SmartAlertRule[]>([]);
  const [history, setHistory] = useState<SmartAlertEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
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
    if (!customerId) {
      setRules([]);
      setHistory([]);
      setLoading(false);
      setError(null);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [rulesResult, rulesErr] = await to(fetchSmartAlertRules(customerId, ctrl.signal));
      if (cancelled) return;
      if (rulesErr && rulesErr.name !== 'AbortError') {
        setError(rulesErr);
        setLoading(false);
        return;
      }
      setRules(rulesResult ?? []);

      const [historyResult, historyErr] = await to(
        fetchSmartAlertHistory(customerId, historyLimit, ctrl.signal)
      );
      if (cancelled) return;
      if (historyErr && historyErr.name !== 'AbortError') {
        setError(historyErr);
        setLoading(false);
        return;
      }
      setHistory(historyResult ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [customerId, historyLimit, reloadToken]);

  const onCustomerApply = useCallback(
    (nextCustomerId: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextCustomerId) next.set('customer_id', nextCustomerId);
      else next.delete('customer_id');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onHistoryLimitChange = useCallback(
    (limit: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(limit));
      setSearchParams(next, { replace: true });
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
      webhook_url: string;
      enabled: boolean;
      campaign_id?: string;
    }) => {
      setBusy(true);
      try {
        await createSmartAlertRule(body);
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
        await deleteSmartAlertRule(id);
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

  const onAckEvent = useCallback(
    async (id: string) => {
      setBusy(true);
      try {
        await ackSmartAlertEvent(id);
        pushToastMessage({ title: 'Event acknowledged', message: id });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Ack failed',
          message: err instanceof Error ? err.message : 'Ack failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [reload]
  );

  return (
    <SmartAlertsPanel
      customerId={customerId}
      rules={rules}
      history={history}
      historyLimit={historyLimit}
      loading={loading}
      error={error}
      busy={busy}
      onCustomerApply={onCustomerApply}
      onHistoryLimitChange={onHistoryLimitChange}
      onCreateRule={(body) => {
        void onCreateRule(body);
      }}
      onDeleteRule={(id) => {
        void onDeleteRule(id);
      }}
      onAckEvent={(id) => {
        void onAckEvent(id);
      }}
    />
  );
}
