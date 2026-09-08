// L3 fraud reasons report: URL filters, offset pagination, CSV export (EH-SI1).
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { getFraudReasonsReport } from '@/api/reports_api';
import type { FraudReasonRow, FraudReasonsReportKey } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fetchCustomersComboboxCached } from '@/lib/customers_combobox_cache';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { defaultReportRange } from '@/lib/report_paths';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { mutationError } from '@/lib/mutation_audit';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import {
  exportFraudReasonRowsCsv,
  listAllFraudReasonRowsForExport,
} from '@/domains/fraud/fraud_reasons_export';
import { FRAUD_REASONS_REPORT_META } from '@/domains/fraud/fraud_reasons_types';

export function useFraudReasonsPageWorkspace(reportKey: FraudReasonsReportKey) {
  const meta = FRAUD_REASONS_REPORT_META[reportKey];
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();

  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedLimit = parseListLimit(searchParams.get('limit'), 1000);
  const appliedOffset = parseListOffset(searchParams.get('offset'));

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);

  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<Error | undefined>();
  const [exportTruncated, setExportTruncated] = useState(false);

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

  const queryKey = [
    reportKey,
    appliedCustomerId,
    appliedFrom,
    appliedTo,
    appliedCampaignId,
    appliedLimit,
    appliedOffset,
  ];

  const { data, error, fetching, revalidating: listRevalidating } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getFraudReasonsReport(
        reportKey,
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
    queryKey
  );

  const updateQuery = useCallback(
    (patch: {
      customer_id?: string;
      from?: string;
      to?: string;
      campaign_id?: string;
      limit?: number;
      offset?: number;
    }) => {
      const next = new URLSearchParams(searchParams);
      const customerId = patch.customer_id ?? appliedCustomerId;
      const from = patch.from ?? appliedFrom;
      const to = patch.to ?? appliedTo;
      const campaignId = patch.campaign_id ?? appliedCampaignId;
      const limit = patch.limit ?? appliedLimit;
      const offset = patch.offset ?? appliedOffset;

      if (customerId) {
        next.set('customer_id', customerId);
      } else {
        next.delete('customer_id');
      }
      next.set('from', from);
      next.set('to', to);
      if (campaignId) {
        next.set('campaign_id', campaignId);
      } else {
        next.delete('campaign_id');
      }
      next.set('limit', String(limit));
      next.set('offset', String(Math.max(0, offset)));
      replaceSearchParams(next);
    },
    [
      appliedCampaignId,
      appliedCustomerId,
      appliedFrom,
      appliedLimit,
      appliedOffset,
      appliedTo,
      replaceSearchParams,
      searchParams,
    ]
  );

  const onApplyFilters = useCallback(
    (event?: { preventDefault?: () => void }) => {
      event?.preventDefault?.();
      const fromIso = fromDatetimeLocalValue(draftFrom) ?? defaultRange.from;
      const toIso = fromDatetimeLocalValue(draftTo) ?? defaultRange.to;
      updateQuery({
        customer_id: draftCustomerId.trim(),
        from: fromIso,
        to: toIso,
        campaign_id: draftCampaignId.trim(),
        offset: 0,
      });
    },
    [
      defaultRange.from,
      defaultRange.to,
      draftCampaignId,
      draftCustomerId,
      draftFrom,
      draftTo,
      updateQuery,
    ]
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      updateQuery({ offset: Math.max(0, nextOffset) });
    },
    [updateQuery]
  );

  const onExportCsv = useCallback(() => {
    const customerId = appliedCustomerId.trim();
    if (!customerId || exporting) {
      return;
    }
    setExporting(true);
    setExportError(undefined);
    setExportTruncated(false);
    void listAllFraudReasonRowsForExport(
      reportKey,
      {
        customer_id: customerId,
        from: appliedFrom,
        to: appliedTo,
        campaign_id: appliedCampaignId || undefined,
      }
    )
      .then((dataset) => {
        if (dataset.rows.length === 0) {
          toast.message('No rows to export for the current filters.');
          return;
        }
        exportFraudReasonRowsCsv(reportKey, dataset.rows);
        setExportTruncated(dataset.truncated);
        toast.success(
          dataset.truncated
            ? `Exported first ${dataset.rows.length} rows (cap reached).`
            : `Exported ${dataset.rows.length} rows.`
        );
      })
      .catch((err: unknown) => {
        const nextError = mutationError(err);
        setExportError(nextError);
        toast.error(nextError.message);
      })
      .finally(() => {
        setExporting(false);
      });
  }, [appliedCampaignId, appliedCustomerId, appliedFrom, appliedTo, exporting, reportKey]);

  const rows: FraudReasonRow[] = data?.rows ?? [];
  const canGoPrev = appliedOffset > 0;
  const canGoNext = Boolean(data?.next_cursor) || rows.length >= appliedLimit;

  return {
    reportKey,
    title: meta.title,
    description: meta.description,
    rows,
    freshness: data?.freshness,
    customerOptions,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    limit: appliedLimit,
    offset: appliedOffset,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error,
    hasSnapshot: !shouldFetch || data != null,
    exporting,
    exportError,
    exportTruncated,
    canGoPrev,
    canGoNext,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onDraftCampaignIdChange: setDraftCampaignId,
    onApplyFilters,
    onPageChange,
    onExportCsv,
  };
}
