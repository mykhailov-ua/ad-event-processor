import { useCallback, useEffect, useMemo, useState } from 'react';

import type { FraudEvidencePack, ReportRunQuery } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { defaultReportRange } from '@/lib/report_paths';
import type { EvidencePackReportConfig } from '@/domains/reports/evidence_pack_report_meta';

export function useEvidencePackReportWorkspace(config: EvidencePackReportConfig) {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedClickId = searchParams.get('click_id') ?? '';

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftClickId, setDraftClickId] = useState(appliedClickId);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
    setDraftCampaignId(appliedCampaignId);
    setDraftClickId(appliedClickId);
  }, [appliedCampaignId, appliedClickId, appliedCustomerId, appliedFrom, appliedTo]);

  const shouldFetch =
    Boolean(appliedClickId.trim()) &&
    (config.key === 'fraud-evidence-pack' || Boolean(appliedCustomerId.trim()));

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      const params: ReportRunQuery = {
        customer_id: appliedCustomerId || undefined,
        from: appliedFrom,
        to: appliedTo,
        campaign_id: appliedCampaignId || undefined,
        click_id: appliedClickId,
      };
      return config.loadReport(params, signal);
    },
    [
      appliedCampaignId,
      appliedClickId,
      appliedCustomerId,
      appliedFrom,
      appliedTo,
      config,
      shouldFetch,
    ]
  );

  const onApplyFilters = useCallback(
    (event?: { preventDefault?: () => void }) => {
      event?.preventDefault?.();
      const next = new URLSearchParams();
      const customerId = draftCustomerId.trim();
      if (customerId) {
        next.set('customer_id', customerId);
      }
      const from = fromDatetimeLocalValue(draftFrom);
      const to = fromDatetimeLocalValue(draftTo);
      if (from) {
        next.set('from', from);
      }
      if (to) {
        next.set('to', to);
      }
      const campaignId = draftCampaignId.trim();
      if (campaignId) {
        next.set('campaign_id', campaignId);
      }
      const clickId = draftClickId.trim();
      if (clickId) {
        next.set('click_id', clickId);
      }
      replaceSearchParams(next);
    },
    [draftCampaignId, draftClickId, draftCustomerId, draftFrom, draftTo, replaceSearchParams]
  );

  return {
    evidencePack: data as FraudEvidencePack | undefined,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    draftClickId,
    fetching: fetching || listQueryPending,
    listRevalidating,
    error,
    hasSnapshot: data != null || !shouldFetch,
    clickReady: shouldFetch,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onDraftCampaignIdChange: setDraftCampaignId,
    onDraftClickIdChange: setDraftClickId,
    onApplyFilters,
  };
}
