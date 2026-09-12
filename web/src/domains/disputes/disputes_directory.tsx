import { Link } from 'react-router-dom';

import type { DisputeRow } from '@/api/types';
import type { DisputesPageWorkspace } from '@/domains/disputes/use_disputes_page_workspace';
import { listPageRange } from '@/lib/list_page_stats';
import { displayMicro, displayTimestamp } from '@/lib/display';
import { DirectoryPageShell } from '@/shell/directory_page_shell';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { EmptyState } from '@/shell/empty_state';

export type DisputesDirectoryProps = DisputesPageWorkspace;

function disputeRowKey(row: DisputeRow, index: number): string {
  return row.intent_id ?? row.provider_dispute_id ?? `dispute-${index}`;
}

export function DisputesDirectory(workspace: DisputesDirectoryProps) {
  const {
    items,
    total,
    limit,
    offset,
    fetching,
    listRevalidating,
    error,
    hasSnapshot,
    onPageChange,
    onLimitChange,
  } = workspace;

  const rows = items ?? [];
  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const pageRange = listPageRange(total, limit, offset, rows.length);
  const rangeLabel =
    pageRange.rangeStart > 0
      ? `${pageRange.rangeStart} - ${pageRange.rangeEnd} of ${total}`
      : '0 of 0';

  return (
    <DirectoryPageShell
      title="Disputes"
      description="Payment disputes across customers. Secondary surface; deep-link only."
      blockingErrorTitle="Could not load disputes"
      fetchState={{ fetching, error, hasSnapshot, revalidating: listRevalidating }}
      footer={
        <DirectoryPaginationFooter
          canGoNext={canGoNext}
          canGoPrev={canGoPrev}
          disabled={fetching}
          limit={limit}
          pageSizeId="disputes-page-size"
          rangeLabel={rangeLabel}
          onLimitChange={onLimitChange}
          onNext={() => onPageChange(offset + limit)}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
        />
      }
    >
      {hasSnapshot && rows.length === 0 ? (
        <EmptyState description="No disputes returned for this page." title="No disputes" />
      ) : (
        <DirectoryTable>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Intent</DirectoryTableHead>
              <DirectoryTableHead>Customer</DirectoryTableHead>
              <DirectoryTableHead>Amount</DirectoryTableHead>
              <DirectoryTableHead>Provider ID</DirectoryTableHead>
              <DirectoryTableHead>Updated</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row, index) => (
              <TableRow key={disputeRowKey(row, index)}>
                <TableCell>{row.intent_id ?? ''}</TableCell>
                <TableCell>
                  {row.customer_id ? (
                    <Link to={`/customers/${row.customer_id}`}>{row.customer_id}</Link>
                  ) : (
                    ''
                  )}
                </TableCell>
                <TableCell>{displayMicro(row.amount_micro, undefined)}</TableCell>
                <TableCell>{row.provider_dispute_id ?? ''}</TableCell>
                <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      )}
    </DirectoryPageShell>
  );
}
