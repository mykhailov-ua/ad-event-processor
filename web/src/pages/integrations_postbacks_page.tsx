import { useCallback, useMemo, useState } from 'react';

import {
  fetchPostbacksSnapshot,
  retryPostbackDlq,
  testPostbackConfig,
  updatePostbackConfig,
} from '@/api/integrations_api';
import type { PostbackConfig, PostbackDryRunResult, UpdatePostbackConfigRequest } from '@/api/types';
import {
  IntegrationsPostbacks,
  type IntegrationsPostbacksTab,
} from '@/domains/integrations/integrations_postbacks';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function IntegrationsPostbacksPage() {
  const [tab, setTab] = useState<IntegrationsPostbacksTab>('configs');
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource(
    (signal) => fetchPostbacksSnapshot(signal),
    [refreshToken],
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

  const listBusy = fetching || saving || testing || retryingId != null;
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
      bumpRefreshCoalesced();
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
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
    } catch (err) {
      setTestError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setTesting(false);
    }
  }, [draftCampaignId, testing]);

  const onRetryDlq = useCallback(async (id: string) => {
    if (retryingId != null) {
      return;
    }
    setRetryingId(id);
    setRetryError(undefined);
    try {
      await retryPostbackDlq(id);
      bumpRefreshCoalesced();
    } catch (err) {
      setRetryError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setRetryingId(undefined);
    }
  }, [bumpRefreshCoalesced, retryingId]);

  return (
    <IntegrationsPostbacks
      tab={tab}
      onTabChange={setTab}
      configs={configs}
      dlq={dlq}
      campaignStatus={campaignStatus}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      configForm={{
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
      }}
      dlqActions={{
        retryingId,
        retryError,
        onRetry: (id) => {
          void onRetryDlq(id);
        },
      }}
    />
  );
}
