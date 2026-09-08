import { useCallback, useEffect, useMemo, useState } from 'react';

import { getPostbackReconciliationReport } from '@/api/reports_api';
import type { PostbackReconRow } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fetchCustomersComboboxCached } from '@/lib/customers_combobox_cache';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { defaultReportRange } from '@/lib/report_paths';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';

export function usePostbackReconciliationPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedLimit = parseListLimit(searchParams.get('limit'), 100);
  const appliedOffset = parseListOffset(searchParams.get('offset'));

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
    setDraftCampaignId(appliedCampaignId);
  }, [appliedCampaignId, appliedCustomerId, appliedFrom, appliedTo]);

  const { data: customersData } = useResource((signal) => fetchCustomersComboboxCached(signal), []);

  const customerOptions = useMemo((): CustomerComboboxOption[] => {
    return (customersData?.items ?? [])
      .filter((customer) => customer.id)
      .map((customer) => ({
        id: customer.id as string,
        name: customer.name ?? customer.id ?? '',
      }));
  }, [customersData?.items]);

  const shouldFetch = Boolean(appliedCustomerId.trim());

  const { data, error, fetching, revalidating: listRevalidating } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getPostbackReconciliationReport(
        {
          customer_id: appliedCustomerId,
          from: appliedFrom,
          to: appliedTo,
          campaign_id: appliedCampaignId || undefined,
          limit: appliedLimit,
          offset: appliedOffset,
        },
        signal
      );
    },
    [
      appliedCampaignId,
      appliedCustomerId,
      appliedFrom,
      appliedLimit,
      appliedOffset,
      appliedTo,
      shouldFetch,
    ]
  );

  const rows: PostbackReconRow[] = data?.rows ?? [];
  const canGoPrev = appliedOffset > 0;
  const canGoNext = Boolean(data?.next_cursor) || rows.length >= appliedLimit;

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
      next.set('limit', String(appliedLimit));
      next.set('offset', '0');
      replaceSearchParams(next);
    },
    [appliedLimit, draftCampaignId, draftCustomerId, draftFrom, draftTo, replaceSearchParams]
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('offset', String(Math.max(0, nextOffset)));
      replaceSearchParams(next);
    },
    [replaceSearchParams, searchParams]
  );

  return {
    rows,
    freshness: data?.freshness,
    customerOptions,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    fetching: fetching || listQueryPending,
    listRevalidating,
    error,
    hasSnapshot: data != null || !shouldFetch,
    canGoPrev,
    canGoNext,
    limit: appliedLimit,
    offset: appliedOffset,
    rangeLabel:
      rows.length > 0
        ? `${appliedOffset + 1} - ${appliedOffset + rows.length}${data?.next_cursor ? '+' : ''}`
        : '0 of 0',
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onDraftCampaignIdChange: setDraftCampaignId,
    onApplyFilters,
    onPageChange,
  };
}
