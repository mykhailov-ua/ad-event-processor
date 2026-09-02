import { useCallback, useMemo, useState } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';

import { ApiError } from '@/api/client';
import { getCampaign } from '@/api/campaigns_api';
import { getCampaignDashboard } from '@/api/dashboards_api';
import { getFlow } from '@/api/flows_api';
import { CampaignDashboardView } from '@/domains/dashboards/campaign_dashboard_view';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';
import { defaultReportRange } from '@/lib/report_paths';

export function CampaignDashboardPage() {
  const { id: campaignId = '' } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();

  const defaultRange = useMemo(() => defaultReportRange('7d'), []);
  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;

  const [draftQ, setDraftQ] = useState('');
  const [refreshNonce, setRefreshNonce] = useState(0);

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
    [appliedFrom, appliedTo, campaignId, refreshNonce, shouldFetch],
  );

  const { data: campaignData } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getCampaign(campaignId, signal);
    },
    [campaignId, shouldFetch],
  );

  const flowId = campaignData?.flow_id?.trim() ?? '';

  const { data: flowData } = useResource(
    (signal) => {
      if (!flowId) {
        return Promise.resolve(undefined);
      }
      return getFlow(flowId, signal);
    },
    [flowId],
  );

  const licenseGated = error instanceof ApiError && error.status === 403;

  const onRefresh = useCallback(() => {
    setRefreshNonce((value) => value + 1);
  }, []);

  const dashboardCampaignName =
    data && typeof data === 'object' && 'campaign_name' in data
      ? String((data as { campaign_name?: string }).campaign_name ?? '')
      : campaignData?.name;
  useBreadcrumbSegmentLabel(campaignId || undefined, dashboardCampaignName || undefined);

  return (
    <CampaignDashboardView
      campaignId={campaignId}
      campaignName={dashboardCampaignName}
      draftQ={draftQ}
      flow={flowData}
      payload={data as Record<string, unknown> | undefined}
      fetching={fetching}
      error={licenseGated ? undefined : error}
      hasSnapshot={!shouldFetch || data != null || licenseGated}
      licenseGated={licenseGated}
      onDraftQChange={setDraftQ}
      onRefresh={onRefresh}
    />
  );
}
