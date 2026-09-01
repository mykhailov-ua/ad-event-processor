import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { OutboxEvent } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

export type OpsOutboxProps = {
  items: OutboxEvent[];
  nextCursor?: string;
  total?: number;
  limit: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  canGoPrev: boolean;
  onPrev: () => void;
  onNext: () => void;
};

export function OpsOutbox({
  items,
  nextCursor,
  total,
  limit,
  fetching,
  error,
  hasSnapshot,
  canGoPrev,
  onPrev,
  onNext,
}: OpsOutboxProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Outbox">
        <OpsNav />
        {opsPanelError(error, 'Could not load outbox')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Outbox">
      <OpsNav />

      <form
        className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
        onSubmit={(event) => event.preventDefault()}
      >
        <p className="col-span-full text-sm text-muted-foreground">
          {total != null ? `${total} events total` : 'Outbox event tail'}
        </p>
        <PaginationPrevNext
          canGoPrev={canGoPrev}
          canGoNext={Boolean(nextCursor)}
          disabled={fetching}
          onPrev={onPrev}
          onNext={onNext}
        />
      </form>

      {items.length === 0 ? (
        <EmptyState title="No outbox events" description="Outbox tail is empty for this page." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Event type</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id ?? `${row.event_type}-${row.created_at}`}>
                  <TableCell>{row.id ?? ''}</TableCell>
                  <TableCell>{row.event_type ?? ''}</TableCell>
                  <TableCell>{row.status ?? ''}</TableCell>
                  <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
