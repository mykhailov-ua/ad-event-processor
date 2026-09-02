import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';

import { runEvidencePackReport, runReport, exportTelegramReport } from '@/api/reports_api';
import type { DataFreshness, FraudEvidencePack, ReportMapRow } from '@/api/types';
import { ReportRunner } from '@/domains/reports/report_runner';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import {
  EVIDENCE_PACK_REPORT_KEYS,
  EXPORT_ONLY_REPORT_KEYS,
  defaultReportRange,
} from '@/lib/report_paths';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { deriveColumns } from '@/lib/report_table';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { fetchReportCatalogCached } from '@/lib/report_catalog_cache';

type ReportRunnerSnapshot = {
  rows: ReportMapRow[];
  columns: string[];
  freshness?: DataFreshness;
  nextCursor?: string;
  evidencePack?: FraudEvidencePack;
};

export function ReportRunnerPage() {
  const { key: reportKeyParam } = useParams<{ key: string }>();
  const reportKey = reportKeyParam ?? '';
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();

  const { data: catalog } = useResource((signal) => fetchReportCatalogCached(signal), []);
  const catalogRow = useMemo(
    () => catalog?.rows?.find((row) => row.key === reportKey),
    [catalog?.rows, reportKey],
  );

  const defaultRange = useMemo(
    () => defaultReportRange(catalogRow?.default_range),
    [catalogRow?.default_range],
  );

  const appliedCustomerId =
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedClickId = searchParams.get('click_id') ?? '';
  const appliedLimit = parseListLimit(searchParams.get('limit'));
  const appliedOffset = parseListOffset(searchParams.get('offset'));

  const mode = EXPORT_ONLY_REPORT_KEYS.has(reportKey)
    ? 'export-only'
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
  const [telegramExportMessage, setTelegramExportMessage] = useState<string | undefined>();

  const showTelegramExport = reportKey.includes('telegram');
  const shouldFetch =
    Boolean(reportKey) &&
    mode !== 'export-only' &&
    mode !== 'unsupported' &&
    Boolean(appliedCustomerId || mode === 'evidence') &&
    (mode !== 'evidence' || Boolean(appliedClickId));

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

  const { data, error, fetching } = useResource(
  async (signal) => {
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

    const envelope = await runReport(reportKey, params, signal);
    const rows = envelope.rows ?? [];
    return {
      rows,
      columns: deriveColumns(rows),
      freshness: envelope.freshness,
      nextCursor: envelope.next_cursor,
    } satisfies ReportRunnerSnapshot;
  },
  queryKey,
  );

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

      setSearchParams(next, { replace: true });
    },
    [
      appliedCampaignId,
      appliedClickId,
      appliedCustomerId,
      appliedFrom,
      appliedLimit,
      appliedOffset,
      appliedTo,
      searchParams,
      setSearchParams,
    ],
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
    [updateQuery],
  );

  const onExportTelegram = useCallback(() => {
    const customerId = draftCustomerId.trim() || appliedCustomerId;
    if (!customerId) {
      setTelegramExportMessage('Customer ID is required for export.');
      return;
    }
    setExportingTelegram(true);
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
          downloadUrl ? `Export ready: ${downloadUrl}` : 'Telegram export completed.',
        );
      })
      .catch((err: unknown) => {
        setTelegramExportMessage(
          err instanceof Error ? err.message : 'Telegram export failed.',
        );
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

  return (
    <ReportRunner
      reportKey={reportKey}
      title={catalogRow?.title ?? reportKey}
      description={catalogRow?.description}
      mode={mode}
      rows={data?.rows ?? []}
      columns={data?.columns ?? []}
      freshness={data?.freshness}
      nextCursor={data?.nextCursor}
      evidencePack={data?.evidencePack}
      draftCustomerId={draftCustomerId}
      draftFrom={draftFrom}
      draftTo={draftTo}
      draftCampaignId={draftCampaignId}
      draftClickId={draftClickId}
      limit={appliedLimit}
      offset={appliedOffset}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null || !shouldFetch}
      onDraftCustomerIdChange={setDraftCustomerId}
      onDraftFromChange={setDraftFrom}
      onDraftToChange={setDraftTo}
      onDraftCampaignIdChange={setDraftCampaignId}
      onDraftClickIdChange={setDraftClickId}
      onApplyFilters={onApplyFilters}
      onPageChange={onPageChange}
      showTelegramExport={showTelegramExport}
      exportingTelegram={exportingTelegram}
      telegramExportMessage={telegramExportMessage}
      onExportTelegram={onExportTelegram}
    />
  );
}
