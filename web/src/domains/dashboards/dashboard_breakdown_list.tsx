import { Link } from 'react-router-dom';

import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import type { DashboardBreakdownRow, DashboardBreakdownTable } from '@/domains/dashboards/dashboard_types';
import { DASHBOARD_BREAKDOWN_UI_CAP } from '@/domains/dashboards/dashboard_types';
import { formatDashboardRoiPct } from '@/lib/display_metrics';
import { displayCount, displayMicro } from '@/lib/display';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';

type DashboardBreakdownListProps = {
  table: DashboardBreakdownTable | undefined;
  exportHref: string;
  stale: boolean;
};

function staleAwareMicro(value: number | undefined, stale: boolean): string {
  if (stale && (value == null || value === 0)) {
    return '-';
  }
  return displayMicro(value);
}

function rowKey(row: DashboardBreakdownRow, index: number): string {
  return row.id ?? `${row.name ?? 'row'}-${index}`;
}

export function DashboardBreakdownList({ table, exportHref, stale }: DashboardBreakdownListProps) {
  const rows = (table?.rows ?? []).slice(0, DASHBOARD_BREAKDOWN_UI_CAP);
  if (rows.length === 0) {
    return null;
  }

  const truncated = table?.truncated === true || (table?.total ?? rows.length) > rows.length;

  return (
    <DashboardPanelSection
      data-testid="dashboard-breakdown"
      footer={
        truncated ? (
          <>
            Showing {rows.length} of {table?.total ?? rows.length}.{' '}
            <Link data-testid="dashboard-export-breakdown" to={exportHref}>
              Export full report
            </Link>
          </>
        ) : undefined
      }
      tableAriaLabel="Top campaigns by performance"
      title="Top campaigns"
    >
      <TableHeader>
        <TableRow>
          <DirectoryTableHead>Campaign</DirectoryTableHead>
          <DirectoryTableHead>Clicks</DirectoryTableHead>
          <DirectoryTableHead>Conversions</DirectoryTableHead>
          <DirectoryTableHead>Revenue</DirectoryTableHead>
          <DirectoryTableHead>ROI</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row, index) => (
          <TableRow key={rowKey(row, index)}>
            <TableCell>{row.name ?? row.id ?? ''}</TableCell>
            <TableCell>{displayCount(row.clicks)}</TableCell>
            <TableCell>{displayCount(row.conversions)}</TableCell>
            <TableCell>{staleAwareMicro(row.revenue_micro, stale)}</TableCell>
            <TableCell>
              {stale && (row.roi_pct == null || row.roi_pct === 0)
                ? '-'
                : formatDashboardRoiPct(row.roi_pct ?? 0)}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DashboardPanelSection>
  );
}
