import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/shell/action_buttons';
import { ReportKpiGrid } from '@/domains/reports/report_kpi_grid';
import { formatRatio } from '@/domains/reports/report_metric_display';
import { displayCount, displayTimestamp } from '@/lib/display';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import type { EdgeParityReportResponse } from '@/api/types';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';

export type EdgeParityDirectoryProps = {
  data: EdgeParityReportResponse | undefined;
  error: Error | undefined;
  fetching: boolean;
  hasSnapshot: boolean;
  shouldFetch: boolean;
  draftFrom: string;
  draftTo: string;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
};

export function EdgeParityDirectory({
  data,
  error,
  fetching,
  hasSnapshot,
  shouldFetch,
  draftFrom,
  draftTo,
  onDraftFromChange,
  onDraftToChange,
  onApplyFilters,
}: EdgeParityDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={4} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load edge parity" message={error.message} />;
  }

  const badge = data?.alert ? (
    <Badge variant="destructive">Divergence alert</Badge>
  ) : data?.freshness?.stale ? (
    <Badge variant="secondary">Stale data</Badge>
  ) : data?.freshness?.as_of ? (
    <span className="text-xs text-muted-foreground">
      As of {displayTimestamp(data.freshness.as_of)}
    </span>
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
              <FilterField htmlFor="edge-parity-from" label="From">
                <DatetimePicker
                  id="edge-parity-from"
                  label="From"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor="edge-parity-to" label="To">
                <DatetimePicker
                  id="edge-parity-to"
                  label="To"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description="Edge ingress vs tracker event parity (max 15 minute window)."
      title="Edge parity"
    >
      {!shouldFetch ? (
        <EmptyState title="Date range required" description="Set from and to, then apply." />
      ) : !data ? (
        <EmptyState title="No data" description="No parity snapshot for this window." />
      ) : (
        <TableHost className="w-full">
          <ReportKpiGrid
            items={[
              { label: 'Window from', value: displayTimestamp(data.from) || '-' },
              { label: 'Window to', value: displayTimestamp(data.to) || '-' },
              { label: 'Edge ingress', value: displayCount(data.edge_ingress) || '-' },
              { label: 'Tracker events', value: displayCount(data.tracker_events) || '-' },
              { label: 'Divergence', value: formatRatio(data.divergence_pct) },
              { label: 'Edge blocked', value: displayCount(data.edge_blocked_total) || '-' },
              { label: 'Blacklist stale', value: displayCount(data.blacklist_stale) || '-' },
              {
                label: 'Shard hint',
                value: data.shard_mismatch_hint?.trim() ? data.shard_mismatch_hint : '-',
              },
            ]}
          />
        </TableHost>
      )}
    </PageLayout>
  );
}
