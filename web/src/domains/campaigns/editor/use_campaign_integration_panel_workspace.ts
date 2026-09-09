// campaign integration tab: template apply, dry-run preview, copy URLs, on-demand health.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  applyCampaignTemplates,
  dryRunCampaignTemplates,
  getCampaignIntegrationHealth,
} from '@/api/campaigns_api';
import type {
  ApplyCampaignTemplatesResult,
  Campaign,
  CampaignIntegrationHealth,
  DryRunCampaignTemplatesResult,
} from '@/api/types';

export function useCampaignIntegrationPanelWorkspace(
  campaignId: string,
  campaign?: Campaign
) {
  const [draftTrafficSource, setDraftTrafficSource] = useState('');
  const [draftAffiliateNetwork, setDraftAffiliateNetwork] = useState('');
  const [draftTrackingDomain, setDraftTrackingDomain] = useState('');
  const [applying, setApplying] = useState(false);
  const [dryRunning, setDryRunning] = useState(false);
  const [applyResult, setApplyResult] = useState<ApplyCampaignTemplatesResult | undefined>();
  const [dryRunResult, setDryRunResult] = useState<DryRunCampaignTemplatesResult | undefined>();
  const [applyError, setApplyError] = useState<Error | undefined>();
  const [dryRunError, setDryRunError] = useState<Error | undefined>();
  const [health, setHealth] = useState<CampaignIntegrationHealth | undefined>();
  const [healthError, setHealthError] = useState<Error | undefined>();
  const [healthLoading, setHealthLoading] = useState(false);

  useEffect(() => {
    setApplyResult(undefined);
    setDryRunResult(undefined);
    setApplyError(undefined);
    setDryRunError(undefined);
    setHealth(undefined);
    setHealthError(undefined);
  }, [campaignId]);

  const buildApplyBody = useCallback(() => {
    const body: Parameters<typeof applyCampaignTemplates>[1] = {};
    if (draftTrafficSource.trim()) {
      body.traffic_source = draftTrafficSource.trim();
    }
    if (draftAffiliateNetwork.trim()) {
      body.affiliate_network = draftAffiliateNetwork.trim();
    }
    if (draftTrackingDomain.trim()) {
      body.tracking_domain = draftTrackingDomain.trim();
    }
    return body;
  }, [draftAffiliateNetwork, draftTrackingDomain, draftTrafficSource]);

  const onLoadHealth = useCallback(async () => {
    setHealthLoading(true);
    setHealthError(undefined);
    try {
      const result = await getCampaignIntegrationHealth(campaignId);
      setHealth(result);
    } catch (err: unknown) {
      setHealthError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setHealthLoading(false);
    }
  }, [campaignId]);

  const onDryRunTemplates = useCallback(async () => {
    setDryRunning(true);
    setDryRunError(undefined);
    setDryRunResult(undefined);
    try {
      const result = await dryRunCampaignTemplates(campaignId, buildApplyBody());
      setDryRunResult(result);
    } catch (err: unknown) {
      setDryRunError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDryRunning(false);
    }
  }, [buildApplyBody, campaignId]);

  const onApplyTemplates = useCallback(async () => {
    setApplying(true);
    setApplyError(undefined);
    setApplyResult(undefined);
    try {
      const result = await applyCampaignTemplates(campaignId, buildApplyBody());
      setApplyResult(result);
      toast.success('Integration templates applied');
    } catch (err: unknown) {
      setApplyError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [buildApplyBody, campaignId]);

  const clickCopyURL =
    applyResult?.traffic_source?.target_url ??
    dryRunResult?.target_url ??
    campaign?.target_url ??
    '';
  const postbackCopyURL =
    applyResult?.affiliate_postback?.panel_postback_url ??
    dryRunResult?.panel_postback_url ??
    applyResult?.affiliate_postback?.url_template ??
    dryRunResult?.postback_url_template ??
    '';

  return {
    draftTrafficSource,
    setDraftTrafficSource,
    draftAffiliateNetwork,
    setDraftAffiliateNetwork,
    draftTrackingDomain,
    setDraftTrackingDomain,
    applying,
    dryRunning,
    applyResult,
    dryRunResult,
    applyError,
    dryRunError,
    health,
    healthError,
    healthLoading,
    clickCopyURL,
    postbackCopyURL,
    statusIntegrationSchemaName: campaign?.status_integration_schema_name,
    onLoadHealth,
    onApplyTemplates,
    onDryRunTemplates,
  };
}

export type CampaignIntegrationPanelWorkspace = ReturnType<
  typeof useCampaignIntegrationPanelWorkspace
>;
