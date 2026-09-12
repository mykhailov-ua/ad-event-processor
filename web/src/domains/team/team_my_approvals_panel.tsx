import type { TeamBudgetApproval } from '@/api/types';
import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import { DirectoryFetchError } from '@/shell/directory_page_shell';
import { Badge } from '@/components/ui/badge';
import { displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';

type TeamMyApprovalsPanelProps = {
  items: TeamBudgetApproval[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

function approvalStatusVariant(
  status: string | undefined
): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch ((status ?? '').toUpperCase()) {
    case 'PENDING':
      return 'secondary';
    case 'DENIED':
      return 'destructive';
    case 'APPROVED':
      return 'default';
    default:
      return 'outline';
  }
}

export function TeamMyApprovalsPanel({
  items,
  fetching,
  error,
  hasSnapshot,
}: TeamMyApprovalsPanelProps) {
  return (
    <>
      {error ? (
        <DirectoryFetchError
          error={error}
          fetchState={{ error, fetching, hasSnapshot }}
          title="Could not load your budget requests"
        />
      ) : null}
      {!error && hasSnapshot && items.length === 0 && !fetching ? (
        <p className={adminTypography.bodyMuted}>No budget approval requests yet.</p>
      ) : null}
      {!error && hasSnapshot && items.length > 0 ? (
        <DashboardPanelSection
          title="My budget requests"
          tableAriaLabel="My budget approval requests"
        >
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Campaign</DirectoryTableHead>
              <DirectoryTableHead>Requested</DirectoryTableHead>
              <DirectoryTableHead>Previous</DirectoryTableHead>
              <DirectoryTableHead>Status</DirectoryTableHead>
              <DirectoryTableHead>Created</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((row) => (
              <TableRow key={row.id}>
                <TableCell>{row.campaign_id ?? '-'}</TableCell>
                <TableCell>{row.requested_budget_micro ?? '-'}</TableCell>
                <TableCell>{row.previous_budget_micro ?? '-'}</TableCell>
                <TableCell>
                  <Badge variant={approvalStatusVariant(row.status)}>{row.status ?? '-'}</Badge>
                </TableCell>
                <TableCell>{displayTimestamp(row.created_at_display ?? row.created_at)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DashboardPanelSection>
      ) : null}
    </>
  );
}
