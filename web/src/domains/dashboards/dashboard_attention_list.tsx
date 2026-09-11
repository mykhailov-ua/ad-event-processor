import { Link } from 'react-router-dom';

import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import type { DashboardAttentionRow } from '@/domains/dashboards/dashboard_types';
import { DASHBOARD_ATTENTION_UI_CAP } from '@/domains/dashboards/dashboard_types';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { Button } from '@/components/ui/button';

type DashboardAttentionListProps = {
  rows: DashboardAttentionRow[] | undefined;
};

export function DashboardAttentionList({ rows }: DashboardAttentionListProps) {
  const capped = (rows ?? []).slice(0, DASHBOARD_ATTENTION_UI_CAP);
  if (capped.length === 0) {
    return null;
  }

  return (
    <DashboardPanelSection
      data-testid="dashboard-attention"
      tableAriaLabel="Campaigns needing attention"
      title="Needs attention"
    >
      <TableHeader>
        <TableRow>
          <DirectoryTableHead>Campaign</DirectoryTableHead>
          <DirectoryTableHead>Reason</DirectoryTableHead>
          <DirectoryTableHead>Action</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {capped.map((row) => (
          <TableRow key={row.id}>
            <TableCell>{row.name}</TableCell>
            <TableCell>{row.reason}</TableCell>
            <TableCell>
              <Button asChild data-testid={`dashboard-attention-${row.id}`} type="button" variant="outline">
                <Link to={`/campaigns/${row.id}/edit`}>Open</Link>
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DashboardPanelSection>
  );
}
