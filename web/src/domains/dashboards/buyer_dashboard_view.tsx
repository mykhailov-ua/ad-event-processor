import { Link } from 'react-router-dom';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import { getRoleDashboard } from '@/api/dashboards_api';
import { useResource } from '@/api/use_resource';
import { DashboardKpiStrip } from '@/domains/dashboards/dashboard_kpi_strip';
import {
  campaignReportBreakdownLink,
  DashboardBreakdownTableSection,
} from '@/domains/dashboards/dashboard_breakdown_table';
import {
  dashboardTablesPairRowClass,
  dashboardTablesPairSlotClass,
  dashboardTablesStackClass,
} from '@/domains/dashboards/dashboard_classes';
import { buildKpiTiles } from '@/domains/dashboards/dashboard_metrics';
import { DashboardMultiAxisChart } from '@/domains/dashboards/dashboard_multi_axis_chart';
import { DashboardRecentClicks } from '@/domains/dashboards/dashboard_recent_clicks';
import type { BuyerDashboardPreferences } from '@/domains/dashboards/dashboard_preferences';
import {
  BREAKDOWN_ENTITY_LABELS,
  DASHBOARD_TOP_CAMPAIGNS_EMPTY_DESCRIPTION,
} from '@/domains/dashboards/dashboard_preferences';
import {
  resolveBuyerDashboardPortfolio,
  resolveDashboardChartSeries,
  isDashboardChartMockEnabled,
} from '@/domains/dashboards/dashboard_series_mock';
import {
  buildDashboardTablesLayout,
  type DashboardBreakdownSectionConfig,
} from '@/domains/dashboards/dashboard_tables_layout';
import type {
  BuyerPortfolio,
  DashboardBreakdownTable,
} from '@/domains/dashboards/buyer_dashboard_types';
import { parseBuyerPortfolio } from '@/domains/dashboards/buyer_dashboard_types';
import { buildCampaignsDirectoryHref, campaignReportPath } from '@/lib/campaign_nav';
import { StubBanner } from '@/shell/stub_banner';
import { cn } from '@/lib/utils';

export type BuyerDashboardViewProps = {
  portfolio: BuyerPortfolio;
  preferences: BuyerDashboardPreferences;
  clickLogHref?: string;
  scopedCampaignId?: string;
  customerId?: string;
  periodFrom?: string;
  periodTo?: string;
};

function resolveScopedBreakdownTable(
  portfolio: BuyerPortfolio | undefined,
  key: 'landers' | 'offers'
): DashboardBreakdownTable | undefined {
  if (!portfolio) {
    return undefined;
  }
  return resolveBuyerDashboardPortfolio(portfolio).breakdowns?.[key];
}

