// RTB overview: typed rtb/* report APIs; URL date range drives refresh.
import { useCallback, useEffect, useMemo, useState } from 'react';

import { ApiError } from '@/api/client';
import {
  getRtbGeoDeviceReport,
  getRtbNoBidReasonsReport,
  getRtbOverviewReport,
} from '@/api/reports_api';
import { useCoalescedBumpRefresh } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { defaultReportRange } from '@/lib/report_paths';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';

export function useRtbPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;

  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));

  useEffect(() => {
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
  }, [appliedFrom, appliedTo]);

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    async (signal) => {
      const params = { from: appliedFrom, to: appliedTo, limit: 50, offset: 0 };
      const [overview, noBid, geoDevice] = await Promise.all([
        getRtbOverviewReport(params, signal),
        getRtbNoBidReasonsReport(params, signal),
        getRtbGeoDeviceReport(params, signal),
      ]);
      return {
        overviewRows: overview.rows ?? [],
        noBidRows: noBid.rows ?? [],
        geoDeviceRows: geoDevice.rows ?? [],
        freshness: overview.freshness ?? noBid.freshness ?? geoDevice.freshness,
      };
    },
    [appliedFrom, appliedTo]
  );

  const licenseGated = error instanceof ApiError && error.status === 403;

  const applyRtbRange = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    next.set('from', fromDatetimeLocalValue(draftFrom) ?? defaultRange.from);
    next.set('to', fromDatetimeLocalValue(draftTo) ?? defaultRange.to);
    replaceSearchParams(next);
  }, [defaultRange.from, defaultRange.to, draftFrom, draftTo, replaceSearchParams, searchParams]);

  const onApply = useCoalescedBumpRefresh(applyRtbRange, fetching);

  return {
    overviewRows: data?.overviewRows ?? [],
    noBidRows: data?.noBidRows ?? [],
    geoDeviceRows: data?.geoDeviceRows ?? [],
    freshness: data?.freshness,
    draftFrom,
    draftTo,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error: licenseGated ? undefined : error,
    hasSnapshot: data != null || licenseGated,
    licenseGated,
    onDraftFromChange: setDraftFrom,
    onDraftToChange: setDraftTo,
    onApply,
  };
}
