import { useCallback, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';

import type { CampaignStats } from '@/api/types';
import {
  HourlyTrendChart,
  MetricsSection,
} from '@/domains/campaigns/list/campaign_metrics_shared';
import { ReportKpiGrid } from '@/domains/reports/report_kpi_grid';
import {
  buildReportColumnOverviewFields,
  directoryOperateRowsIndexed,
  directoryRecordMapIndexed,
  reportRowLabelFromColumn,
  type ReportDirectoryColumn,
} from '@/domains/reports/report_select_overview';
import type { CampaignStatsGranularity } from '@/domains/reports/use_campaign_stats_report_workspace';
import { FilterApplyButton } from '@/shell/action_buttons';
import { DirectorySelectOverviewTable } from '@/shell/directory_select_overview_table';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { displayCount, displayMoneyDecimal, displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';

type CampaignDailyRow = NonNullable<CampaignStats['daily']>[number];

const DAILY_BUCKET_COLUMNS: ReportDirectoryColumn<CampaignDailyRow>[] = [
  {
    id: 'day',
    label: 'Day',
    cell: (row) => displayTimestamp(row.day) || '-',
  },
  {
    id: 'impressions',
    label: 'Impressions',
    cell: (row) => displayCount(row.impressions) || '-',
  },
  {
    id: 'clicks',
    label: 'Clicks',
    cell: (row) => displayCount(row.clicks) || '-',
  },
  {
    id: 'conversions',
    label: 'Conversions',
    cell: (row) => displayCount(row.conversions) || '-',
  },
];

function dailyBucketId(row: CampaignDailyRow, index: number): string {
  return row.day ?? `daily-${index}`;
}

export type CampaignStatsReportDirectoryProps = {
  stats?: CampaignStats;
  draftCampaignId: string;
  draftFrom: string;
  draftTo: string;
  draftGranularity: CampaignStatsGranularity;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  campaignReady: boolean;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftGranularityChange: (value: CampaignStatsGranularity) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
};

export function CampaignStatsReportDirectory({
  stats,
  draftCampaignId,
  draftFrom,
  draftTo,
  draftGranularity,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  campaignReady,
  onDraftCampaignIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftGranularityChange,
  onApplyFilters,
}: CampaignStatsReportDirectoryProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const dailyRows = stats?.daily ?? [];
  const nameColumn = DAILY_BUCKET_COLUMNS[0];

  const recordById = useMemo(
    () => directoryRecordMapIndexed(dailyRows, (row, index) => dailyBucketId(row, index)),
    [dailyRows]
  );
  const operateRows = useMemo(
    () =>
      directoryOperateRowsIndexed(
        dailyRows,
        (row, index) => dailyBucketId(row, index),
        (row) => reportRowLabelFromColumn(row, nameColumn, 'Day')
      ),
    [dailyRows, nameColumn]
  );

  const buildOverviewFields = useCallback(
    (row: CampaignDailyRow) =>
      buildReportColumnOverviewFields(row, DAILY_BUCKET_COLUMNS, {
        skipColumnIds: [nameColumn.id],
      }),
    [nameColumn.id]
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load campaign stats" message={error.message} />;
  }

  const badge = stats?.stale ? (
    <Badge variant="secondary">Stale ({stats.source ?? 'unknown'})</Badge>
  ) : stats?.consistency ? (
    <span className={adminTypography.bodyMuted}>{stats.consistency}</span>
  ) : null;

  return (
    <PageLayout
      badge={badge}
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Reports catalog</Link>
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={onApplyFilters}>
              <FilterField htmlFor="campaign-stats-id" label="Campaign ID" wide>
                <Input
                  id="campaign-stats-id"
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor="campaign-stats-from" label="From">
                <DatetimePicker
                  id="campaign-stats-from"
                  label="From"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor="campaign-stats-to" label="To">
                <DatetimePicker
                  id="campaign-stats-to"
                  label="To"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              <FilterField htmlFor="campaign-stats-granularity" label="Granularity">
                <Select
                  value={draftGranularity}
                  onValueChange={(value) =>
                    onDraftGranularityChange(value === 'day' ? 'day' : 'hour')
                  }
                >
                  <SelectTrigger id="campaign-stats-granularity">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="hour">Hourly</SelectItem>
                    <SelectItem value="day">Daily</SelectItem>
                  </SelectContent>
                </Select>
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description="Hourly and daily delivery stats for one campaign."
      title="Campaign stats"
    >
      {!campaignReady ? (
        <EmptyState title="Campaign ID required" description="Enter a campaign ID and apply." />
      ) : !stats ? (
        <EmptyState title="No data" description="No stats returned for this campaign window." />
      ) : (
        <TableHost className="grid w-full gap-4">
          <ReportKpiGrid
            items={[
              { label: 'Campaign', value: stats.campaign_id || '-' },
              { label: 'Window from', value: displayTimestamp(stats.from) || '-' },
              { label: 'Window to', value: displayTimestamp(stats.to) || '-' },
              { label: 'Spend', value: displayMoneyDecimal(stats.current_spend) || '-' },
              {
                label: 'Impressions',
                value: displayCount(stats.metrics?.impressions) || '-',
              },
              { label: 'Clicks', value: displayCount(stats.metrics?.clicks) || '-' },
              {
                label: 'Conversions',
                value: displayCount(stats.metrics?.conversions) || '-',
              },
              { label: 'Granularity', value: stats.granularity || '-' },
            ]}
          />

          <MetricsSection title="Hourly trend">
            <HourlyTrendChart buckets={stats.hourly ?? []} loading={fetching} />
          </MetricsSection>

          {dailyRows.length > 0 ? (
            <MetricsSection title="Daily buckets">
              <TableHost className="w-full">
                <DirectorySelectOverviewTable
                  buildOverviewFields={buildOverviewFields}
                  disabled={fetching}
                  nameColumnLabel={nameColumn.label}
                  overviewTitle={(row) =>
                    String(reportRowLabelFromColumn(row, nameColumn, 'Daily bucket'))
                  }
                  recordById={recordById}
                  revalidating={listRevalidating}
                  rows={operateRows}
                  selectedId={selectedId}
                  onSelectedIdChange={setSelectedId}
                />
              </TableHost>
            </MetricsSection>
          ) : null}
        </TableHost>
      )}
    </PageLayout>
  );
}
