import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import type {
  CampaignFlowValidateResponse,
  CampaignSmokeResult,
  PostbackDryRunResult,
} from '@/api/types';
import { toError } from '@/lib/admin_error';

import {
  runIntegrationSmokeTest,
  runPostbackDryRun,
  validateIntegrationFlow,
} from '@/domains/integrations/integration_debug_api';

export function useIntegrationsDebuggerPageWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [loadingKey, setLoadingKey] = useState<string | undefined>();
  const [actionError, setActionError] = useState<Error | undefined>();
  const [smokeResult, setSmokeResult] = useState<CampaignSmokeResult | undefined>();
  const [flowResult, setFlowResult] = useState<CampaignFlowValidateResponse | undefined>();
  const [postbackResult, setPostbackResult] = useState<PostbackDryRunResult | undefined>();

  useEffect(() => {
    setDraftCampaignId(appliedCampaignId);
    setSmokeResult(undefined);
    setFlowResult(undefined);
    setPostbackResult(undefined);
    setActionError(undefined);
  }, [appliedCampaignId]);

  const busy = loadingKey != null;

  const onApplyCampaignId = useCallback(() => {
    const trimmed = draftCampaignId.trim();
    const next = new URLSearchParams(searchParams);
    if (trimmed) {
      next.set('campaign_id', trimmed);
    } else {
      next.delete('campaign_id');
    }
    setSearchParams(next, { replace: true });
  }, [draftCampaignId, searchParams, setSearchParams]);

  const resolveCampaignId = useCallback((): string | undefined => {
    const campaignId = draftCampaignId.trim();
    if (!campaignId) {
      setActionError(new Error('Campaign ID is required.'));
      return undefined;
    }
    return campaignId;
  }, [draftCampaignId]);

  const runAction = useCallback(async (key: string, action: () => Promise<void>) => {
    setLoadingKey(key);
    setActionError(undefined);
    try {
      await action();
    } catch (err: unknown) {
      setActionError(toError(err));
    } finally {
      setLoadingKey(undefined);
    }
  }, []);

  const onRunSmoke = useCallback(() => {
    const campaignId = resolveCampaignId();
    if (!campaignId) {
      return;
    }
    void runAction('smoke', async () => {
      setSmokeResult(await runIntegrationSmokeTest(campaignId));
    });
  }, [resolveCampaignId, runAction]);

  const onValidateFlow = useCallback(() => {
    const campaignId = resolveCampaignId();
    if (!campaignId) {
      return;
    }
    void runAction('flow', async () => {
      setFlowResult(await validateIntegrationFlow(campaignId));
    });
  }, [resolveCampaignId, runAction]);

  const onPostbackDryRun = useCallback(() => {
    const campaignId = resolveCampaignId();
    if (!campaignId) {
      return;
    }
    void runAction('postback', async () => {
      setPostbackResult(await runPostbackDryRun(campaignId));
    });
  }, [resolveCampaignId, runAction]);

  const canRun = draftCampaignId.trim().length > 0;

  return {
    draftCampaignId,
    setDraftCampaignId,
    onApplyCampaignId,
    loadingKey,
    busy,
    canRun,
    actionError,
    smokeResult,
    flowResult,
    postbackResult,
    onRunSmoke,
    onValidateFlow,
    onPostbackDryRun,
  };
}

export type IntegrationsDebuggerPageWorkspace = ReturnType<
  typeof useIntegrationsDebuggerPageWorkspace
>;
