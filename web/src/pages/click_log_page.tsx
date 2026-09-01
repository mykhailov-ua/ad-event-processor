import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { listCustomers } from '@/api/customers_api';
import { getClickLogReport } from '@/api/reports_api';
import type { CustomerComboboxOption } from '@/components/system/customer_combobox';
import { ClickLogDirectory } from '@/domains/reports/click_log_directory';
import { useResource } from '@/hooks/use_resource';
import { useSession } from '@/hooks/use_session';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { defaultReportRange } from '@/lib/report_paths';

export function ClickLogPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();

  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedCustomerId =
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedClickId = searchParams.get('click_id') ?? '';
  const appliedCursor = searchParams.get('cursor') ?? '';

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));
  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftClickId, setDraftClickId] = useState(appliedClickId);
  const [cursorStack, setCursorStack] = useState<string[]>([]);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
    setDraftCampaignId(appliedCampaignId);
    setDraftClickId(appliedClickId);
  }, [appliedCampaignId, appliedClickId, appliedCustomerId, appliedFrom, appliedTo]);

  useEffect(() => {
    setCursorStack([]);
  }, [appliedCampaignId, appliedClickId, appliedCustomerId, appliedFrom, appliedTo]);

  const { data: customersData } = useResource(
    (signal) => listCustomers({ limit: 100, offset: 0, sort: 'name', order: 'asc' }, signal),
    [],
  );

  const customerOptions = useMemo((): CustomerComboboxOption[] => {
    return (customersData?.items ?? [])
      .filter((customer) => customer.id)
      .map((customer) => ({
        id: customer.id as string,
        name: customer.name ?? customer.id ?? '',
      }));
  }, [customersData?.items]);

  const shouldFetch = Boolean(appliedCustomerId.trim());

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getClickLogReport(
        {
          customer_id: appliedCustomerId,
          from: appliedFrom,
          to: appliedTo,
          campaign_id: appliedCampaignId || undefined,
          click_id: appliedClickId || undefined,
          cursor: appliedCursor || undefined,
        },
        signal,
      );
    },
    [
      appliedCampaignId,
      appliedClickId,
      appliedCursor,
      appliedCustomerId,
      appliedFrom,
      appliedTo,
      shouldFetch,
    ],
  );

  const updateQuery = useCallback(
    (patch: {
      customer_id?: string;
      from?: string;
      to?: string;
      campaign_id?: string;
      click_id?: string;
      cursor?: string;
      resetCursor?: boolean;
    }) => {
      const next = new URLSearchParams();
      const customerId = patch.customer_id ?? appliedCustomerId;
      const from = patch.from ?? appliedFrom;
      const to = patch.to ?? appliedTo;
      const campaignId = patch.campaign_id ?? appliedCampaignId;
      const clickId = patch.click_id ?? appliedClickId;
      const cursor = patch.resetCursor ? '' : (patch.cursor ?? appliedCursor);

      if (customerId) {
        next.set('customer_id', customerId);
      }
      next.set('from', from);
      next.set('to', to);
      if (campaignId) {
        next.set('campaign_id', campaignId);
      }
      if (clickId) {
        next.set('click_id', clickId);
      }
      if (cursor) {
        next.set('cursor', cursor);
      }
      setSearchParams(next, { replace: true });
    },
    [
      appliedCampaignId,
      appliedClickId,
      appliedCursor,
      appliedCustomerId,
      appliedFrom,
      appliedTo,
      setSearchParams,
    ],
  );

  const onApplyFilters = useCallback(
    (event?: { preventDefault?: () => void }) => {
      event?.preventDefault?.();
      const fromIso = fromDatetimeLocalValue(draftFrom) ?? defaultRange.from;
      const toIso = fromDatetimeLocalValue(draftTo) ?? defaultRange.to;
      setCursorStack([]);
      updateQuery({
        customer_id: draftCustomerId.trim(),
        from: fromIso,
        to: toIso,
        campaign_id: draftCampaignId.trim(),
        click_id: draftClickId.trim(),
        resetCursor: true,
      });
    },
    [
      defaultRange.from,
      defaultRange.to,
      draftCampaignId,
      draftClickId,
      draftCustomerId,
      draftFrom,
      draftTo,
      updateQuery,
    ],
  );

  const onNextPage = useCallback(() => {
    const nextCursor = data?.next_cursor;
    if (!nextCursor) {
      return;
    }
    setCursorStack((stack) => [...stack, appliedCursor]);
    updateQuery({ cursor: nextCursor });
  }, [appliedCursor, data?.next_cursor, updateQuery]);

  const onPrevPage = useCallback(() => {
    setCursorStack((stack) => {
      const nextStack = [...stack];
      const previousCursor = nextStack.pop() ?? '';
      updateQuery({ cursor: previousCursor, resetCursor: !previousCursor });
      return nextStack;
    });
  }, [updateQuery]);

  const timelineMode = Boolean(appliedClickId.trim());

  return (
    <ClickLogDirectory
      events={data?.events ?? []}
      postbacks={data?.postbacks ?? []}
      freshness={data?.freshness}
      nextCursor={data?.next_cursor}
      timelineMode={timelineMode}
      customerOptions={customerOptions}
      draftCustomerId={draftCustomerId}
      draftFrom={draftFrom}
      draftTo={draftTo}
      draftCampaignId={draftCampaignId}
      draftClickId={draftClickId}
      cursor={appliedCursor}
      fetching={fetching}
      error={error}
      hasSnapshot={!shouldFetch || data != null}
      onDraftCustomerIdChange={setDraftCustomerId}
      onDraftFromChange={setDraftFrom}
      onDraftToChange={setDraftTo}
      onDraftCampaignIdChange={setDraftCampaignId}
      onDraftClickIdChange={setDraftClickId}
      onApplyFilters={onApplyFilters}
      onNextPage={onNextPage}
      onPrevPage={onPrevPage}
      canGoPrev={cursorStack.length > 0}
    />
  );
}
