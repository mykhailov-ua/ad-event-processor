// L3 report runner: catalog cached; applied filters from URL; draft* until commit; shouldFetch gates evidence vs table vs export-only modes.
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';

import { ApiError } from '@/api/client';
import { runEvidencePackReport, runReport, exportTelegramReport } from '@/api/reports_api';
import { getCampaignStats } from '@/api/campaigns_api';
import type { CampaignStats, DataFreshness, FraudEvidencePack, ReportMapRow } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import {
  CAMPAIGN_STATS_REPORT_KEYS,
  EVIDENCE_PACK_REPORT_KEYS,
  EXPORT_ONLY_REPORT_KEYS,
  defaultReportRange,
} from '@/lib/report_paths';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { deriveColumns } from '@/lib/report_table';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { fetchReportCatalogCached } from '@/lib/report_catalog_cache';
import { mutationError } from '@/lib/mutation_audit';
import type { ReportRunnerProps } from '@/domains/reports/report_runner';

type ReportRunnerSnapshot = {
  rows: ReportMapRow[];
  columns: string[];
  freshness?: DataFreshness;
  nextCursor?: string;
  evidencePack?: FraudEvidencePack;
  campaignStats?: CampaignStats;
};

export function useReportRunnerPage(reportKeyOverride?: string) {
  const { key: reportKeyParam } = useParams<{ key: string }>();
  const reportKey = reportKeyOverride ?? reportKeyParam ?? '';
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();

  const { data: catalog } = useResource((signal) => fetchReportCatalogCached(signal), []);
  const catalogRow = useMemo(
    () => catalog?.rows?.find((row) => row.key === reportKey),
    [catalog?.rows, reportKey]
  );

  const defaultRange = useMemo(
    () => defaultReportRange(catalogRow?.default_range),
    [catalogRow?.default_range]
  );

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedClickId = searchParams.get('click_id') ?? '';
  const appliedLimit = parseListLimit(searchParams.get('limit'));
  const appliedOffset = parseListOffset(searchParams.get('offset'));

  const mode: ReportRunnerProps['mode'] = EXPORT_ONLY_REPORT_KEYS.has(reportKey)
    ? 'export-only'
    : CAMPAIGN_STATS_REPORT_KEYS.has(reportKey)
      ? 'campaign-stats'
      : EVIDENCE_PACK_REPORT_KEYS.has(reportKey)
        ? 'evidence'
        : reportKey
          ? 'table'
          : 'unsupported';

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

  const [exportingTelegram, setExportingTelegram] = useState(false);
  const [telegramExportError, setTelegramExportError] = useState<Error | undefined>();
  const [telegramExportMessage, setTelegramExportMessage] = useState<string | undefined>();

  const showTelegramExport = reportKey === 'telegram';
  const shouldFetch =
    Boolean(reportKey) &&
    mode !== 'export-only' &&
    mode !== 'unsupported' &&
    (mode === 'campaign-stats'
      ? Boolean(appliedCampaignId)
      : Boolean(appliedCustomerId || mode === 'evidence') &&
        (mode !== 'evidence' || Boolean(appliedClickId)));

  const queryKey = [
    reportKey,
    appliedCustomerId,
    appliedFrom,
    appliedTo,
    appliedCampaignId,
    appliedClickId,
    appliedLimit,
    appliedOffset,
    mode,
  ];

  const { data, error, fetching, revalidating: listRevalidating } = useResource(async (signal) => {
    if (!shouldFetch) {
      return undefined;
    }

    const params = {
      customer_id: appliedCustomerId || undefined,
      from: appliedFrom,
      to: appliedTo,
      campaign_id: appliedCampaignId || undefined,
      click_id: appliedClickId || undefined,
      limit: appliedLimit,
      offset: appliedOffset,
    };

    if (mode === 'evidence') {
      const evidencePack = await runEvidencePackReport(reportKey, params, signal);
      return {
        rows: [],
        columns: [],
        evidencePack,
      } satisfies ReportRunnerSnapshot;
    }

    if (mode === 'campaign-stats') {
      const campaignStats = await getCampaignStats(
        appliedCampaignId,
        {
          from: appliedFrom,
          to: appliedTo,
        },
        signal
      );
      return {
        rows: [],
        columns: [],
        campaignStats,
      } satisfies ReportRunnerSnapshot;
    }

    const envelope = await runReport(reportKey, params, signal);
    const rows = envelope.rows ?? [];
    return {
      rows,
      columns: deriveColumns(rows),
      freshness: envelope.freshness,
      nextCursor: envelope.next_cursor,
    } satisfies ReportRunnerSnapshot;
  }, queryKey);

  const licenseGated =
    Boolean(catalogRow?.license_gated) && error instanceof ApiError && error.status === 403;

  const updateQuery = useCallback(
    (patch: {
      customer_id?: string;
      from?: string;
      to?: string;
      campaign_id?: string;
      click_id?: string;
      limit?: number;
      offset?: number;
    }) => {
      const next = new URLSearchParams(searchParams);

      const customerId = patch.customer_id ?? appliedCustomerId;
      const from = patch.from ?? appliedFrom;
      const to = patch.to ?? appliedTo;
      const campaignId = patch.campaign_id ?? appliedCampaignId;
      const clickId = patch.click_id ?? appliedClickId;
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
      if (clickId) {
        next.set('click_id', clickId);
      } else {
        next.delete('click_id');
      }
      next.set('limit', String(limit));
      next.set('offset', String(Math.max(0, offset)));

      replaceSearchParams(next);
    },
    [
      appliedCampaignId,
      appliedClickId,
      appliedCustomerId,
      appliedFrom,
      appliedLimit,
      appliedOffset,
      appliedTo,
      replaceSearchParams,
      searchParams,
    ]
  );

  const onApplyFilters = useCallback(() => {
    const fromIso = fromDatetimeLocalValue(draftFrom) ?? defaultRange.from;
    const toIso = fromDatetimeLocalValue(draftTo) ?? defaultRange.to;
    updateQuery({
      customer_id: draftCustomerId.trim(),
      from: fromIso,
      to: toIso,
      campaign_id: draftCampaignId.trim(),
      click_id: draftClickId.trim(),
      offset: 0,
    });
  }, [
    defaultRange.from,
    defaultRange.to,
    draftCampaignId,
    draftClickId,
    draftCustomerId,
    draftFrom,
    draftTo,
    updateQuery,
  ]);

  const onPageChange = useCallback(
    (nextOffset: number) => {
      updateQuery({ offset: Math.max(0, nextOffset) });
    },
    [updateQuery]
  );

  const onExportTelegram = useCallback(() => {
    const customerId = draftCustomerId.trim() || appliedCustomerId;
    if (!customerId) {
      setTelegramExportMessage('Customer ID is required for export.');
      return;
    }
    setExportingTelegram(true);
    setTelegramExportError(undefined);
    setTelegramExportMessage(undefined);
    void exportTelegramReport({
      customer_id: customerId,
      from: fromDatetimeLocalValue(draftFrom) ?? appliedFrom,
      to: fromDatetimeLocalValue(draftTo) ?? appliedTo,
      campaign_id: draftCampaignId.trim() || appliedCampaignId || undefined,
    })
      .then((result) => {
        const downloadUrl =
          typeof result.download_url === 'string' ? result.download_url : undefined;
        setTelegramExportMessage(
          downloadUrl ? `Export ready: ${downloadUrl}` : 'Telegram export completed.'
        );
        toast.success('Telegram export completed');
      })
      .catch((err: unknown) => {
        const nextError = mutationError(err);
        setTelegramExportError(nextError);
        toast.error(nextError.message);
      })
      .finally(() => {
        setExportingTelegram(false);
      });
  }, [
    appliedCampaignId,
    appliedCustomerId,
    appliedFrom,
    appliedTo,
    draftCampaignId,
    draftCustomerId,
    draftFrom,
    draftTo,
  ]);

  return {
    reportKey,
    title: catalogRow?.title ?? reportKey,
    description: catalogRow?.description,
    mode,
    rows: data?.rows ?? [],
    columns: data?.columns ?? [],
    freshness: data?.freshness,
    nextCursor: data?.nextCursor,
    evidencePack: data?.evidencePack,
    campaignStats: data?.campaignStats,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    draftClickId,
    limit: appliedLimit,
    offset: appliedOffset,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error: licenseGated ? undefined : error,
    hasSnapshot: data != null || !shouldFetch || licenseGated,
    licenseGated,
    licenseFeatureKey: catalogRow?.feature_key,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onDraftCampaignIdChange: setDraftCampaignId,
    onDraftClickIdChange: setDraftClickId,
    onApplyFilters,
    onPageChange,
    showTelegramExport,
    exportingTelegram,
    telegramExportError,
    telegramExportMessage,
    onExportTelegram,
  };
}
