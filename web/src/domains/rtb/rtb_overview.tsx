import { memo, useMemo } from 'react';
import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { DirectoryFilterForm, FilterPanel } from '@/shell/filter_panel';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { ReportMapTable } from '@/shell/report_map_table';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import type { DataFreshness, ReportMapRow } from '@/api/types';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';
import { deriveColumns } from '@/lib/report_table';

export type RtbOverviewProps = {
  overviewRows: ReportMapRow[];
  noBidRows: ReportMapRow[];
  freshness?: DataFreshness;
  draftFrom: string;
  draftTo: string;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onApply: () => void;
};

export function RtbOverview({
  overviewRows,
  noBidRows,
  freshness,
  draftFrom,
  draftTo,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  licenseGated,
  onDraftFromChange,
  onDraftToChange,
  onApply,
}: RtbOverviewProps) {
  if (licenseGated) {
    return (
      <PageChrome title="RTB">
        <RtbNav />
        <RtbLicenseStub />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="RTB">
        <RtbNav />
        {rtbPanelError(error, 'Could not load RTB reports')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="RTB"
      badge={
        freshness?.stale ? (
          <Badge variant="secondary">stale CH lag {freshness.ch_lag_seconds ?? '?'}s</Badge>
        ) : undefined
      }
      controlPanel={
        <div className="grid gap-3">
          <RtbNav />
          <div className="flex flex-wrap gap-3 text-sm text-muted-foreground">
            <Link className="hover:underline" to="/reports/rtb-overview">
              Full report runner
            </Link>
            <Link className="hover:underline" to="/reports/rtb-no-bid-reasons">
              No-bid reasons
            </Link>
            <Link className="hover:underline" to="/reports/rtb-geo-device">
              Geo / device
            </Link>
          </div>
          <FilterPanel>
            <DirectoryFilterForm
              layout="auto-fill"
              onSubmit={(event) => {
                event.preventDefault();
                onApply();
              }}
            >
              <DatetimePicker
                id="rtb-from"
                label="From"
                value={draftFrom}
                onChange={onDraftFromChange}
              />
              <DatetimePicker id="rtb-to" label="To" value={draftTo} onChange={onDraftToChange} />
              <FilterApplyButton disabled={fetching} type="submit">
                Refresh
              </FilterApplyButton>
            </DirectoryFilterForm>
          </FilterPanel>
        </div>
      }
    >
      {overviewRows.length === 0 && noBidRows.length === 0 ? (
        <EmptyState title="No RTB rows" description="Adjust the time range and refresh." />
      ) : (
        <RtbReportTables
          listRevalidating={listRevalidating}
          overviewRows={overviewRows}
          noBidRows={noBidRows}
        />
      )}

      {error && hasSnapshot ? rtbPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}

const RtbReportTables = memo(function RtbReportTables({
  overviewRows,
  noBidRows,
  listRevalidating,
}: {
  overviewRows: ReportMapRow[];
  noBidRows: ReportMapRow[];
  listRevalidating: boolean;
}) {
  const overviewColumns = useMemo(() => deriveColumns(overviewRows), [overviewRows]);
  const noBidColumns = useMemo(() => deriveColumns(noBidRows), [noBidRows]);

  return (
    <div className="grid gap-4">
      {overviewRows.length > 0 ? (
        <ReportMapTable
          caption="Auction overview"
          columns={overviewColumns}
          revalidating={listRevalidating}
          rows={overviewRows}
        />
      ) : null}
      {noBidRows.length > 0 ? (
        <ReportMapTable
          caption="No-bid reasons"
          columns={noBidColumns}
          revalidating={listRevalidating}
          rows={noBidRows}
        />
      ) : null}
    </div>
  );
});
