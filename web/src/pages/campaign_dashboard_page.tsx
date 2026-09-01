import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';

import { ApiError } from '@/api/client';
import { getCampaignDashboard } from '@/api/dashboards_api';
import { CampaignDashboardView } from '@/domains/dashboards/campaign_dashboard_view';
import { useBreadcrumbSegmentLabel } from '@/components/system/breadcrumb_context';
import { useResource } from '@/hooks/use_resource';
import { defaultReportRange } from '@/lib/report_paths';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';

export function CampaignDashboardPage() {
  const { id: campaignId = '' } = useParams<{ id: string }>();
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

  const shouldFetch = Boolean(campaignId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getCampaignDashboard(
        campaignId,
        {
          from: appliedFrom,
          to: appliedTo,
        },
        signal,
      );
    },
    [appliedFrom, appliedTo, campaignId, shouldFetch],
  );

  const licenseGated = error instanceof ApiError && error.status === 403;

  const onApply = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    next.set('from', fromDatetimeLocalValue(draftFrom) ?? defaultRange.from);
    next.set('to', fromDatetimeLocalValue(draftTo) ?? defaultRange.to);
    setSearchParams(next, { replace: true });
  }, [defaultRange.from, defaultRange.to, draftFrom, draftTo, searchParams, setSearchParams]);

  const dashboardCampaignName =
    data && typeof data === 'object' && 'campaign_name' in data
      ? String((data as { campaign_name?: string }).campaign_name ?? '')
      : undefined;
  useBreadcrumbSegmentLabel(campaignId || undefined, dashboardCampaignName || undefined);

  return (
    <CampaignDashboardView
      campaignId={campaignId}
      draftFrom={draftFrom}
      draftTo={draftTo}
      payload={data as Record<string, unknown> | undefined}
      fetching={fetching}
      error={licenseGated ? undefined : error}
      hasSnapshot={!shouldFetch || data != null || licenseGated}
      licenseGated={licenseGated}
      onDraftFromChange={setDraftFrom}
      onDraftToChange={setDraftTo}
      onApply={onApply}
    />
  );
}
