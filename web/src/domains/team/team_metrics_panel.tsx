import type { TeamMetricsResponse } from '@/api/types';
import { DashboardKpiBlocks } from '@/domains/dashboards/dashboard_kpi_blocks';
import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import { DirectoryFetchError } from '@/shell/directory_page_shell';
import { BentoSection } from '@/shell/bento_card';
import { formatDashboardRoiPct } from '@/lib/display_metrics';
import { displayCount, displayMicro } from '@/lib/display';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { adminTypography } from '@/lib/admin_kit';

export const TEAM_LEADERBOARD_UI_CAP = 10;

type TeamMetricsPanelProps = {
  metrics: TeamMetricsResponse | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  rangeLabel: string;
};

export function TeamMetricsPanel({
  metrics,
  fetching,
  error,
  hasSnapshot,
  rangeLabel,
}: TeamMetricsPanelProps) {
  const stale = metrics?.aggregate?.freshness?.stale ?? false;
  const leaderboard = (metrics?.by_owner ?? []).slice(0, TEAM_LEADERBOARD_UI_CAP);

  return (
    <BentoSection title={`Team KPIs (${rangeLabel})`}>
      {error ? (
        <DirectoryFetchError
          error={error}
          fetchState={{ error, fetching, hasSnapshot }}
          title="Could not load team metrics"
        />
      ) : null}
      {!error && hasSnapshot ? (
        <>
          <DashboardKpiBlocks kpis={metrics?.aggregate} stale={stale} />
          {leaderboard.length > 0 ? (
            <DashboardPanelSection tableAriaLabel="Team ROI leaderboard">
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Owner</DirectoryTableHead>
                  <DirectoryTableHead>Spend</DirectoryTableHead>
                  <DirectoryTableHead>Revenue</DirectoryTableHead>
                  <DirectoryTableHead>ROI</DirectoryTableHead>
                  <DirectoryTableHead>Conversions</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {leaderboard.map((row) => (
                  <TableRow key={row.user_id ?? row.email}>
                    <TableCell>{row.email ?? row.user_id ?? '-'}</TableCell>
                    <TableCell>{displayMicro(row.kpis?.spend_micro)}</TableCell>
                    <TableCell>{stale ? '-' : displayMicro(row.kpis?.revenue_micro)}</TableCell>
                    <TableCell>
                      {stale || row.kpis?.roi_pct == null
                        ? '-'
                        : formatDashboardRoiPct(row.kpis.roi_pct)}
                    </TableCell>
                    <TableCell>{displayCount(row.kpis?.conversions)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DashboardPanelSection>
          ) : fetching ? null : (
            <p className={adminTypography.bodyMuted}>No owner metrics for this range.</p>
          )}
        </>
      ) : null}
    </BentoSection>
  );
}
