import { Link } from 'react-router-dom';

import type { FlowsPageWorkspace } from '@/domains/creative/use_flows_page_workspace';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { displayTimestamp } from '@/lib/display';
import { DirectoryPageShell } from '@/shell/directory_page_shell';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { EmptyState } from '@/shell/empty_state';

export type FlowsDirectoryProps = FlowsPageWorkspace;

export function FlowsDirectory(workspace: FlowsDirectoryProps) {
  const { flows, error, fetching, listRevalidating, hasSnapshot } = workspace;
  const rows = flows ?? [];

  return (
    <DirectoryPageShell
      title="Flows"
      description="Campaign stream paths: landers, offers, rotation, and geo/device filters."
      blockingErrorTitle="Could not load flows"
      fetchState={{ fetching, error, hasSnapshot, revalidating: listRevalidating }}
    >
      <CreativeDirectoryStack>
        {hasSnapshot && rows.length === 0 ? (
          <EmptyState description="Create a flow via API or campaign wizard." title="No flows" />
        ) : (
          <DirectoryTable>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Name</DirectoryTableHead>
                <DirectoryTableHead>Flow ID</DirectoryTableHead>
                <DirectoryTableHead>Created</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <Link to={`/flows/${row.id}`}>{row.name}</Link>
                  </TableCell>
                  <TableCell>{row.id}</TableCell>
                  <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        )}
        {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </CreativeDirectoryStack>
    </DirectoryPageShell>
  );
}
