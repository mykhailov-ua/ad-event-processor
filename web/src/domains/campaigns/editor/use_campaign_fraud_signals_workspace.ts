import { useCallback, useState } from 'react';

import { getCrowdWaveSummary, getProbeClusterSummary } from '@/api/fraud_api';
import type { CrowdWaveSummary, ProbeClusterSummary } from '@/api/types';
import { toError } from '@/lib/admin_error';

export function useCampaignFraudSignalsWorkspace(campaignId: string) {
  const [clusterId, setClusterId] = useState('');
  const [probeLoading, setProbeLoading] = useState(false);
  const [crowdLoading, setCrowdLoading] = useState(false);
  const [probeError, setProbeError] = useState<Error | undefined>();
  const [crowdError, setCrowdError] = useState<Error | undefined>();
  const [probeSummary, setProbeSummary] = useState<ProbeClusterSummary | undefined>();
  const [crowdSummary, setCrowdSummary] = useState<CrowdWaveSummary | undefined>();

  const onLoadCrowdWave = useCallback(async () => {
    if (crowdLoading || !campaignId) {
      return;
    }
    setCrowdLoading(true);
    setCrowdError(undefined);
    try {
      setCrowdSummary(await getCrowdWaveSummary(campaignId));
    } catch (err) {
      setCrowdError(toError(err));
      setCrowdSummary(undefined);
    } finally {
      setCrowdLoading(false);
    }
  }, [campaignId, crowdLoading]);

  const onLoadProbeCluster = useCallback(async () => {
    const id = clusterId.trim();
    if (probeLoading || !id) {
      return;
    }
    setProbeLoading(true);
    setProbeError(undefined);
    try {
      setProbeSummary(await getProbeClusterSummary(id));
    } catch (err) {
      setProbeError(toError(err));
      setProbeSummary(undefined);
    } finally {
      setProbeLoading(false);
    }
  }, [clusterId, probeLoading]);

  return {
    clusterId,
    setClusterId,
    probeLoading,
    crowdLoading,
    probeError,
    crowdError,
    probeSummary,
    crowdSummary,
    onLoadCrowdWave,
    onLoadProbeCluster,
  };
}

export type CampaignFraudSignalsWorkspace = ReturnType<typeof useCampaignFraudSignalsWorkspace>;
