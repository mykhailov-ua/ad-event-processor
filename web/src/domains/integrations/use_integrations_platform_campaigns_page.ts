// platform campaigns page: customer scope gates list fetch; row mutations coalesce bumpRefresh while listBusy.
import { useCallback, useState } from 'react';

import {
  deletePlatformCampaignLink,
  listPlatformCampaignLinks,
  pausePlatformCampaign,
  refreshPlatformCampaignLink,
  resumePlatformCampaign,
  runPlatformCampaignSync,
  setPlatformCampaignBudget,
  upsertPlatformCampaignLink,
} from '@/api/integrations_api';
import type { PlatformCampaignLink, PlatformCampaignMutation } from '@/api/types';
import { confirmDestructiveAction, mutationError as toMutationError } from '@/lib/mutation_audit';
import {
  type AdminValidationError,
  requireInteger,
  requireNonEmpty,
  toastValidationError,
} from '@/lib/admin_validation_error';
import { newRandomUuid } from '@/lib/uuid';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

export function useIntegrationsPlatformCampaignsPage() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const shouldFetch = Boolean(appliedCustomerId);
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listPlatformCampaignLinks({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, shouldFetch, refreshToken]
  );

  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [draftNetwork, setDraftNetwork] = useState('');
  const [draftExternalCampaignId, setDraftExternalCampaignId] = useState('');
  const [draftAccountId, setDraftAccountId] = useState('');
  const [draftDailyBudgetMicro, setDraftDailyBudgetMicro] = useState('');
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [pausing, setPausing] = useState(false);
  const [resuming, setResuming] = useState(false);
  const [settingBudget, setSettingBudget] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [deleteError, setDeleteError] = useState<Error | undefined>();
  const [refreshError, setRefreshError] = useState<Error | undefined>();
  const [syncError, setSyncError] = useState<Error | undefined>();
  const [mutationError, setMutationError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [deleteSuccess, setDeleteSuccess] = useState(false);
  const [refreshSuccess, setRefreshSuccess] = useState(false);
  const [syncSuccess, setSyncSuccess] = useState(false);
  const [mutationResult, setMutationResult] = useState<PlatformCampaignMutation | undefined>();
  const [formValidationError, setFormValidationError] = useState<AdminValidationError | undefined>();

  const clearFormValidationError = useCallback(() => {
    setFormValidationError(undefined);
  }, []);

  const reportFormValidationFailure = useCallback((error: AdminValidationError) => {
    setFormValidationError(error);
    toastValidationError(error);
  }, []);

  const onDraftCampaignIdChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftCampaignId(value);
    },
    [clearFormValidationError]
  );

  const onDraftNetworkChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftNetwork(value);
    },
    [clearFormValidationError]
  );

  const onDraftExternalCampaignIdChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftExternalCampaignId(value);
    },
    [clearFormValidationError]
  );

  const onDraftAccountIdChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftAccountId(value);
    },
    [clearFormValidationError]
  );

  const onDraftDailyBudgetMicroChange = useCallback(
    (value: string) => {
      clearFormValidationError();
      setDraftDailyBudgetMicro(value);
    },
    [clearFormValidationError]
  );

  const listBusy =
    fetching || saving || deleting || refreshing || syncing || pausing || resuming || settingBudget;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const clearActionFeedback = useCallback(() => {
    setSaveError(undefined);
    setDeleteError(undefined);
    setRefreshError(undefined);
    setSyncError(undefined);
    setMutationError(undefined);
    setSaveSuccess(false);
    setDeleteSuccess(false);
    setRefreshSuccess(false);
    setSyncSuccess(false);
    setMutationResult(undefined);
  }, []);

  const onPrefillFromLink = useCallback(
    (row: PlatformCampaignLink) => {
      clearFormValidationError();
      setDraftCampaignId(row.campaign_id ?? '');
      setDraftNetwork(row.network ?? '');
      setDraftExternalCampaignId(row.external_campaign_id ?? '');
      setDraftAccountId(row.account_id ?? '');
      setDraftDailyBudgetMicro(
        row.external_daily_budget_micro != null ? String(row.external_daily_budget_micro) : ''
      );
      clearActionFeedback();
    },
    [clearActionFeedback, clearFormValidationError]
  );

  const onSave = useCallback(async () => {
    if (saving) {
      return;
    }
    const campaignIdCheck = requireNonEmpty(draftCampaignId, 'Campaign ID', 'campaign_id');
    if (!campaignIdCheck.ok) {
      reportFormValidationFailure(campaignIdCheck.error);
      return;
    }
    const networkCheck = requireNonEmpty(draftNetwork, 'Network', 'network');
    if (!networkCheck.ok) {
      reportFormValidationFailure(networkCheck.error);
      return;
    }
    const externalCampaignIdCheck = requireNonEmpty(
      draftExternalCampaignId,
      'External campaign ID',
      'external_campaign_id'
    );
    if (!externalCampaignIdCheck.ok) {
      reportFormValidationFailure(externalCampaignIdCheck.error);
      return;
    }
    const customerIdCheck = requireNonEmpty(appliedCustomerId, 'Customer ID', 'customer_id');
    if (!customerIdCheck.ok) {
      reportFormValidationFailure(customerIdCheck.error);
      return;
    }
    const campaignId = campaignIdCheck.value;
    const network = networkCheck.value;
    const externalCampaignId = externalCampaignIdCheck.value;
    clearFormValidationError();
    setSaving(true);
    clearActionFeedback();
    try {
      await upsertPlatformCampaignLink(campaignId, network, {
        customer_id: customerIdCheck.value,
        external_campaign_id: externalCampaignId,
        account_id: draftAccountId.trim() || undefined,
      });
      setSaveSuccess(true);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [
    appliedCustomerId,
    bumpRefreshCoalesced,
    clearActionFeedback,
    draftAccountId,
    draftCampaignId,
    draftExternalCampaignId,
    draftNetwork,
    reportFormValidationFailure,
    clearFormValidationError,
    saving,
  ]);

  const onDelete = useCallback(async () => {
    if (deleting) {
      return;
    }
    const campaignIdCheck = requireNonEmpty(draftCampaignId, 'Campaign ID', 'campaign_id');
    if (!campaignIdCheck.ok) {
      reportFormValidationFailure(campaignIdCheck.error);
      return;
    }
    const networkCheck = requireNonEmpty(draftNetwork, 'Network', 'network');
    if (!networkCheck.ok) {
      reportFormValidationFailure(networkCheck.error);
      return;
    }
    const campaignId = campaignIdCheck.value;
    const network = networkCheck.value;
    if (!confirmDestructiveAction(`Delete platform link ${campaignId} / ${network}?`)) {
      return;
    }
    setDeleting(true);
    clearActionFeedback();
    try {
      await deletePlatformCampaignLink(campaignId, network);
      setDeleteSuccess(true);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setDeleteError(toMutationError(err));
    } finally {
      setDeleting(false);
    }
  }, [
    bumpRefreshCoalesced,
    clearActionFeedback,
    deleting,
    draftCampaignId,
    draftNetwork,
    reportFormValidationFailure,
  ]);

  const onRefresh = useCallback(async () => {
    if (refreshing) {
      return;
    }
    const campaignIdCheck = requireNonEmpty(draftCampaignId, 'Campaign ID', 'campaign_id');
    if (!campaignIdCheck.ok) {
      reportFormValidationFailure(campaignIdCheck.error);
      return;
    }
    const networkCheck = requireNonEmpty(draftNetwork, 'Network', 'network');
    if (!networkCheck.ok) {
      reportFormValidationFailure(networkCheck.error);
      return;
    }
    const campaignId = campaignIdCheck.value;
    const network = networkCheck.value;
    setRefreshing(true);
    clearActionFeedback();
    try {
      await refreshPlatformCampaignLink(campaignId, network);
      setRefreshSuccess(true);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setRefreshError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setRefreshing(false);
    }
  }, [
    bumpRefreshCoalesced,
    clearActionFeedback,
    draftCampaignId,
    draftNetwork,
    refreshing,
    reportFormValidationFailure,
  ]);

  const onSyncRun = useCallback(async () => {
    if (syncing) {
      return;
    }
    const campaignIdCheck = requireNonEmpty(draftCampaignId, 'Campaign ID', 'campaign_id');
    if (!campaignIdCheck.ok) {
      reportFormValidationFailure(campaignIdCheck.error);
      return;
    }
    const campaignId = campaignIdCheck.value;
    setSyncing(true);
    clearActionFeedback();
    try {
      await runPlatformCampaignSync({ campaign_id: campaignId });
      setSyncSuccess(true);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setSyncError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSyncing(false);
    }
  }, [bumpRefreshCoalesced, clearActionFeedback, draftCampaignId, reportFormValidationFailure, syncing]);

  const runMutation = useCallback(
    async (
      action: 'pause' | 'resume' | 'budget',
      setter: (value: boolean) => void,
      inFlight: boolean
    ) => {
      if (inFlight) {
        return;
      }
      const campaignIdCheck = requireNonEmpty(draftCampaignId, 'Campaign ID', 'campaign_id');
      if (!campaignIdCheck.ok) {
        reportFormValidationFailure(campaignIdCheck.error);
        return;
      }
      const networkCheck = requireNonEmpty(draftNetwork, 'Network', 'network');
      if (!networkCheck.ok) {
        reportFormValidationFailure(networkCheck.error);
        return;
      }
      const campaignId = campaignIdCheck.value;
      const network = networkCheck.value;
      const body: {
        network: string;
        idempotency_key: string;
        daily_budget_micro?: number;
      } = {
        network,
        idempotency_key: newRandomUuid(),
      };
      if (action === 'budget') {
        const budgetCheck = requireInteger(draftDailyBudgetMicro, 'Daily budget', {
          min: 0,
          field: 'daily_budget_micro',
        });
        if (!budgetCheck.ok) {
          reportFormValidationFailure(budgetCheck.error);
          return;
        }
        body.daily_budget_micro = budgetCheck.value;
      }
      setter(true);
      clearActionFeedback();
      try {
        let result: PlatformCampaignMutation;
        if (action === 'pause') {
          result = await pausePlatformCampaign(campaignId, body);
        } else if (action === 'resume') {
          result = await resumePlatformCampaign(campaignId, body);
        } else {
          result = await setPlatformCampaignBudget(campaignId, body);
        }
        setMutationResult(result);
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        setMutationError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setter(false);
      }
    },
    [
      bumpRefreshCoalesced,
      clearActionFeedback,
      draftCampaignId,
      draftDailyBudgetMicro,
      draftNetwork,
      reportFormValidationFailure,
    ]
  );

  const onRefreshLink = useCoalescedBumpRefresh(() => {
    void onRefresh();
  }, refreshing || fetching);

  const onPause = useCallback(() => {
    void runMutation('pause', setPausing, pausing);
  }, [pausing, runMutation]);

  const onResume = useCallback(() => {
    void runMutation('resume', setResuming, resuming);
  }, [resuming, runMutation]);

  const onSetBudget = useCallback(() => {
    void runMutation('budget', setSettingBudget, settingBudget);
  }, [runMutation, settingBudget]);

  return {
    links: data,
    appliedCustomerId,
    draftCustomerId,
    fetching,
    error,
    hasSnapshot: data != null,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
    linkForm: {
      draftCampaignId,
      draftNetwork,
      draftExternalCampaignId,
      draftAccountId,
      draftDailyBudgetMicro,
      saving,
      deleting,
      refreshing,
      syncing,
      pausing,
      resuming,
      settingBudget,
      saveError,
      deleteError,
      refreshError,
      syncError,
      mutationError,
      saveSuccess,
      deleteSuccess,
      refreshSuccess,
      syncSuccess,
      mutationResult,
      formValidationError,
      onDraftCampaignIdChange,
      onDraftNetworkChange,
      onDraftExternalCampaignIdChange,
      onDraftAccountIdChange,
      onDraftDailyBudgetMicroChange,
      onSave: () => {
        void onSave();
      },
      onDelete: () => {
        void onDelete();
      },
      onRefresh: onRefreshLink,
      onSyncRun: () => {
        void onSyncRun();
      },
      onPause,
      onResume,
      onSetBudget,
      onPrefillFromLink,
    },
  };
}
