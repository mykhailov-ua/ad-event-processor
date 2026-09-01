import { useCallback, useMemo, useState } from 'react';

import {
  deleteCostSyncCredential,
  fetchCostSyncSnapshot,
  runCostSync,
  upsertCostSyncCredential,
} from '@/api/integrations_api';
import type { CostSyncCredential } from '@/api/types';
import {
  IntegrationsCostSync,
  type IntegrationsCostSyncPanel,
} from '@/domains/integrations/integrations_cost_sync';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/hooks/use_resource';

function defaultSyncDateUtc(): string {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() - 1);
  return date.toISOString().slice(0, 10);
}

export function IntegrationsCostSyncPage() {
  const [panel, setPanel] = useState<IntegrationsCostSyncPanel>('networks');
  const [refreshToken, setRefreshToken] = useState(0);

  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const { data, error, fetching } = useResource(
    (signal) =>
      fetchCostSyncSnapshot(
        { customer_id: appliedCustomerId || undefined, limit: 50 },
        signal,
      ),
    [appliedCustomerId, refreshToken],
  );

  const [draftNetwork, setDraftNetwork] = useState('');
  const [draftAccountId, setDraftAccountId] = useState('');
  const [draftAccessToken, setDraftAccessToken] = useState('');
  const [draftRefreshToken, setDraftRefreshToken] = useState('');
  const [draftApiKey, setDraftApiKey] = useState('');
  const [draftSyncIntervalMinutes, setDraftSyncIntervalMinutes] = useState('60');
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [deleteError, setDeleteError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [deleteSuccess, setDeleteSuccess] = useState(false);
  const [runNetwork, setRunNetwork] = useState('');
  const [runFrom, setRunFrom] = useState(defaultSyncDateUtc);
  const [runTo, setRunTo] = useState(defaultSyncDateUtc);
  const [running, setRunning] = useState(false);
  const [runError, setRunError] = useState<Error | undefined>();
  const [runSuccess, setRunSuccess] = useState(false);

  const networks = useMemo(() => data?.networks ?? [], [data?.networks]);
  const credentials = useMemo(() => data?.credentials ?? [], [data?.credentials]);
  const history = useMemo(() => data?.history ?? [], [data?.history]);

  const scopedError = panel === 'networks' ? undefined : error;
  const fetchingScoped = panel === 'networks' ? fetching : fetching;
  const hasScopedData = data != null;

  const onPrefillFromCredential = useCallback((row: CostSyncCredential) => {
    setDraftNetwork(row.network ?? '');
    setDraftAccountId(row.account_id ?? '');
    setDraftSyncIntervalMinutes(
      row.sync_interval_minutes != null ? String(row.sync_interval_minutes) : '60',
    );
    setSaveSuccess(false);
    setDeleteSuccess(false);
    setSaveError(undefined);
    setDeleteError(undefined);
  }, []);

  const onSave = useCallback(async () => {
    const network = draftNetwork.trim();
    if (!network || !appliedCustomerId) {
      return;
    }
    const interval = Number.parseInt(draftSyncIntervalMinutes.trim(), 10);
    setSaving(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    try {
      await upsertCostSyncCredential(network, {
        customer_id: appliedCustomerId,
        account_id: draftAccountId.trim() || undefined,
        access_token: draftAccessToken.trim() || undefined,
        refresh_token: draftRefreshToken.trim() || undefined,
        api_key: draftApiKey.trim() || undefined,
        sync_interval_minutes: (Number.isFinite(interval) ? interval : 60) as 15 | 30 | 60 | 1440,
      });
      setSaveSuccess(true);
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [
    appliedCustomerId,
    draftAccessToken,
    draftAccountId,
    draftApiKey,
    draftNetwork,
    draftRefreshToken,
    draftSyncIntervalMinutes,
  ]);

  const onDelete = useCallback(async () => {
    const network = draftNetwork.trim();
    if (!network || !appliedCustomerId) {
      return;
    }
    setDeleting(true);
    setDeleteError(undefined);
    setDeleteSuccess(false);
    try {
      await deleteCostSyncCredential(network, appliedCustomerId);
      setDeleteSuccess(true);
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setDeleteError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDeleting(false);
    }
  }, [appliedCustomerId, draftNetwork]);

  const onRunSync = useCallback(async () => {
    if (!appliedCustomerId) {
      return;
    }
    setRunning(true);
    setRunError(undefined);
    setRunSuccess(false);
    try {
      await runCostSync({
        customer_id: appliedCustomerId,
        network: runNetwork.trim() || undefined,
        from: runFrom.trim() || undefined,
        to: runTo.trim() || undefined,
      });
      setRunSuccess(true);
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setRunError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setRunning(false);
    }
  }, [appliedCustomerId, runFrom, runNetwork, runTo]);

  return (
    <IntegrationsCostSync
      panel={panel}
      onPanelChange={setPanel}
      networks={networks}
      credentials={credentials}
      history={history}
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      fetchingNetworks={fetching}
      fetchingScoped={fetchingScoped}
      networksError={error}
      scopedError={scopedError}
      hasNetworks={data != null}
      hasScopedData={hasScopedData}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
      runSyncForm={{
        draftNetwork: runNetwork,
        draftFrom: runFrom,
        draftTo: runTo,
        running,
        runError,
        runSuccess,
        onDraftNetworkChange: setRunNetwork,
        onDraftFromChange: setRunFrom,
        onDraftToChange: setRunTo,
        onRun: () => {
          void onRunSync();
        },
      }}
      credentialForm={{
        draftNetwork,
        draftAccountId,
        draftAccessToken,
        draftRefreshToken,
        draftApiKey,
        draftSyncIntervalMinutes,
        saving,
        deleting,
        saveError,
        deleteError,
        saveSuccess,
        deleteSuccess,
        onDraftNetworkChange: setDraftNetwork,
        onDraftAccountIdChange: setDraftAccountId,
        onDraftAccessTokenChange: setDraftAccessToken,
        onDraftRefreshTokenChange: setDraftRefreshToken,
        onDraftApiKeyChange: setDraftApiKey,
        onDraftSyncIntervalMinutesChange: setDraftSyncIntervalMinutes,
        onSave: () => {
          void onSave();
        },
        onDelete: () => {
          void onDelete();
        },
        onPrefillFromCredential,
      }}
    />
  );
}
