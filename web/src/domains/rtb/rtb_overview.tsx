import { memo } from 'react';

import { FilterApplyButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { DirectoryFilterForm, FilterPanel } from '@/shell/filter_panel';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import type {
  DataFreshness,
  RtbGeoDeviceRow,
  RtbNoBidReasonRow,
  RtbOverviewRow,
} from '@/api/types';
import { RtbNav, RtbLicenseStub, rtbPanelError } from '@/domains/rtb/rtb_nav';
import { formatRatio } from '@/domains/reports/report_metric_display';
import { displayCount, displayMicro } from '@/lib/display';
import { TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';

export type RtbOverviewProps = {
  overviewRows: RtbOverviewRow[];
  noBidRows: RtbNoBidReasonRow[];
  geoDeviceRows: RtbGeoDeviceRow[];
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
  geoDeviceRows,
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

  const hasRows = overviewRows.length > 0 || noBidRows.length > 0 || geoDeviceRows.length > 0;

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
      {!hasRows ? (
        <EmptyState title="No RTB rows" description="Adjust the time range and refresh." />
      ) : (
        <RtbReportTables
          geoDeviceRows={geoDeviceRows}
          listRevalidating={listRevalidating}
          noBidRows={noBidRows}
          overviewRows={overviewRows}
        />
      )}

      {error && hasSnapshot ? rtbPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}

const RtbReportTables = memo(function RtbReportTables({
  overviewRows,
  noBidRows,
  geoDeviceRows,
  listRevalidating,
}: {
  overviewRows: RtbOverviewRow[];
  noBidRows: RtbNoBidReasonRow[];
  geoDeviceRows: RtbGeoDeviceRow[];
  listRevalidating: boolean;
}) {
  const tableClass = cn('w-full', directoryTableRevalidatingClass(listRevalidating));

  return (
    <div className="grid gap-6">
      {overviewRows.length > 0 ? (
        <section className="grid gap-2">
          <h2 className="text-sm font-medium text-foreground">Auction overview</h2>
          <TableHost className="w-full">
            <DirectoryTable className={tableClass} horizontalScroll nested>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Deal</DirectoryTableHead>
                  <DirectoryTableHead align="end">Bids</DirectoryTableHead>
                  <DirectoryTableHead align="end">Wins</DirectoryTableHead>
                  <DirectoryTableHead align="end">Win rate</DirectoryTableHead>
                  <DirectoryTableHead align="end">Spend</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {overviewRows.map((row, index) => (
                  <TableRow key={`${row.deal_id ?? 'deal'}-${index}`}>
                    <TableCell className="font-mono text-xs">{row.deal_id ?? '-'}</TableCell>
                    <TableCell className="text-right">{displayCount(row.bids) || '-'}</TableCell>
                    <TableCell className="text-right">{displayCount(row.wins) || '-'}</TableCell>
                    <TableCell className="text-right">{formatRatio(row.win_rate)}</TableCell>
                    <TableCell className="text-right">
                      {displayMicro(row.spend_micro) || '-'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          </TableHost>
        </section>
      ) : null}
      {noBidRows.length > 0 ? (
        <section className="grid gap-2">
          <h2 className="text-sm font-medium text-foreground">No-bid reasons</h2>
          <TableHost className="w-full">
            <DirectoryTable className={tableClass} horizontalScroll nested>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Reason</DirectoryTableHead>
                  <DirectoryTableHead align="end">Bid count</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {noBidRows.map((row, index) => (
                  <TableRow key={`${row.no_bid_reason ?? 'reason'}-${index}`}>
                    <TableCell>{row.no_bid_reason ?? '-'}</TableCell>
                    <TableCell className="text-right">{displayCount(row.bid_count) || '-'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          </TableHost>
        </section>
      ) : null}
      {geoDeviceRows.length > 0 ? (
        <section className="grid gap-2">
          <h2 className="text-sm font-medium text-foreground">Geo and device</h2>
          <TableHost className="w-full">
            <DirectoryTable className={tableClass} horizontalScroll nested>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Country</DirectoryTableHead>
                  <DirectoryTableHead>Device OS</DirectoryTableHead>
                  <DirectoryTableHead align="end">Bids</DirectoryTableHead>
                  <DirectoryTableHead align="end">Wins</DirectoryTableHead>
                  <DirectoryTableHead align="end">Win rate</DirectoryTableHead>
                  <DirectoryTableHead align="end">Spend</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {geoDeviceRows.map((row, index) => (
                  <TableRow key={`${row.country ?? 'c'}-${row.device_os ?? 'd'}-${index}`}>
                    <TableCell>{row.country ?? '-'}</TableCell>
                    <TableCell>{row.device_os ?? '-'}</TableCell>
                    <TableCell className="text-right">{displayCount(row.bids) || '-'}</TableCell>
                    <TableCell className="text-right">{displayCount(row.wins) || '-'}</TableCell>
                    <TableCell className="text-right">{formatRatio(row.win_rate)}</TableCell>
                    <TableCell className="text-right">
                      {displayMicro(row.spend_micro) || '-'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          </TableHost>
        </section>
      ) : null}
    </div>
  );
});
