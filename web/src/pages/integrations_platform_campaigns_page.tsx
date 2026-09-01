import { useCallback, useMemo, useState } from 'react';

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
import { IntegrationsPlatformCampaigns } from '@/domains/integrations/integrations_platform_campaigns';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/hooks/use_resource';

export function IntegrationsPlatformCampaignsPage() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const shouldFetch = Boolean(appliedCustomerId);
  const [refreshToken, setRefreshToken] = useState(0);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve([]);
      }
      return listPlatformCampaignLinks({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, shouldFetch, refreshToken],
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

  const links = useMemo(() => data ?? [], [data]);

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
      setDraftCampaignId(row.campaign_id ?? '');
      setDraftNetwork(row.network ?? '');
      setDraftExternalCampaignId(row.external_campaign_id ?? '');
      setDraftAccountId(row.account_id ?? '');
      setDraftDailyBudgetMicro(
        row.external_daily_budget_micro != null ? String(row.external_daily_budget_micro) : '',
      );
      clearActionFeedback();
    },
    [clearActionFeedback],
  );

  const onSave = useCallback(async () => {
    const campaignId = draftCampaignId.trim();
    const network = draftNetwork.trim();
    const externalCampaignId = draftExternalCampaignId.trim();
    if (!campaignId || !network || !externalCampaignId || !appliedCustomerId) {
      return;
    }
    setSaving(true);
    clearActionFeedback();
    try {
      await upsertPlatformCampaignLink(campaignId, network, {
        customer_id: appliedCustomerId,
        external_campaign_id: externalCampaignId,
        account_id: draftAccountId.trim() || undefined,
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
    draftAccountId,
    draftCampaignId,
    draftExternalCampaignId,
    draftNetwork,
    clearActionFeedback,
  ]);

  const onDelete = useCallback(async () => {
    const campaignId = draftCampaignId.trim();
    const network = draftNetwork.trim();
    if (!campaignId || !network) {
      return;
    }
    setDeleting(true);
    clearActionFeedback();
    try {
      await deletePlatformCampaignLink(campaignId, network);
      setDeleteSuccess(true);
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setDeleteError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDeleting(false);
    }
  }, [draftCampaignId, draftNetwork, clearActionFeedback]);

  const onRefresh = useCallback(async () => {
    const campaignId = draftCampaignId.trim();
    const network = draftNetwork.trim();
    if (!campaignId || !network) {
      return;
    }
    setRefreshing(true);
    clearActionFeedback();
    try {
      await refreshPlatformCampaignLink(campaignId, network);
      setRefreshSuccess(true);
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setRefreshError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setRefreshing(false);
    }
  }, [draftCampaignId, draftNetwork, clearActionFeedback]);

  const onSyncRun = useCallback(async () => {
    const campaignId = draftCampaignId.trim();
    if (!campaignId) {
      return;
    }
    setSyncing(true);
    clearActionFeedback();
    try {
      await runPlatformCampaignSync({ campaign_id: campaignId });
      setSyncSuccess(true);
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setSyncError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSyncing(false);
    }
  }, [clearActionFeedback, draftCampaignId]);

  const runMutation = useCallback(
    async (
      action: 'pause' | 'resume' | 'budget',
      setter: (value: boolean) => void,
    ) => {
      const campaignId = draftCampaignId.trim();
      const network = draftNetwork.trim();
      if (!campaignId || !network) {
        return;
      }
      const body = {
        network,
        idempotency_key: crypto.randomUUID(),
        ...(action === 'budget'
          ? { daily_budget_micro: Number.parseInt(draftDailyBudgetMicro.trim(), 10) }
          : {}),
      };
      if (action === 'budget' && !Number.isFinite(body.daily_budget_micro)) {
        setMutationError(new Error('Daily budget must be a valid integer'));
        return;
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
        setRefreshToken((value) => value + 1);
      } catch (err) {
        setMutationError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setter(false);
      }
    },
    [clearActionFeedback, draftCampaignId, draftDailyBudgetMicro, draftNetwork],
  );

  const onPause = useCallback(() => {
    void runMutation('pause', setPausing);
  }, [runMutation]);

  const onResume = useCallback(() => {
    void runMutation('resume', setResuming);
  }, [runMutation]);

  const onSetBudget = useCallback(() => {
    void runMutation('budget', setSettingBudget);
  }, [runMutation]);

  return (
    <IntegrationsPlatformCampaigns
      links={links}
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      fetching={fetching}
      error={error}
      hasSnapshot={!shouldFetch || data != null}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
      linkForm={{
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
        onDraftCampaignIdChange: setDraftCampaignId,
        onDraftNetworkChange: setDraftNetwork,
        onDraftExternalCampaignIdChange: setDraftExternalCampaignId,
        onDraftAccountIdChange: setDraftAccountId,
        onDraftDailyBudgetMicroChange: setDraftDailyBudgetMicro,
        onSave: () => {
          void onSave();
        },
        onDelete: () => {
          void onDelete();
        },
        onRefresh: () => {
          void onRefresh();
        },
        onSyncRun: () => {
          void onSyncRun();
        },
        onPause,
        onResume,
        onSetBudget,
        onPrefillFromLink,
      }}
    />
  );
}
