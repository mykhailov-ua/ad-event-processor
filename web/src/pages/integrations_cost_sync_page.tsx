import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  deleteCostSyncCredential,
  fetchCostSyncCredentials,
  fetchCostSyncHistory,
  fetchCostSyncNetworks,
  runCostSync,
  upsertCostSyncCredential,
  type CostSyncCredential,
  type CostSyncNetworkSchema,
  type CostSyncRun,
} from '../helpers/integrations_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { CostSyncPanel } from '../ui/cost_sync/cost_sync_panel.js';

const DEFAULT_LIMIT = 50;

function parseLimit(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value <= 0) return DEFAULT_LIMIT;
  return Math.min(value, 200);
}

function parseOffset(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value < 0) return 0;
  return value;
}

export function IntegrationsCostSyncPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const customerId = searchParams.get('customer_id') ?? '';
  const limit = parseLimit(searchParams.get('limit'));
  const offset = parseOffset(searchParams.get('offset'));

  const [networks, setNetworks] = useState<CostSyncNetworkSchema[]>([]);
  const [credentials, setCredentials] = useState<CostSyncCredential[]>([]);
  const [history, setHistory] = useState<CostSyncRun[]>([]);
  const [historyHasMore, setHistoryHasMore] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const [busyNetwork, setBusyNetwork] = useState<string | null>(null);

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
      const [result, err] = await to(fetchCostSyncNetworks(ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setNetworks(result ?? []);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [reloadToken]);

  useEffect(() => {
    if (!customerId) {
      setCredentials([]);
      setHistory([]);
      setHistoryHasMore(false);
      setLoading(false);
      setError(null);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [credResult, credErr] = await to(fetchCostSyncCredentials(customerId, ctrl.signal));
      if (cancelled) return;
      if (credErr && credErr.name !== 'AbortError') {
        setError(credErr);
        setLoading(false);
        return;
      }
      setCredentials(credResult ?? []);

      const [histResult, histErr] = await to(
        fetchCostSyncHistory({ customerId, limit, offset }, ctrl.signal)
      );
      if (cancelled) return;
      if (histErr && histErr.name !== 'AbortError') {
        setError(histErr);
        setLoading(false);
        return;
      }
      const rows = histResult ?? [];
      setHistory(rows);
      setHistoryHasMore(rows.length >= limit);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [customerId, limit, offset, reloadToken]);

  const patchParams = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams);
      for (const [key, value] of Object.entries(patch)) {
        if (value === null || value === '') next.delete(key);
        else next.set(key, value);
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onCustomerApply = useCallback(
    (nextCustomerId: string) => {
      patchParams({ customer_id: nextCustomerId || null, offset: '0' });
    },
    [patchParams]
  );

  const onHistoryOffsetChange = useCallback(
    (nextOffset: number) => {
      patchParams({ offset: String(nextOffset) });
    },
    [patchParams]
  );

  const onSaveCredential = useCallback(
    async (network: string, accountId: string, apiKey: string) => {
      if (!customerId) return;
      setBusyNetwork(network);
      try {
        await upsertCostSyncCredential(network, {
          customer_id: customerId,
          account_id: accountId,
          api_key: apiKey || undefined,
        });
        pushToastMessage({ title: 'Credential saved', message: network });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Save failed',
          message: err instanceof Error ? err.message : 'Save failed',
        });
      } finally {
        setBusyNetwork(null);
      }
    },
    [customerId, reload]
  );

  const onDeleteCredential = useCallback(
    async (network: string) => {
      if (!customerId) return;
      setBusyNetwork(network);
      try {
        await deleteCostSyncCredential(network, customerId);
        pushToastMessage({ title: 'Credential deleted', message: network });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Delete failed',
          message: err instanceof Error ? err.message : 'Delete failed',
        });
      } finally {
        setBusyNetwork(null);
      }
    },
    [customerId, reload]
  );

  const onRunSync = useCallback(
    async (network: string) => {
      if (!customerId) return;
      setBusyNetwork(network);
      try {
        await runCostSync({ customer_id: customerId, network });
        pushToastMessage({ title: 'Sync queued', message: network });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Run failed',
          message: err instanceof Error ? err.message : 'Run failed',
        });
      } finally {
        setBusyNetwork(null);
      }
    },
    [customerId, reload]
  );

  return (
    <CostSyncPanel
      customerId={customerId}
      networks={networks}
      credentials={credentials}
      history={history}
      historyLimit={limit}
      historyOffset={offset}
      historyHasMore={historyHasMore}
      loading={loading}
      error={error}
      busyNetwork={busyNetwork}
      onCustomerApply={onCustomerApply}
      onHistoryOffsetChange={onHistoryOffsetChange}
      onSaveCredential={(network, accountId, apiKey) => {
        void onSaveCredential(network, accountId, apiKey);
      }}
      onDeleteCredential={(network) => {
        void onDeleteCredential(network);
      }}
      onRunSync={(network) => {
        void onRunSync(network);
      }}
    />
  );
}
