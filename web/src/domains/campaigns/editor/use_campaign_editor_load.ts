// campaign editor load: GET campaign + flow; campaignSnapshot tracks post-mutation state separate from useResource data.
// Auto publish-check once when status is PAUSED (autoPublishCheckDone ref resets on id change).
import { useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import { checkCampaignPublish, getCampaign } from '@/api/campaigns_api';
import { isAbortError } from '@/api/client';
import { getFlow } from '@/api/flows_api';
import type { Campaign, CampaignPublishCheck } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { toError } from '@/lib/admin_error';
import { validationError } from '@/lib/admin_validation_error';

export type UseCampaignEditorLoadArgs = {
  syncFormFromCampaign: (campaign: Campaign) => void;
};

export function useCampaignEditorLoad({ syncFormFromCampaign }: UseCampaignEditorLoadArgs) {
  const { id } = useParams<{ id: string }>();
  const [campaignSnapshot, setCampaignSnapshot] = useState<Campaign | undefined>(undefined);
  const [checking, setChecking] = useState(false);
  const [publishCheck, setPublishCheck] = useState<CampaignPublishCheck | undefined>(undefined);
  const [publishCheckError, setPublishCheckError] = useState<Error | undefined>(undefined);
  const autoPublishCheckDone = useRef(false);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!id) {
        return Promise.reject(validationError('Campaign id is required.', { field: 'id' }));
      }
      return getCampaign(id, signal);
    },
    [id]
  );

  const flowId = (campaignSnapshot ?? data)?.flow_id?.trim() ?? '';

  const { data: flowData } = useResource(
    (signal) => {
      if (!flowId) {
        return Promise.resolve(undefined);
      }
      return getFlow(flowId, signal);
    },
    [flowId]
  );

  useEffect(() => {
    if (!data) {
      return;
    }
    setCampaignSnapshot(data);
    syncFormFromCampaign(data);
  }, [data, syncFormFromCampaign]);

  useEffect(() => {
    autoPublishCheckDone.current = false;
    setPublishCheck(undefined);
    setPublishCheckError(undefined);
  }, [id]);

  useEffect(() => {
    if (!id || !data || autoPublishCheckDone.current) {
      return;
    }
    if (data.status !== 'PAUSED') {
      return;
    }
    autoPublishCheckDone.current = true;

    const controller = new AbortController();
    setChecking(true);
    setPublishCheckError(undefined);

    void (async () => {
      try {
        const result = await checkCampaignPublish(id, controller.signal);
        setPublishCheck(result);
      } catch (err: unknown) {
        if (isAbortError(err)) {
          return;
        }
        setPublishCheckError(toError(err));
      } finally {
        if (!controller.signal.aborted) {
          setChecking(false);
        }
      }
    })();

    return () => {
      controller.abort();
    };
  }, [data, id]);

  const campaign = campaignSnapshot ?? data;
  const flowPaths = useMemo(() => {
    const paths = flowData?.paths;
    return Array.isArray(paths) ? paths : undefined;
  }, [flowData?.paths]);

  return {
    id,
    campaign,
    campaignSnapshot,
    setCampaignSnapshot,
    flowPaths,
    fetching,
    loadError: error,
    hasSnapshot: campaignSnapshot != null || data != null,
    checking,
    publishCheck,
    publishCheckError,
    setPublishCheck,
    setPublishCheckError,
    setChecking,
  };
}
