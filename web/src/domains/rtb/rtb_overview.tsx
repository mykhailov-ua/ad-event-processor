import { memo, useMemo } from 'react';
import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { StubBanner } from '@/shell/stub_banner';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { DataFreshness, ReportMapRow } from '@/api/types';
import { RtbNav } from '@/domains/rtb/rtb_nav';
import { deriveColumns, formatMapCell, reportMapRowKey } from '@/lib/report_table';

export type RtbOverviewProps = {
  overviewRows: ReportMapRow[];
  noBidRows: ReportMapRow[];
  freshness?: DataFreshness;
  draftFrom: string;
  draftTo: string;
  fetching: boolean;
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
        <StubBanner
          title="OpenRTB license required"
          message="RTB dashboards require the openrtb license feature. Contact your operator to enable RTB."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load RTB reports" message={error.message} />;
  }

  return (
    <PageChrome
      title="RTB"
      badge={
        freshness?.stale ? (
          <Badge variant="secondary">stale CH lag {freshness.ch_lag_seconds ?? '?'}s</Badge>
        ) : undefined
      }
    >
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

      <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
        <DatetimePicker id="rtb-from" label="From" value={draftFrom} onChange={onDraftFromChange} />
        <DatetimePicker id="rtb-to" label="To" value={draftTo} onChange={onDraftToChange} />
        <FilterApplyButton disabled={fetching} onClick={onApply} type="button">
          Refresh
        </FilterApplyButton>
      </div>

      {overviewRows.length === 0 && noBidRows.length === 0 ? (
        <EmptyState title="No RTB rows" description="Adjust the time range and refresh." />
      ) : (
        <RtbReportTables overviewRows={overviewRows} noBidRows={noBidRows} />
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}

const RtbReportTables = memo(function RtbReportTables({
  overviewRows,
  noBidRows,
}: {
  overviewRows: ReportMapRow[];
  noBidRows: ReportMapRow[];
}) {
  const overviewColumns = useMemo(() => deriveColumns(overviewRows), [overviewRows]);
  const noBidColumns = useMemo(() => deriveColumns(noBidRows), [noBidRows]);

  return (
    <div className="grid gap-4">
      {overviewRows.length > 0 ? (
        <ReportTable caption="Auction overview" columns={overviewColumns} rows={overviewRows} />
      ) : null}
      {noBidRows.length > 0 ? (
        <ReportTable caption="No-bid reasons" columns={noBidColumns} rows={noBidRows} />
      ) : null}
    </div>
  );
});

function ReportTable({
  caption,
  columns,
  rows,
}: {
  caption: string;
  columns: string[];
  rows: ReportMapRow[];
}) {
  return (
    <div className="ui-table-frame">
      <p className="border-b px-4 py-2 text-sm font-medium">{caption}</p>
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((column) => (
              <TableHead key={column}>{column}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, index) => (
            <TableRow key={reportMapRowKey(row, columns, index, caption)}>
              {columns.map((column) => (
                <TableCell key={column}>{formatMapCell(row[column])}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
