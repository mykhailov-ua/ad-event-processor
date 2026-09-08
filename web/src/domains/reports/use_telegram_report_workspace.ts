import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { exportTelegramReport } from '@/api/reports_api';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fetchCustomersComboboxCached } from '@/lib/customers_combobox_cache';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { defaultReportRange } from '@/lib/report_paths';
import { mutationError } from '@/lib/mutation_audit';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import type { TelegramReportFetcher } from '@/domains/reports/telegram_report_meta';

export function useTelegramReportWorkspace<Row>(
  fetchReport: TelegramReportFetcher<Row>,
  enableExport = false
) {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';

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

  const shouldFetch = Boolean(appliedFrom && appliedTo);

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
      return fetchReport(
        {
          customer_id: appliedCustomerId || undefined,
          from: appliedFrom,
          to: appliedTo,
          campaign_id: appliedCampaignId || undefined,
        },
        signal
      );
    },
    [appliedCampaignId, appliedCustomerId, appliedFrom, appliedTo, fetchReport, shouldFetch]
  );

  const [exportingTelegram, setExportingTelegram] = useState(false);
  const [telegramExportError, setTelegramExportError] = useState<Error | undefined>();
  const [telegramExportMessage, setTelegramExportMessage] = useState<string | undefined>();

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
      replaceSearchParams(next);
    },
    [draftCampaignId, draftCustomerId, draftFrom, draftTo, replaceSearchParams]
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
    rows: data?.rows ?? [],
    freshness: data?.freshness,
    extras: data?.extras,
    customerOptions,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    fetching: fetching || listQueryPending,
    listRevalidating,
    error,
    hasSnapshot: data != null || !shouldFetch,
    rangeReady: shouldFetch,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onDraftCampaignIdChange: setDraftCampaignId,
    onApplyFilters,
    enableExport,
    exportingTelegram,
    telegramExportError,
    telegramExportMessage,
    onExportTelegram,
  };
}
