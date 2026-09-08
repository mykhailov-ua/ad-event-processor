import { useMemo } from 'react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { DashboardKpiStrip } from '@/domains/dashboards/dashboard_kpi_strip';
import { DashboardMultiAxisChart } from '@/domains/dashboards/dashboard_multi_axis_chart';
import { DashboardBreakdownTableSection } from '@/domains/dashboards/dashboard_breakdown_table';
import { ListRefreshBand } from '@/shell/list_refresh_band';
import { PageLayout } from '@/shell/page_layout';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { StubBanner } from '@/shell/stub_banner';
import type { DashboardSeriesPoint } from '@/domains/dashboards/buyer_dashboard_types';
import type { DashboardBreakdownTable } from '@/domains/dashboards/buyer_dashboard_types';
import type { CampaignReportDimension } from '@/domains/campaigns/report/campaign_dashboard_types';
import {
  DEFAULT_CAMPAIGN_DASHBOARD_BREAKDOWN_COLUMNS,
  DEFAULT_CAMPAIGN_DASHBOARD_CHART_METRICS,
  DEFAULT_CAMPAIGN_DASHBOARD_KPI_METRICS,
  buildCampaignDashboardKpiTiles,
  type CampaignDashboardKpis,
} from '@/domains/campaigns/report/campaign_dashboard_metrics';
import type { DashboardBreakdownScope } from '@/domains/dashboards/dashboard_table_column_prefs';
import { buildCampaignsDirectoryHref, campaignEditPath } from '@/lib/campaign_nav';

function campaignDashboardBreakdownScope(
  dimension: CampaignReportDimension
): DashboardBreakdownScope {
  switch (dimension) {
    case 'offers':
      return 'offers';
    case 'landers':
      return 'landers';
    default:
      return 'campaigns';
  }
}

export type CampaignDashboardViewProps = {
  campaignId: string;
  campaignName?: string;
  period?: { from?: string; to?: string };
  dimension: CampaignReportDimension;
  kpis?: CampaignDashboardKpis;
  series?: DashboardSeriesPoint[];
  breakdown?: DashboardBreakdownTable;
  fetching: boolean;
  revalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
  lastUpdatedAt?: string | null;
  onRefresh: () => void;
  onDimensionChange: (dimension: CampaignReportDimension) => void;
};

export function CampaignDashboardView({
  campaignId,
  campaignName,
  period,
  dimension,
  kpis,
  series,
  breakdown,
  fetching,
  revalidating = false,
  error,
  hasSnapshot,
  licenseGated,
  lastUpdatedAt = null,
  onRefresh,
  onDimensionChange,
}: CampaignDashboardViewProps) {
  const kpiTiles = useMemo(
    () => buildCampaignDashboardKpiTiles(kpis, DEFAULT_CAMPAIGN_DASHBOARD_KPI_METRICS),
    [kpis]
  );
  const title = campaignName ? `${campaignName} report` : 'Campaign report';
  const description = campaignName ? campaignId : 'Read-only campaign analytics';
  const campaignsHref = useMemo(
    () =>
      buildCampaignsDirectoryHref({
        from: period?.from,
        to: period?.to,
      }),
    [period?.from, period?.to]
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (licenseGated) {
    return (
      <div className="flex min-h-0 flex-1 flex-col gap-2">
        <StubBanner
          title="Dashboard unavailable"
          message="License or permission denied for campaign dashboard."
        />
      </div>
    );
  }

  if (error && !hasSnapshot) {
    return (
      <div className="flex min-h-0 flex-1 flex-col gap-2">
        <ErrorBlock title="Could not load campaign dashboard" message={error.message} />
      </div>
    );
  }

  return (
    <PageLayout
      description={description}
      headerActions={
        <div className="flex flex-wrap items-center gap-2">
          <Button asChild type="button" variant="secondary">
            <Link to={campaignsHref}>All campaigns</Link>
          </Button>
          <Button asChild type="button" variant="default">
            <Link to={campaignEditPath(campaignId)}>Edit campaign</Link>
          </Button>
        </div>
      }
      controlPanel={
        <div className="grid gap-3">
          <div
            aria-label="Report dimension"
            className="flex flex-wrap items-center gap-2"
            role="group"
          >
            {DIMENSION_PILLS.map((pill) => (
              <Button
                key={pill.id}
                type="button"
                aria-pressed={dimension === pill.id}
                variant={dimension === pill.id ? 'default' : 'secondary'}
                onClick={() => onDimensionChange(pill.id)}
              >
                {pill.label}
              </Button>
            ))}
          </div>
          <div className="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <span>
              {period?.from && period?.to
                ? `${period.from.slice(0, 10)} to ${period.to.slice(0, 10)}`
                : null}
            </span>
            <ListRefreshBand
              ariaLabel="Refresh campaign report"
              lastUpdatedAt={lastUpdatedAt}
              loading={fetching || revalidating}
              onRefresh={onRefresh}
            />
          </div>
        </div>
      }
      title={title}
    >
      <div className="grid min-w-0 gap-4">
        <DashboardKpiStrip tiles={kpiTiles} />
        <DashboardMultiAxisChart
          series={series ?? []}
          chartMetricIds={DEFAULT_CAMPAIGN_DASHBOARD_CHART_METRICS}
        />
        <DashboardBreakdownTableSection
          title={DIMENSION_LABELS[dimension]}
          scope={campaignDashboardBreakdownScope(dimension)}
          table={breakdown}
          columns={DEFAULT_CAMPAIGN_DASHBOARD_BREAKDOWN_COLUMNS}
          emptyDescription={
            dimension === 'paths'
              ? 'Path-level metrics are not available for this period yet.'
              : 'No data in this range.'
          }
        />
      </div>
    </PageLayout>
  );
}

const DIMENSION_PILLS: { id: CampaignReportDimension; label: string }[] = [
  { id: 'default', label: 'My presets' },
  { id: 'paths', label: 'Paths' },
  { id: 'offers', label: 'Offers' },
  { id: 'landers', label: 'Landers' },
  { id: 'rules', label: 'Rules' },
  { id: 'tokens', label: 'Tokens' },
  { id: 'connection', label: 'Connection' },
  { id: 'device', label: 'Device' },
  { id: 'country', label: 'Country' },
];

const DIMENSION_LABELS: Record<CampaignReportDimension, string> = {
  default: 'Campaign overview',
  paths: 'Paths',
  offers: 'Offers',
  landers: 'Landing pages',
  rules: 'Rules',
  tokens: 'Tokens',
  connection: 'Connection',
  device: 'Device',
  country: 'Country',
};
