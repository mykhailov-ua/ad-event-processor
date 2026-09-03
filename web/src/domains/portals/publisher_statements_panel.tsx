import { Link } from 'react-router-dom';

import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { PublisherStatement } from '@/api/types';
import { PortalsNav, portalsPanelError } from '@/domains/portals/portals_nav';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type PublisherStatementsPanelProps = {
  statements: PublisherStatement[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function PublisherStatementsPanel({
  statements,
  fetching,
  error,
  hasSnapshot,
}: PublisherStatementsPanelProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Publisher statements">
        <PortalsNav />
        {portalsPanelError(error, 'Could not load publisher statements')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Publisher statements">
      <PortalsNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/publisher/dashboard">
        Dashboard
      </Link>

      {statements.length === 0 ? (
        <EmptyState title="No statements" description="Publisher revenue statements are empty." />
      ) : (
        <DirectoryTable>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>ID</DirectoryTableHead>
                <DirectoryTableHead>Campaign</DirectoryTableHead>
                <DirectoryTableHead>Amount (micro)</DirectoryTableHead>
                <DirectoryTableHead>Created</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {statements.map((row) => (
                <TableRow key={String(row.id ?? row.idempotency_hash ?? row.created_at)}>
                  <TableCell>{row.id ?? ''}</TableCell>
                  <TableCell className="font-mono text-xs">{row.campaign_id ?? ''}</TableCell>
                  <TableCell>{displayMicro(row.amount_micro)}</TableCell>
                  <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
      )}

      {error && hasSnapshot ? portalsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
