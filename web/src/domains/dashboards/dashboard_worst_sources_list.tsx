import type { AdopsDashboardPayload } from '@/domains/dashboards/dashboard_types';
import { DASHBOARD_WORST_SOURCES_UI_CAP } from '@/domains/dashboards/dashboard_types';
import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import { formatDashboardCrPct } from '@/lib/display_metrics';
import { displayCount } from '@/lib/display';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';

type DashboardWorstSourcesListProps = {
  rows: AdopsDashboardPayload['worst_sources'];
};

export function DashboardWorstSourcesList({ rows }: DashboardWorstSourcesListProps) {
  const capped = (rows ?? []).slice(0, DASHBOARD_WORST_SOURCES_UI_CAP);
  if (capped.length === 0) {
    return null;
  }

  return (
    <DashboardPanelSection
      data-testid="dashboard-worst-sources"
      tableAriaLabel="Worst IVT sources"
      title="Highest IVT sources"
    >
      <TableHeader>
        <TableRow>
          <DirectoryTableHead>Sub1</DirectoryTableHead>
          <DirectoryTableHead>Campaign</DirectoryTableHead>
          <DirectoryTableHead>Clicks</DirectoryTableHead>
          <DirectoryTableHead>IVT rate</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {capped.map((row, index) => (
          <TableRow key={`${row.campaign_id ?? 'campaign'}-${row.sub1 ?? index}`}>
            <TableCell>{row.sub1 ?? ''}</TableCell>
            <TableCell>{row.campaign_id ?? ''}</TableCell>
            <TableCell>{displayCount(row.clicks)}</TableCell>
            <TableCell>{formatDashboardCrPct(row.ivt_rate)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DashboardPanelSection>
  );
}