export function BuyerDashboardView({
  portfolio,
  preferences,
  clickLogHref,
  scopedCampaignId,
  customerId,
  periodFrom,
  periodTo,
}: BuyerDashboardViewProps) {
  const [selectedCampaignId, setSelectedCampaignId] = useState<string | null>(null);

  useEffect(() => {
    setSelectedCampaignId(null);
  }, [customerId, periodFrom, periodTo]);

  const resolvedPortfolio = useMemo(() => resolveBuyerDashboardPortfolio(portfolio), [portfolio]);
  const kpiTiles = useMemo(
    () => buildKpiTiles(resolvedPortfolio, preferences.kpiMetrics),
    [preferences.kpiMetrics, resolvedPortfolio]
  );
  const chartSeries = useMemo(
    () => resolveDashboardChartSeries(resolvedPortfolio.series),
    [resolvedPortfolio.series]
  );
  const campaignsDirectoryHref = useMemo(
    () =>
      buildCampaignsDirectoryHref({
        customerId,
        from: periodFrom,
        to: periodTo,
      }),
    [customerId, periodFrom, periodTo]
  );

  const shouldFetchScopedBreakdowns = Boolean(selectedCampaignId && customerId);
  const { data: scopedPortfolioPayload } = useResource(
    (signal) => {
      if (!shouldFetchScopedBreakdowns || !selectedCampaignId || !customerId) {
        return Promise.resolve(undefined);
      }
      return getRoleDashboard(
        'buyer',
        {
          customer_id: customerId,
          campaign_id: selectedCampaignId,
          from: periodFrom,
          to: periodTo,
        },
        signal
      );
    },
    [customerId, periodFrom, periodTo, selectedCampaignId, shouldFetchScopedBreakdowns]
  );
  const scopedPortfolio = useMemo(
    () => parseBuyerPortfolio(scopedPortfolioPayload),
    [scopedPortfolioPayload]
  );

  const landersTable = useMemo(() => {
    if (!selectedCampaignId) {
      return resolvedPortfolio.breakdowns?.landers;
    }
    if (!scopedPortfolio) {
      return { rows: [] };
    }
    return resolveScopedBreakdownTable(scopedPortfolio, 'landers');
  }, [resolvedPortfolio.breakdowns?.landers, scopedPortfolio, selectedCampaignId]);

  const offersTable = useMemo(() => {
    if (!selectedCampaignId) {
      return resolvedPortfolio.breakdowns?.offers;
    }
    if (!scopedPortfolio) {
      return { rows: [] };
    }
    return resolveScopedBreakdownTable(scopedPortfolio, 'offers');
  }, [resolvedPortfolio.breakdowns?.offers, scopedPortfolio, selectedCampaignId]);

  const clearCampaignFilter = useCallback(() => {
    setSelectedCampaignId(null);
  }, []);

  const handleCampaignRowSelect = useCallback((rowId: string) => {
    setSelectedCampaignId(rowId);
  }, []);

  const breakdownSections = useMemo((): DashboardBreakdownSectionConfig[] => {
    const sections: DashboardBreakdownSectionConfig[] = [
      {
        id: 'campaigns',
        title: BREAKDOWN_ENTITY_LABELS.campaigns,
        table: resolvedPortfolio.breakdowns?.campaigns,
        nameLink: campaignReportBreakdownLink,
      },
      {
        id: 'landers',
        title: BREAKDOWN_ENTITY_LABELS.landers,
        table: landersTable,
      },
      {
        id: 'offers',
        title: BREAKDOWN_ENTITY_LABELS.offers,
        table: offersTable,
      },
      {
        id: 'sources',
        title: BREAKDOWN_ENTITY_LABELS.sources,
        table: resolvedPortfolio.breakdowns?.sources,
      },
    ];
    return sections.filter((section) => {
      if (!preferences.breakdownEntities.includes(section.id)) {
        return false;
      }
      if (scopedCampaignId && section.id === 'campaigns') {
        return false;
      }
      return true;
    });
  }, [
    landersTable,
    offersTable,
    preferences.breakdownEntities,
    resolvedPortfolio.breakdowns,
    scopedCampaignId,
  ]);

  const tablesLayout = useMemo(
    () => buildDashboardTablesLayout(breakdownSections, true),
    [breakdownSections]
  );

  const clearFilterAction = selectedCampaignId ? (
    <Button type="button" variant="link" onClick={clearCampaignFilter}>
      Сбросить фильтр
    </Button>
  ) : null;

  const renderBreakdownSection = (
    section: DashboardBreakdownSectionConfig,
    options?: { fillContainer?: boolean; fillHeight?: boolean }
  ) => (
    <DashboardBreakdownTableSection
      key={section.id}
      columns={preferences.breakdownColumns}
      emptyDescription={
        section.id === 'campaigns' ? DASHBOARD_TOP_CAMPAIGNS_EMPTY_DESCRIPTION : undefined
      }
      enableCampaignSort={section.id === 'campaigns'}
      fillContainer={options?.fillContainer ?? true}
      fillHeight={options?.fillHeight ?? false}
      meta={
        section.id === 'campaigns' ? (
          <Link className="text-sm text-primary hover:underline" to={campaignsDirectoryHref}>
            Manage campaigns
          </Link>
        ) : undefined
      }
      nameLink={section.nameLink}
      scope={section.id}
      selectedRowId={section.id === 'campaigns' ? selectedCampaignId : undefined}
      table={section.table}
      title={section.title}
      titleSuffix={section.id === 'landers' ? clearFilterAction : undefined}
      onRowSelect={section.id === 'campaigns' ? handleCampaignRowSelect : undefined}
    />
  );

  const renderGridSlot = (
    slot: (typeof tablesLayout.pairRows)[number][number],
    options?: { fillHeight?: boolean }
  ) => {
    if (slot.kind === 'recent_clicks') {
      return (
        <DashboardRecentClicks
          key="recent_clicks"
          columns={preferences.recentClickColumns}
          events={resolvedPortfolio.recent_clicks ?? []}
          fillContainer
          fillHeight={options?.fillHeight ?? false}
          viewAllHref={clickLogHref}
        />
      );
    }
    return renderBreakdownSection(slot.section, {
      fillContainer: true,
      fillHeight: options?.fillHeight ?? false,
    });
  };

  const hasTables = tablesLayout.top != null || tablesLayout.pairRows.some((row) => row.length > 0);

  return (
    <div className="grid min-w-0 gap-3">
      {isDashboardChartMockEnabled() ? (
        <StubBanner
          message="KPIs, charts, and tables use fixture series from ?chart_mock=1. Set chart_mock=0 or remove the parameter for live API metrics."
          title="Synthetic dashboard data"
        />
      ) : null}

      {scopedCampaignId ? (
        <p className="m-0 text-sm text-muted-foreground">
          Scoped to one campaign.{' '}
          <Link className="text-primary hover:underline" to={campaignReportPath(scopedCampaignId)}>
            Open campaign report
          </Link>
          {' | '}
          <Link className="text-primary hover:underline" to={campaignsDirectoryHref}>
            Manage campaigns
          </Link>
        </p>
      ) : null}

      <DashboardKpiStrip tiles={kpiTiles} />

      <DashboardMultiAxisChart series={chartSeries} chartMetricIds={preferences.chartMetrics} />

      {hasTables ? (
        <div className={dashboardTablesStackClass}>
          {tablesLayout.top
            ? renderBreakdownSection(tablesLayout.top, { fillContainer: true })
            : null}

          {tablesLayout.pairRows.map((pair, rowIndex) => (
            <div key={rowIndex} className={dashboardTablesPairRowClass}>
              {pair.map((slot) => (
                <div
                  key={slot.kind === 'breakdown' ? slot.section.id : slot.kind}
                  className={cn(dashboardTablesPairSlotClass, pair.length === 1 && 'lg:col-span-2')}
                >
                  {renderGridSlot(slot, { fillHeight: true })}
                </div>
              ))}
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}
