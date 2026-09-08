// fraud decision lookup: URL searchParams are applied filters; draft* sync on navigation.
// GET /fraud/decision runs only when customer_id and ip_hash are set (useResource gate).
import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { getFraudDecision } from '@/api/fraud_api';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';

export function useFraudDecisionPageWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedIpHash = searchParams.get('ip_hash') ?? '';
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedHours = searchParams.get('hours') ?? '24';

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftIpHash, setDraftIpHash] = useState(appliedIpHash);
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftHours, setDraftHours] = useState(appliedHours);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftIpHash(appliedIpHash);
    setDraftCampaignId(appliedCampaignId);
    setDraftHours(appliedHours);
  }, [appliedCampaignId, appliedCustomerId, appliedHours, appliedIpHash]);

  const shouldFetch = Boolean(appliedCustomerId && appliedIpHash);
  const parsedHours = Number.parseInt(appliedHours, 10);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getFraudDecision(
        {
          customer_id: appliedCustomerId,
          ip_hash: appliedIpHash,
          campaign_id: appliedCampaignId || undefined,
          hours: Number.isFinite(parsedHours) ? parsedHours : 24,
        },
        signal
      );
    },
    [appliedCampaignId, appliedCustomerId, appliedHours, appliedIpHash, parsedHours, shouldFetch]
  );

  const onExplain = useCallback(() => {
    const next = new URLSearchParams();
    const customerId = draftCustomerId.trim();
    const ipHash = draftIpHash.trim();
    if (customerId) {
      next.set('customer_id', customerId);
    }
    if (ipHash) {
      next.set('ip_hash', ipHash);
    }
    if (draftCampaignId.trim()) {
      next.set('campaign_id', draftCampaignId.trim());
    }
    if (draftHours.trim()) {
      next.set('hours', draftHours.trim());
    }
    setSearchParams(next, { replace: true });
  }, [draftCampaignId, draftCustomerId, draftHours, draftIpHash, setSearchParams]);

  return {
    decision: data,
    draftCustomerId,
    draftIpHash,
    draftCampaignId,
    draftHours,
    fetching,
    error,
    hasSnapshot: !shouldFetch || data != null,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftIpHashChange: setDraftIpHash,
    onDraftCampaignIdChange: setDraftCampaignId,
    onDraftHoursChange: setDraftHours,
    onExplain,
  };
}
