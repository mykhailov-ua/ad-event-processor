import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { ApiError } from '@/api/client';
import { runReport } from '@/api/reports_api';
import { RtbOverview } from '@/domains/rtb/rtb_overview';
import { useResource } from '@/hooks/use_resource';
import { defaultReportRange } from '@/lib/report_paths';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';

export function RtbPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const defaultRange = useMemo(() => defaultReportRange('7d'), []);

  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;

  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));

  useEffect(() => {
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
  }, [appliedFrom, appliedTo]);

  const { data, error, fetching } = useResource(
    async (signal) => {
      const params = { from: appliedFrom, to: appliedTo, limit: 50, offset: 0 };
      const [overview, noBid] = await Promise.all([
        runReport('rtb-overview', params, signal),
        runReport('rtb-no-bid-reasons', params, signal),
      ]);
      return {
        overviewRows: overview.rows ?? [],
        noBidRows: noBid.rows ?? [],
        freshness: overview.freshness ?? noBid.freshness,
      };
    },
    [appliedFrom, appliedTo],
  );

  const licenseGated = error instanceof ApiError && error.status === 403;

  const onApply = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    next.set('from', fromDatetimeLocalValue(draftFrom) ?? defaultRange.from);
    next.set('to', fromDatetimeLocalValue(draftTo) ?? defaultRange.to);
    setSearchParams(next, { replace: true });
  }, [defaultRange.from, defaultRange.to, draftFrom, draftTo, searchParams, setSearchParams]);

  return (
    <RtbOverview
      overviewRows={data?.overviewRows ?? []}
      noBidRows={data?.noBidRows ?? []}
      freshness={data?.freshness}
      draftFrom={draftFrom}
      draftTo={draftTo}
      fetching={fetching}
      error={licenseGated ? undefined : error}
      hasSnapshot={data != null || licenseGated}
      licenseGated={licenseGated}
      onDraftFromChange={setDraftFrom}
      onDraftToChange={setDraftTo}
      onApply={onApply}
    />
  );
}
