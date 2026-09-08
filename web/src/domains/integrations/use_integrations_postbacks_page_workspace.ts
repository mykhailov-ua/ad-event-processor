// postbacks hub: configs/DLQ/test tabs over fetchPostbacksSnapshot; per-tab draft forms.
import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  fetchPostbackHealth,
  fetchPostbacksSnapshot,
  retryPostbackDlq,
  testPostbackConfig,
  updatePostbackConfig,
} from '@/api/integrations_api';
import type {
  PostbackConfig,
  PostbackDryRunResult,
  PostbackHealthRow,
  UpdatePostbackConfigRequest,
} from '@/api/types';
import { type IntegrationsPostbacksTab } from '@/domains/integrations/integrations_postbacks';
import { mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useIntegrationsPostbacksPageWorkspace() {
  const [tab, setTab] = useState<IntegrationsPostbacksTab>('configs');
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource(
    (signal) => fetchPostbacksSnapshot(signal),
    [refreshToken]
  );

  const {
    data: healthData,
    error: healthError,
    fetching: healthFetching,
  } = useResource(
    (signal) => (tab === 'health' ? fetchPostbackHealth(signal) : Promise.resolve(undefined)),
    [refreshToken, tab]
  );

  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [draftProvider, setDraftProvider] = useState('webhook');
  const [draftUrlTemplate, setDraftUrlTemplate] = useState('');
  const [draftTargetEvent, setDraftTargetEvent] = useState('');
  const [draftApiToken, setDraftApiToken] = useState('');
  const [draftTestEventCode, setDraftTestEventCode] = useState('');
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [testError, setTestError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [testResult, setTestResult] = useState<PostbackDryRunResult | undefined>();
  const [retryingId, setRetryingId] = useState<string | undefined>();
  const [retryError, setRetryError] = useState<Error | undefined>();

  const configs = useMemo(() => data?.configs ?? [], [data?.configs]);
  const dlq = useMemo(() => data?.dlq ?? [], [data?.dlq]);
  const campaignStatus = useMemo(() => data?.campaignStatus ?? [], [data?.campaignStatus]);
  const healthRows = useMemo(() => healthData?.rows ?? [], [healthData?.rows]);
  const healthAlertThreshold = healthData?.alert_threshold_success_rate ?? 95;

  const listBusy = fetching || healthFetching || saving || testing || retryingId != null;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const onPrefillFromConfig = useCallback((row: PostbackConfig) => {
    setDraftCampaignId(row.campaign_id ?? '');
    setDraftProvider(row.provider ?? 'webhook');
    setDraftUrlTemplate(row.url_template ?? '');
    setDraftTargetEvent(row.target_event ?? '');
    setDraftApiToken('');
    setDraftTestEventCode('');
    setSaveSuccess(false);
    setSaveError(undefined);
    setTestError(undefined);
    setTestResult(undefined);
  }, []);

  const onSave = useCallback(async () => {
    if (saving) {
      return;
    }
    const campaignId = draftCampaignId.trim();
    if (!campaignId) {
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    try {
      await updatePostbackConfig(campaignId, {
        provider: draftProvider as UpdatePostbackConfigRequest['provider'],
        url_template: draftUrlTemplate.trim(),
        api_token: draftApiToken.trim() || undefined,
        target_event: draftTargetEvent.trim() || undefined,
        test_event_code: draftTestEventCode.trim() || undefined,
      });
      setSaveSuccess(true);
      toast.success('Postback config saved');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setSaveError(mutationError(err));
    } finally {
      setSaving(false);
    }
  }, [
    draftApiToken,
    draftCampaignId,
    draftProvider,
    draftTargetEvent,
    draftTestEventCode,
    draftUrlTemplate,
    bumpRefreshCoalesced,
  ]);

  const onTest = useCallback(async () => {
    if (testing) {
      return;
    }
    const campaignId = draftCampaignId.trim();
    if (!campaignId) {
      return;
    }
    setTesting(true);
    setTestError(undefined);
    setTestResult(undefined);
    try {
      const result = await testPostbackConfig(campaignId);
      setTestResult(result);
    } catch (err: unknown) {
      setTestError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setTesting(false);
    }
  }, [draftCampaignId, testing]);

  const onRetryDlq = useCallback(
    async (id: string) => {
      if (retryingId != null) {
        return;
      }
      setRetryingId(id);
      setRetryError(undefined);
      try {
        await retryPostbackDlq(id);
        toast.success('DLQ entry retried');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setRetryError(nextError);
        toast.error(nextError.message);
      } finally {
        setRetryingId(undefined);
      }
    },
    [bumpRefreshCoalesced, retryingId]
  );

  return {
    tab,
    onTabChange: setTab,
    configs,
    dlq,
    campaignStatus,
    healthRows,
    healthAlertThreshold,
    healthRunbookPath: healthData?.runbook_path,
    healthFetching,
    healthError,
    hasHealthSnapshot: healthData != null,
    fetching,
    error,
    hasSnapshot: data != null,
    configForm: {
      draftCampaignId,
      draftProvider,
      draftUrlTemplate,
      draftTargetEvent,
      draftApiToken,
      draftTestEventCode,
      saving,
      testing,
      saveError,
      testError,
      saveSuccess,
      testResult,
      onDraftCampaignIdChange: setDraftCampaignId,
      onDraftProviderChange: setDraftProvider,
      onDraftUrlTemplateChange: setDraftUrlTemplate,
      onDraftTargetEventChange: setDraftTargetEvent,
      onDraftApiTokenChange: setDraftApiToken,
      onDraftTestEventCodeChange: setDraftTestEventCode,
      onSave: () => {
        void onSave();
      },
      onTest: () => {
        void onTest();
      },
      onPrefillFromConfig,
    },
    dlqActions: {
      retryingId,
      retryError,
      onRetry: (id: string) => {
        void onRetryDlq(id);
      },
    },
  };
}
