import { useCallback, useEffect, useMemo, useState } from 'react';

import { getSourceQualityReport } from '@/api/reports_api';
import type { SourceQualityGroupBy, SourceQualityRow } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fetchCustomersComboboxCached } from '@/lib/customers_combobox_cache';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { defaultReportRange } from '@/lib/report_paths';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';

const SOURCE_QUALITY_GROUP_BY: SourceQualityGroupBy[] = [
  'placement',
  'campaign',
  'country',
  'city',
  'device',
  'sub_id',
];

function parseGroupByParam(searchParams: URLSearchParams): SourceQualityGroupBy[] {
  const raw = searchParams.getAll('group_by');
  const seen = new Set<SourceQualityGroupBy>();
  const out: SourceQualityGroupBy[] = [];
  for (const value of raw) {
    for (const part of value.split(',')) {
      const trimmed = part.trim() as SourceQualityGroupBy;
      if (!SOURCE_QUALITY_GROUP_BY.includes(trimmed) || seen.has(trimmed)) {
        continue;
      }
      seen.add(trimmed);
      out.push(trimmed);
    }
  }
  return out;
}

export function sourceQualityNeedsDetailRows(groupBy: SourceQualityGroupBy[]): boolean {
  return groupBy.some(
    (dim) => dim === 'country' || dim === 'city' || dim === 'device' || dim === 'sub_id'
  );
}

export function useSourceQualityPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session, user } = useSession();
  const canWrite = user?.permissions?.includes('campaigns:write') ?? false;
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedLimit = parseListLimit(searchParams.get('limit'), 100);
  const appliedOffset = parseListOffset(searchParams.get('offset'));
  const appliedCompare = searchParams.get('compare') === '1';
  const appliedGroupBy = useMemo(() => parseGroupByParam(searchParams), [searchParams]);

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftCompare, setDraftCompare] = useState(appliedCompare);
  const [draftGroupBy, setDraftGroupBy] = useState<SourceQualityGroupBy[]>(appliedGroupBy);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
    setDraftCampaignId(appliedCampaignId);
    setDraftCompare(appliedCompare);
    setDraftGroupBy(appliedGroupBy);
  }, [
    appliedCampaignId,
    appliedCompare,
    appliedCustomerId,
    appliedFrom,
    appliedGroupBy,
    appliedTo,
  ]);

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
  const detailMode = sourceQualityNeedsDetailRows(appliedGroupBy);

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
      return getSourceQualityReport(
        {
          customer_id: appliedCustomerId,
          from: appliedFrom,
          to: appliedTo,
          campaign_id: appliedCampaignId || undefined,
          limit: appliedLimit,
          offset: appliedOffset,
          compare: appliedCompare ? '1' : undefined,
          group_by: appliedGroupBy.length > 0 ? appliedGroupBy : undefined,
        },
        signal
      );
    },
    [
      appliedCampaignId,
      appliedCompare,
      appliedCustomerId,
      appliedFrom,
      appliedGroupBy,
      appliedLimit,
      appliedOffset,
      appliedTo,
      shouldFetch,
    ]
  );

  const rows: SourceQualityRow[] = data?.rows ?? [];
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
      for (const dim of draftGroupBy) {
        next.append('group_by', dim);
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
      draftGroupBy,
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

  const onToggleGroupBy = useCallback((dim: SourceQualityGroupBy, enabled: boolean) => {
    setDraftGroupBy((current) => {
      if (enabled) {
        if (current.includes(dim)) {
          return current;
        }
        return [...current, dim];
      }
      return current.filter((value) => value !== dim);
    });
  }, []);

  return {
    rows,
    canWrite,
    appliedCustomerId,
    appliedFrom,
    appliedTo,
    appliedCampaignId,
    appliedGroupBy,
    appliedCompare,
    freshness: data?.freshness,
    customerOptions,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    draftCompare,
    draftGroupBy,
    detailMode,
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
    onToggleGroupBy,
    onApplyFilters,
    onPageChange,
  };
}
