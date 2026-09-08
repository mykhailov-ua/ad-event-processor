// editor context panel: one loadKey at a time (geo|fraud|shell); loadToken bump re-fetches active lane on expand/preview toggle.
import { useCallback, useEffect, useState } from 'react';

import {
  getCampaignEditorShell,
  getCampaignFraudEditorSummary,
  getCampaignGeoSummary,
} from '@/api/campaigns_api';
import { useResource } from '@/api/use_resource';

type ContextLoadKey = 'geo' | 'fraud' | 'shell';

export function useCampaignEditorContextLoad(campaignId: string, enabled: boolean) {
  const [loadKey, setLoadKey] = useState<ContextLoadKey | undefined>();
  const [geoExpand, setGeoExpand] = useState(false);
  const [fraudPreview, setFraudPreview] = useState(false);
  const [loadToken, setLoadToken] = useState(0);

  useEffect(() => {
    setLoadKey(undefined);
    setGeoExpand(false);
    setFraudPreview(false);
    setLoadToken(0);
  }, [campaignId]);

  useEffect(() => {
    if (!enabled) {
      setLoadKey(undefined);
    }
  }, [enabled]);

  const geoResource = useResource(
    async (signal) => {
      if (!enabled || loadKey !== 'geo') {
        return undefined;
      }
      return getCampaignGeoSummary(campaignId, { expand: geoExpand }, signal);
    },
    [campaignId, enabled, geoExpand, loadKey, loadToken]
  );

  const fraudResource = useResource(
    async (signal) => {
      if (!enabled || loadKey !== 'fraud') {
        return undefined;
      }
      return getCampaignFraudEditorSummary(campaignId, { preview: fraudPreview }, signal);
    },
    [campaignId, enabled, fraudPreview, loadKey, loadToken]
  );

  const shellResource = useResource(
    async (signal) => {
      if (!enabled || loadKey !== 'shell') {
        return undefined;
      }
      return getCampaignEditorShell(campaignId, signal);
    },
    [campaignId, enabled, loadKey, loadToken]
  );

  const onLoadGeo = useCallback(() => {
    setLoadKey('geo');
    setLoadToken((token) => token + 1);
  }, []);

  const onLoadFraud = useCallback(() => {
    setLoadKey('fraud');
    setLoadToken((token) => token + 1);
  }, []);

  const onLoadShell = useCallback(() => {
    setLoadKey('shell');
    setLoadToken((token) => token + 1);
  }, []);

  const busy =
    (loadKey === 'geo' && geoResource.fetching) ||
    (loadKey === 'fraud' && fraudResource.fetching) ||
    (loadKey === 'shell' && shellResource.fetching);

  return {
    loadKey,
    geoExpand,
    fraudPreview,
    geoResource,
    fraudResource,
    shellResource,
    busy,
    onLoadGeo,
    onLoadFraud,
    onLoadShell,
    onGeoExpandChange: setGeoExpand,
    onFraudPreviewChange: setFraudPreview,
  };
}
