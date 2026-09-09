// cost sync hub: customer-scoped snapshot + credential/sync panel mutations.
import { useCallback, useMemo, useState } from 'react';

import {
  deleteCostSyncCredential,
  fetchCostSyncSnapshot,
  runCostSync,
  upsertCostSyncCredential,
} from '@/api/integrations_api';
import type { CostSyncCredential } from '@/api/types';
import { type IntegrationsCostSyncPanel } from '@/domains/integrations/integrations_cost_sync';
import { toError } from '@/lib/admin_error.ts';
import {
  actionGuardError,
  requireInteger,
  requireNonEmpty,
  toastValidationError,
  type AdminValidationError,
} from '@/lib/admin_validation_error';
import { confirmDestructiveAction, mutationError as toMutationError } from '@/lib/mutation_audit';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

function defaultSyncDateUtc(): string {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() - 1);
  return date.toISOString().slice(0, 10);
}

export function useIntegrationsCostSyncPageWorkspace() {
  const [panel, setPanel] = useState<IntegrationsCostSyncPanel>('networks');
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const { data, error, fetching } = useResource(
    (signal) =>
      fetchCostSyncSnapshot({ customer_id: appliedCustomerId || undefined, limit: 50 }, signal),
    [appliedCustomerId, refreshToken]
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
  const [credentialValidationError, setCredentialValidationError] = useState<
    AdminValidationError | undefined
  >();
  const [runValidationError, setRunValidationError] = useState<AdminValidationError | undefined>();
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

  const listBusy = fetching || saving || deleting || running;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const scopedError = panel === 'networks' ? undefined : error;
  const fetchingScoped = panel === 'networks' ? fetching : fetching;
  const hasScopedData = data != null;

  const onPrefillFromCredential = useCallback((row: CostSyncCredential) => {
    setDraftNetwork(row.network ?? '');
    setDraftAccountId(row.account_id ?? '');
    setDraftSyncIntervalMinutes(
      row.sync_interval_minutes != null ? String(row.sync_interval_minutes) : '60'
    );
    setSaveSuccess(false);
    setDeleteSuccess(false);
    setSaveError(undefined);
    setDeleteError(undefined);
  }, []);

  const onSave = useCallback(async () => {
    if (saving) {
      return;
    }
    if (!appliedCustomerId) {
      const err = actionGuardError('Apply a customer ID before saving credentials.');
      setCredentialValidationError(err);
      toastValidationError(err);
      return;
    }
    const networkCheck = requireNonEmpty(draftNetwork, 'Network', 'network');
    if (!networkCheck.ok) {
      setCredentialValidationError(networkCheck.error);
      toastValidationError(networkCheck.error);
      return;
    }
    const intervalCheck = requireInteger(draftSyncIntervalMinutes, 'Sync interval', {
      min: 15,
      field: 'sync_interval_minutes',
    });
    if (!intervalCheck.ok) {
      setCredentialValidationError(intervalCheck.error);
      toastValidationError(intervalCheck.error);
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    setCredentialValidationError(undefined);
    setSaveSuccess(false);
    try {
      await upsertCostSyncCredential(networkCheck.value, {
        customer_id: appliedCustomerId,
        account_id: draftAccountId.trim() || undefined,
        access_token: draftAccessToken.trim() || undefined,
        refresh_token: draftRefreshToken.trim() || undefined,
        api_key: draftApiKey.trim() || undefined,
        sync_interval_minutes: intervalCheck.value as 15 | 30 | 60 | 1440,
      });
      setSaveSuccess(true);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setSaveError(toError(err));
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
    bumpRefreshCoalesced,
  ]);

  const onDelete = useCallback(async () => {
    if (deleting) {
      return;
    }
    if (!appliedCustomerId) {
      const err = actionGuardError('Apply a customer ID before deleting credentials.');
      setCredentialValidationError(err);
      toastValidationError(err);
      return;
    }
    const networkCheck = requireNonEmpty(draftNetwork, 'Network', 'network');
    if (!networkCheck.ok) {
      setCredentialValidationError(networkCheck.error);
      toastValidationError(networkCheck.error);
      return;
    }
    if (!confirmDestructiveAction(`Delete credential for network "${networkCheck.value}"?`)) {
      return;
    }
    setDeleting(true);
    setDeleteError(undefined);
    setCredentialValidationError(undefined);
    setDeleteSuccess(false);
    try {
      await deleteCostSyncCredential(networkCheck.value, appliedCustomerId);
      setDeleteSuccess(true);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setDeleteError(toMutationError(err));
    } finally {
      setDeleting(false);
    }
  }, [appliedCustomerId, deleting, draftNetwork, bumpRefreshCoalesced]);

  const onRunSync = useCallback(async () => {
    if (running) {
      return;
    }
    if (!appliedCustomerId) {
      const err = actionGuardError('Apply a customer ID before running cost sync.');
      setRunValidationError(err);
      toastValidationError(err);
      return;
    }
    setRunning(true);
    setRunError(undefined);
    setRunValidationError(undefined);
    setRunSuccess(false);
    try {
      await runCostSync({
        customer_id: appliedCustomerId,
        network: runNetwork.trim() || undefined,
        from: runFrom.trim() || undefined,
        to: runTo.trim() || undefined,
      });
      setRunSuccess(true);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setRunError(toError(err));
    } finally {
      setRunning(false);
    }
  }, [appliedCustomerId, runFrom, runNetwork, runTo, bumpRefreshCoalesced]);

  return {
    panel,
    onPanelChange: setPanel,
    networks,
    credentials,
    history,
    appliedCustomerId,
    draftCustomerId,
    fetchingNetworks: fetching,
    fetchingScoped,
    networksError: error,
    scopedError,
    hasNetworks: data != null,
    hasScopedData,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
    runSyncForm: {
      draftNetwork: runNetwork,
      draftFrom: runFrom,
      draftTo: runTo,
      running,
      runError,
      runValidationError,
      runSuccess,
      onDraftNetworkChange: (value: string) => {
        setRunValidationError(undefined);
        setRunNetwork(value);
      },
      onDraftFromChange: (value: string) => {
        setRunValidationError(undefined);
        setRunFrom(value);
      },
      onDraftToChange: (value: string) => {
        setRunValidationError(undefined);
        setRunTo(value);
      },
      onRun: () => {
        void onRunSync();
      },
    },
    credentialForm: {
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
      credentialValidationError,
      saveSuccess,
      deleteSuccess,
      onDraftNetworkChange: (value: string) => {
        setCredentialValidationError(undefined);
        setDraftNetwork(value);
      },
      onDraftAccountIdChange: (value: string) => {
        setCredentialValidationError(undefined);
        setDraftAccountId(value);
      },
      onDraftAccessTokenChange: (value: string) => {
        setCredentialValidationError(undefined);
        setDraftAccessToken(value);
      },
      onDraftRefreshTokenChange: (value: string) => {
        setCredentialValidationError(undefined);
        setDraftRefreshToken(value);
      },
      onDraftApiKeyChange: (value: string) => {
        setCredentialValidationError(undefined);
        setDraftApiKey(value);
      },
      onDraftSyncIntervalMinutesChange: (value: string) => {
        setCredentialValidationError(undefined);
        setDraftSyncIntervalMinutes(value);
      },
      onSave: () => {
        void onSave();
      },
      onDelete: () => {
        void onDelete();
      },
      onPrefillFromCredential,
    },
  };
}
