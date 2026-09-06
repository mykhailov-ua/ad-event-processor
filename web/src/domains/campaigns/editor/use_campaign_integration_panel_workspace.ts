// L3 campaign integration tab: manual template apply + on-demand GET integration health (not auto on mount).
import { useCallback, useEffect, useState } from 'react';

import { applyCampaignTemplates, getCampaignIntegrationHealth } from '@/api/campaigns_api';
import type { ApplyCampaignTemplatesResult, CampaignIntegrationHealth } from '@/api/types';

export function useCampaignIntegrationPanelWorkspace(campaignId: string) {
  const [draftTrafficSource, setDraftTrafficSource] = useState('');
  const [draftAffiliateNetwork, setDraftAffiliateNetwork] = useState('');
  const [draftTrackingDomain, setDraftTrackingDomain] = useState('');
  const [applying, setApplying] = useState(false);
  const [applyResult, setApplyResult] = useState<ApplyCampaignTemplatesResult | undefined>();
  const [applyError, setApplyError] = useState<Error | undefined>();
  const [health, setHealth] = useState<CampaignIntegrationHealth | undefined>();
  const [healthError, setHealthError] = useState<Error | undefined>();
  const [healthLoading, setHealthLoading] = useState(false);

  useEffect(() => {
    setApplyResult(undefined);
    setApplyError(undefined);
    setHealth(undefined);
    setHealthError(undefined);
  }, [campaignId]);

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

  const onApplyTemplates = useCallback(async () => {
    setApplying(true);
    setApplyError(undefined);
    setApplyResult(undefined);
    try {
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
      const result = await applyCampaignTemplates(campaignId, body);
      setApplyResult(result);
    } catch (err: unknown) {
      setApplyError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setApplying(false);
    }
  }, [campaignId, draftAffiliateNetwork, draftTrackingDomain, draftTrafficSource]);

  return {
    draftTrafficSource,
    setDraftTrafficSource,
    draftAffiliateNetwork,
    setDraftAffiliateNetwork,
    draftTrackingDomain,
    setDraftTrackingDomain,
    applying,
    applyResult,
    applyError,
    health,
    healthError,
    healthLoading,
    onLoadHealth,
    onApplyTemplates,
  };
}

export type CampaignIntegrationPanelWorkspace = ReturnType<
  typeof useCampaignIntegrationPanelWorkspace
>;
