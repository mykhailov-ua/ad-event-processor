import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import { RowActionsMenu } from '@/components/system/row_actions_menu';
import { Badge } from '@/components/ui/badge';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { DLQInboxEntry } from '@/api/types';
import { OpsNav } from '@/domains/ops/ops_nav';
import { displayTimestamp } from '@/lib/display';

export type OpsDlqInboxProps = {
  items: DLQInboxEntry[];
  nextCursor?: string;
  partial?: boolean;
  limit: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  retryingId?: string;
  onPrev: () => void;
  onNext: () => void;
  canGoPrev: boolean;
  onRetry: (entry: DLQInboxEntry) => void;
};

export function OpsDlqInbox({
  items,
  nextCursor,
  partial,
  limit,
  fetching,
  error,
  hasSnapshot,
  retryingId,
  onPrev,
  onNext,
  canGoPrev,
  onRetry,
}: OpsDlqInboxProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="DLQ inbox">
        <OpsNav />
        <ErrorBlock title="Could not load DLQ inbox" message={error.message} />
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="DLQ inbox"
      badge={partial ? <Badge variant="secondary">partial fan-out</Badge> : undefined}
    >
      <OpsNav />
      <form
        className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
        onSubmit={(event) => event.preventDefault()}
      >
        <div className="col-span-full text-sm text-muted-foreground">
          <span>{items.length} entries on this page</span>
          {nextCursor ? <span className="ml-2">More pages available</span> : null}
        </div>
        <PaginationPrevNext
          canGoPrev={canGoPrev}
          canGoNext={Boolean(nextCursor)}
          disabled={fetching}
          onPrev={onPrev}
          onNext={onNext}
        />
      </form>

      {items.length === 0 ? (
        <EmptyState title="DLQ inbox empty" description="No failed deliveries are queued." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Source</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Campaign</TableHead>
                <TableHead>Event</TableHead>
                <TableHead>Error</TableHead>
                <TableHead>Failed</TableHead>
                <TableHead className="text-right">Retries</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((entry) => {
                const rowKey = entry.id ?? `${entry.source}-${entry.failed_at}`;
                const canRetry = Boolean(entry.id && entry.source);
                return (
                  <TableRow key={rowKey}>
                    <TableCell>{entry.source ?? ''}</TableCell>
                    <TableCell>
                      {entry.status ? <Badge variant="outline">{entry.status}</Badge> : ''}
                    </TableCell>
                    <TableCell className="font-mono text-xs">{entry.campaign_id ?? ''}</TableCell>
                    <TableCell>{entry.event_type ?? ''}</TableCell>
                    <TableCell className="max-w-xs truncate text-muted-foreground">
                      {entry.error ?? ''}
                    </TableCell>
                    <TableCell>{displayTimestamp(entry.failed_at, entry.failed_at_display)}</TableCell>
                    <TableCell className="text-right tabular-nums">{entry.retry_count ?? ''}</TableCell>
                    <TableCell className="text-right">
                      {canRetry ? (
                        <RowActionsMenu
                          ariaLabel="DLQ entry actions"
                          disabled={fetching || retryingId === entry.id}
                        >
                          <DropdownMenuItem
                            disabled={fetching || retryingId === entry.id}
                            onClick={() => onRetry(entry)}
                          >
                            {retryingId === entry.id ? 'Retrying…' : 'Retry'}
                          </DropdownMenuItem>
                        </RowActionsMenu>
                      ) : null}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
