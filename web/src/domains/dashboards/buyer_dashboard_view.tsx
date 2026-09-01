import { DashboardKpiStrip } from '@/domains/dashboards/dashboard_kpi_strip';
import {
  campaignBreakdownLink,
  DashboardBreakdownTableSection,
} from '@/domains/dashboards/dashboard_breakdown_table';
import { buildKpiTiles } from '@/domains/dashboards/dashboard_metrics';
import { DashboardMultiAxisChart } from '@/domains/dashboards/dashboard_multi_axis_chart';
import { DashboardRecentClicks } from '@/domains/dashboards/dashboard_recent_clicks';
import type { BuyerDashboardPreferences } from '@/domains/dashboards/dashboard_preferences';
import { resolveBuyerDashboardPortfolio, resolveDashboardChartSeries } from '@/domains/dashboards/dashboard_series_mock';
import type { BuyerPortfolio } from '@/domains/dashboards/buyer_dashboard_types';
import { useMemo } from 'react';

export type BuyerDashboardViewProps = {
  portfolio: BuyerPortfolio;
  preferences: BuyerDashboardPreferences;
  clickLogHref?: string;
};

export function BuyerDashboardView({ portfolio, preferences, clickLogHref }: BuyerDashboardViewProps) {
  const resolvedPortfolio = useMemo(() => resolveBuyerDashboardPortfolio(portfolio), [portfolio]);
  const kpiTiles = useMemo(
    () => buildKpiTiles(resolvedPortfolio, preferences.kpiMetrics),
    [preferences.kpiMetrics, resolvedPortfolio],
  );
  const chartSeries = useMemo(
    () => resolveDashboardChartSeries(resolvedPortfolio.series),
    [resolvedPortfolio.series],
  );

  const breakdownSections = useMemo(() => {
    const sections = [
      {
        id: 'campaigns' as const,
        title: 'Campaigns',
        table: resolvedPortfolio.breakdowns?.campaigns,
        nameLink: campaignBreakdownLink,
      },
      {
        id: 'landers' as const,
        title: 'Landing pages',
        table: resolvedPortfolio.breakdowns?.landers,
        emptyLabel: 'Landing page breakdown is not available yet.',
      },
      {
        id: 'offers' as const,
        title: 'Offers',
        table: resolvedPortfolio.breakdowns?.offers,
        emptyLabel: 'Offer breakdown is not available yet.',
      },
      {
        id: 'sources' as const,
        title: 'Sources',
        table: resolvedPortfolio.breakdowns?.sources,
      },
    ];
    return sections.filter((section) => preferences.breakdownEntities.includes(section.id));
  }, [preferences.breakdownEntities, resolvedPortfolio.breakdowns]);

  return (
    <div className="grid min-w-0 gap-4">
      <DashboardKpiStrip tiles={kpiTiles} />

      <DashboardMultiAxisChart series={chartSeries} chartMetricIds={preferences.chartMetrics} />

      <div className="grid min-w-0 gap-4 lg:grid-cols-2">
        {breakdownSections.map((section) => (
          <DashboardBreakdownTableSection
            key={section.id}
            columns={preferences.breakdownColumns}
            emptyLabel={section.emptyLabel}
            nameLink={section.nameLink}
            table={section.table}
            title={section.title}
          />
        ))}
      </div>

      <DashboardRecentClicks
        columns={preferences.recentClickColumns}
        events={resolvedPortfolio.recent_clicks ?? []}
        viewAllHref={clickLogHref}
      />
    </div>
  );
}
