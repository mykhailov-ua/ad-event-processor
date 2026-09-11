import { Link } from 'react-router-dom';

import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import type {
  AdopsDashboardCampaignRow,
  AdopsDashboardTableMeta,
} from '@/domains/dashboards/dashboard_types';
import { campaignStatusToAdminTone } from '@/lib/admin_kit';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { StatusBadge } from '@/shell/status_badge';
import { Badge } from '@/components/ui/badge';
import { displayMicro } from '@/lib/display';
import { formatDashboardCrPct } from '@/lib/display_metrics';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';

type DashboardAdopsCampaignsListProps = {
  rows: AdopsDashboardCampaignRow[] | undefined;
  tableMeta: AdopsDashboardTableMeta | undefined;
  exportHref: string;
  stale: boolean;
};

function staleAwareMicro(value: number | undefined, stale: boolean): string {
  if (stale && (value == null || value === 0)) {
    return '-';
  }
  return displayMicro(value);
}

function staleAwarePct(value: number | undefined, stale: boolean): string {
  if (stale && (value == null || value === 0)) {
    return '-';
  }
  const formatted = formatDashboardCrPct(value);
  return formatted || '-';
}

function rowKey(row: AdopsDashboardCampaignRow, index: number): string {
  return row.id ?? `campaign-${index}`;
}

export function DashboardAdopsCampaignsList({
  rows,
  tableMeta,
  exportHref,
  stale,
}: DashboardAdopsCampaignsListProps) {
  const visible = rows ?? [];
  if (visible.length === 0) {
    return null;
  }

  const truncated =
    tableMeta?.truncated === true ||
    (tableMeta?.total != null && tableMeta.total > visible.length);

  return (
    <DashboardPanelSection
      data-testid="dashboard-adops-campaigns"
      footer={
        truncated ? (
          <>
            Showing {visible.length} of {tableMeta?.total ?? visible.length}.{' '}
            <Link data-testid="dashboard-export-campaign-overview" to={exportHref}>
              Export campaign overview
            </Link>
          </>
        ) : undefined
      }
      tableAriaLabel="Campaign pacing and budget utilization"
      title="Campaign pacing"
    >
      <TableHeader>
        <TableRow>
          <DirectoryTableHead>Campaign</DirectoryTableHead>
          <DirectoryTableHead>Status</DirectoryTableHead>
          <DirectoryTableHead>Spend</DirectoryTableHead>
          <DirectoryTableHead>Budget</DirectoryTableHead>
          <DirectoryTableHead>Utilization</DirectoryTableHead>
          <DirectoryTableHead>Pacing drift</DirectoryTableHead>
          <DirectoryTableHead>Overspend risk</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {visible.map((row, index) => (
          <TableRow key={rowKey(row, index)}>
            <TableCell>
              {row.id ? (
                <Link className="whitespace-nowrap text-primary" to={`/campaigns/${row.id}/edit`}>
                  {row.name ?? row.id}
                </Link>
              ) : (
                row.name ?? ''
              )}
            </TableCell>
            <TableCell>
              {row.status ? (
                <StatusBadge
                  label={formatCampaignStatusLabel(row.status)}
                  tone={campaignStatusToAdminTone(row.status)}
                />
              ) : (
                ''
              )}
            </TableCell>
            <TableCell>{staleAwareMicro(row.spend_micro, stale)}</TableCell>
            <TableCell>{staleAwareMicro(row.budget_micro, stale)}</TableCell>
            <TableCell>{staleAwarePct(row.utilization_pct, stale)}</TableCell>
            <TableCell>{staleAwarePct(row.pacing_drift_pct, stale)}</TableCell>
            <TableCell>
              {row.overspend_risk ? <Badge variant="destructive">Risk</Badge> : '-'}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DashboardPanelSection>
  );
}
