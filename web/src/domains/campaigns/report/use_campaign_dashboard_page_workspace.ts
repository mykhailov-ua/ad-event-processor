// L3 per-campaign dashboard: URL range + breakdown dimension/sort; GET /dashboards/campaign/{id}.
import { useCallback, useMemo } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';

import { ApiError } from '@/api/client';
import { getCampaignDashboard } from '@/api/dashboards_api';
import type { CampaignDashboardKpis } from '@/domains/campaigns/report/campaign_dashboard_metrics';
import {
  parseCampaignReportDimension,
  parseCampaignReportOrder,
  parseCampaignReportSort,
  type CampaignReportDimension,
} from '@/domains/campaigns/report/campaign_dashboard_types';
import type {
  DashboardBreakdownTable,
  DashboardSeriesPoint,
} from '@/domains/dashboards/buyer_dashboard_types';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { defaultReportRange } from '@/lib/report_paths';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';

export function useCampaignDashboardPageWorkspace() {
  const { id: campaignId = '' } = useParams<{ id: string }>();
  const [searchParams, setSearchParams] = useSearchParams();

  const defaultRange = useMemo(() => defaultReportRange('7d'), []);
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;
  const appliedDimension = parseCampaignReportDimension(searchParams.get('dimension'));
  const appliedQ = searchParams.get('q') ?? '';
  const appliedSort = parseCampaignReportSort(searchParams.get('sort'));
  const appliedOrderDesc = parseCampaignReportOrder(searchParams.get('order'));
  const shouldFetch = Boolean(campaignId);

  const { refreshToken, bumpRefresh } = useRefreshToken();

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
          dimension: appliedDimension,
          q: appliedQ || undefined,
          sort: appliedSort,
          order: appliedOrderDesc ? 'desc' : 'asc',
        },
        signal
      );
    },
    [
      appliedDimension,
      appliedFrom,
      appliedOrderDesc,
      appliedQ,
      appliedSort,
      appliedTo,
      campaignId,
      refreshToken,
      shouldFetch,
    ]
  );

  const licenseGated = error instanceof ApiError && error.status === 403;

  const onRefresh = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const onDimensionChange = useCallback(
    (dimension: CampaignReportDimension) => {
      const next = new URLSearchParams(searchParams);
      next.set('dimension', dimension);
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const payload = useMemo(() => {
    if (!data || typeof data !== 'object') {
      return undefined;
    }
    return data as {
      campaign_id?: string;
      campaign_name?: string;
      period?: { from?: string; to?: string };
      kpis?: CampaignDashboardKpis;
      series?: DashboardSeriesPoint[];
      breakdown?: DashboardBreakdownTable;
    };
  }, [data]);

  useBreadcrumbSegmentLabel(campaignId || undefined, payload?.campaign_name || undefined);

  return {
    campaignId,
    campaignName: payload?.campaign_name,
    period: payload?.period,
    dimension: appliedDimension,
    kpis: payload?.kpis,
    series: payload?.series,
    breakdown: payload?.breakdown,
    fetching,
    error: licenseGated ? undefined : error,
    hasSnapshot: !shouldFetch || data != null || licenseGated,
    licenseGated,
    onDimensionChange,
    onRefresh,
  };
}
