import { Link } from 'react-router-dom';

import type { CampaignStats } from '@/api/types';
import {
  HourlyTrendChart,
  MetricTile,
  MetricsSection,
} from '@/domains/campaigns/list/campaign_metrics_shared';
import { ReportKpiGrid } from '@/domains/reports/report_kpi_grid';
import type { CampaignStatsGranularity } from '@/domains/reports/use_campaign_stats_report_workspace';
import { FilterApplyButton } from '@/shell/action_buttons';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
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
import { cn } from '@/lib/utils';

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
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={4} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load campaign stats" message={error.message} />;
  }

  const badge = stats?.stale ? (
    <Badge variant="secondary">Stale ({stats.source ?? 'unknown'})</Badge>
  ) : stats?.consistency ? (
    <span className="text-xs text-muted-foreground">{stats.consistency}</span>
  ) : null;

  const dailyRows = stats?.daily ?? [];

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
              <DirectoryTable>
                <DirectoryTableHead>
                  <TableRow>
                    <TableHeader>Day</TableHeader>
                    <TableHeader className="text-right">Impressions</TableHeader>
                    <TableHeader className="text-right">Clicks</TableHeader>
                    <TableHeader className="text-right">Conversions</TableHeader>
                  </TableRow>
                </DirectoryTableHead>
                <TableBody className={cn(listRevalidating && directoryTableRevalidatingClass)}>
                  {dailyRows.map((row, index) => (
                    <TableRow key={row.day ?? String(index)}>
                      <TableCell>{displayTimestamp(row.day) || '-'}</TableCell>
                      <TableCell className="text-right">
                        {displayCount(row.impressions) || '-'}
                      </TableCell>
                      <TableCell className="text-right">
                        {displayCount(row.clicks) || '-'}
                      </TableCell>
                      <TableCell className="text-right">
                        {displayCount(row.conversions) || '-'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </DirectoryTable>
            </MetricsSection>
          ) : null}

          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <MetricTile label="Source" value={stats.source?.trim() ? stats.source : '-'} />
            <MetricTile
              label="Consistency"
              value={stats.consistency?.trim() ? stats.consistency : '-'}
            />
          </div>
        </TableHost>
      )}
    </PageLayout>
  );
}
