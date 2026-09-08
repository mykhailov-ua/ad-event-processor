// L3 fraud catalog reports: URL filters and offset pagination per report key meta.
import { useCallback, useEffect, useMemo, useState } from 'react';

import { getFraudCatalogReport } from '@/api/reports_api';
import type { FraudCatalogDimension, FraudCatalogReportRow } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { getFraudCatalogReportMeta } from '@/domains/fraud/fraud_catalog_report_meta';
import type { FraudCatalogReportKey } from '@/domains/fraud/fraud_catalog_report_types';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fetchCustomersComboboxCached } from '@/lib/customers_combobox_cache';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { defaultReportRange } from '@/lib/report_paths';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';

function parseDimension(raw: string | null): FraudCatalogDimension {
  switch (raw) {
    case 'sub1':
    case 'sub2':
    case 'country':
    case 'campaign':
    case 'placement':
      return raw;
    default:
      return 'placement';
  }
}

export function useFraudCatalogReportPageWorkspace(reportKey: FraudCatalogReportKey) {
  const meta = getFraudCatalogReportMeta(reportKey);
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();

  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedDimension = parseDimension(searchParams.get('dimension'));
  const appliedCompare = searchParams.get('compare') === '1';
  const appliedSlice = searchParams.get('slice') === '1';
  const appliedMinDesync = Number.parseInt(searchParams.get('layer_desync_count') ?? '2', 10);
  const appliedLimit = parseListLimit(searchParams.get('limit'), 100);
  const appliedOffset = parseListOffset(searchParams.get('offset'));

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftDimension, setDraftDimension] = useState<FraudCatalogDimension>(appliedDimension);
  const [draftCompare, setDraftCompare] = useState(appliedCompare);
  const [draftSlice, setDraftSlice] = useState(appliedSlice);
  const [draftMinDesync, setDraftMinDesync] = useState(
    Number.isFinite(appliedMinDesync) && appliedMinDesync > 0 ? String(appliedMinDesync) : '2'
  );

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
    setDraftCampaignId(appliedCampaignId);
    setDraftDimension(appliedDimension);
    setDraftCompare(appliedCompare);
    setDraftSlice(appliedSlice);
    setDraftMinDesync(
      Number.isFinite(appliedMinDesync) && appliedMinDesync > 0 ? String(appliedMinDesync) : '2'
    );
  }, [
    appliedCampaignId,
    appliedCompare,
    appliedCustomerId,
    appliedDimension,
    appliedFrom,
    appliedSlice,
    appliedMinDesync,
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

  const shouldFetch = meta.requiresCustomer ? Boolean(appliedCustomerId.trim()) : true;

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
      return getFraudCatalogReport(
        reportKey,
        {
          customer_id: appliedCustomerId || undefined,
          from: appliedFrom,
          to: appliedTo,
          campaign_id: appliedCampaignId || undefined,
          limit: appliedLimit,
          offset: appliedOffset,
          dimension: meta.showDimensionFilter ? appliedDimension : undefined,
          compare: meta.showCompareFilter ? appliedCompare : undefined,
          slice: meta.showSliceFilter ? appliedSlice : undefined,
          layer_desync_count: meta.showMinDesyncFilter ? appliedMinDesync : undefined,
        },
        signal
      );
    },
    [
      appliedCampaignId,
      appliedCompare,
      appliedCustomerId,
      appliedDimension,
      appliedFrom,
      appliedLimit,
      appliedOffset,
      appliedSlice,
      appliedMinDesync,
      appliedTo,
      meta.showCompareFilter,
      meta.showDimensionFilter,
      meta.showMinDesyncFilter,
      meta.showSliceFilter,
      reportKey,
      shouldFetch,
    ]
  );

  const rows: FraudCatalogReportRow[] = data?.rows ?? [];
  const seriesRows: FraudCatalogReportRow[] = data?.series ?? [];
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
      if (meta.showDimensionFilter) {
        next.set('dimension', draftDimension);
      }
      if (meta.showCompareFilter && draftCompare) {
        next.set('compare', '1');
      }
      if (meta.showSliceFilter && draftSlice) {
        next.set('slice', '1');
      }
      if (meta.showMinDesyncFilter) {
        const parsed = Number.parseInt(draftMinDesync, 10);
        if (Number.isFinite(parsed) && parsed > 0) {
          next.set('layer_desync_count', String(parsed));
        }
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
      draftDimension,
      draftFrom,
      draftSlice,
      draftMinDesync,
      draftTo,
      meta.showCompareFilter,
      meta.showDimensionFilter,
      meta.showMinDesyncFilter,
      meta.showSliceFilter,
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
    meta,
    rows,
    seriesRows,
    truncated: data?.truncated ?? false,
    freshness: data?.freshness,
    customerOptions,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    draftDimension,
    draftCompare,
    draftSlice,
    draftMinDesync,
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
    onDraftDimensionChange: setDraftDimension,
    onDraftCompareChange: setDraftCompare,
    onDraftSliceChange: setDraftSlice,
    onDraftMinDesyncChange: setDraftMinDesync,
    onApplyFilters,
    onPageChange,
  };
}
