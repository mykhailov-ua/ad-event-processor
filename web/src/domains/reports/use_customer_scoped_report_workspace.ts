import { useCallback, useEffect, useMemo, useState } from 'react';

import type { CustomerScopedReportQuery, DataFreshness } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fetchCustomersComboboxCached } from '@/lib/customers_combobox_cache';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { defaultReportRange } from '@/lib/report_paths';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';

export type CustomerReportFetchResult<Row> = {
  rows: Row[];
  freshness?: DataFreshness;
  next_cursor?: string;
  extras?: Record<string, unknown>;
};

export type CustomerReportFetcher<Row> = (
  params: CustomerScopedReportQuery,
  signal?: AbortSignal
) => Promise<CustomerReportFetchResult<Row>>;

export function useCustomerScopedReportWorkspace<Row>(fetchReport: CustomerReportFetcher<Row>) {
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
  const appliedCompare = searchParams.get('compare') === '1';

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftCompare, setDraftCompare] = useState(appliedCompare);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
    setDraftCampaignId(appliedCampaignId);
    setDraftCompare(appliedCompare);
  }, [appliedCampaignId, appliedCompare, appliedCustomerId, appliedFrom, appliedTo]);

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
      return fetchReport(
        {
          customer_id: appliedCustomerId,
          from: appliedFrom,
          to: appliedTo,
          campaign_id: appliedCampaignId || undefined,
          limit: appliedLimit,
          offset: appliedOffset,
          compare: appliedCompare,
        },
        signal
      );
    },
    [
      appliedCampaignId,
      appliedCompare,
      appliedCustomerId,
      appliedFrom,
      appliedLimit,
      appliedOffset,
      appliedTo,
      fetchReport,
      shouldFetch,
    ]
  );

  const rows = data?.rows ?? [];
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
      if (draftCompare) {
        next.set('compare', '1');
      }
      next.set('limit', String(appliedLimit));
      next.set('offset', '0');
      replaceSearchParams(next);
    },
    [
      appliedLimit,
      draftCampaignId,
      draftCompare,
      draftCustomerId,
      draftFrom,
      draftTo,
      replaceSearchParams,
    ]
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
    nextCursor: data?.next_cursor,
    extras: data?.extras,
    customerOptions,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    draftCompare,
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
    onDraftCompareChange: setDraftCompare,
    onApplyFilters,
    onPageChange,
  };
}
