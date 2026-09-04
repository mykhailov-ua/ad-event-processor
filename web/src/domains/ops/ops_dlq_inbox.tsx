import { EmptyState } from '@/shell/empty_state';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import type { DLQInboxEntry } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import {
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

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
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError error={error} pageTitle="DLQ inbox" title="Could not load DLQ inbox" />
    );
  }

  return (
    <OpsPageShell
      badge={partial ? <OpsStatusChip status="partial" /> : undefined}
      footer={
        <OpsListFooter
          canGoNext={Boolean(nextCursor)}
          canGoPrev={canGoPrev}
          disabled={fetching}
          summary={`${items.length} entries on this page${nextCursor ? '  /  more pages available' : ''}`}
          onNext={onNext}
          onPrev={onPrev}
        />
      }
      title="DLQ inbox"
    >
      {items.length === 0 ? (
        <EmptyState description="No failed deliveries are queued." title="DLQ inbox empty" />
      ) : (
        <OpsTable
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>Source</OpsTableHead>
              <OpsTableHead>Status</OpsTableHead>
              <OpsTableHead>Campaign</OpsTableHead>
              <OpsTableHead>Event</OpsTableHead>
              <OpsTableHead>Error</OpsTableHead>
              <OpsTableHead>Failed</OpsTableHead>
              <OpsTableHead numeric>Retries</OpsTableHead>
              <OpsTableHead />
            </OpsTableHeaderRow>
          }
        >
          {items.map((entry) => {
            const rowKey = entry.id ?? `${entry.source}-${entry.failed_at}`;
            const canRetry = Boolean(entry.id && entry.source);
            return (
              <OpsTableRow key={rowKey}>
                <OpsTableCell>{entry.source ?? ''}</OpsTableCell>
                <OpsTableCell>
                  {entry.status ? <OpsStatusChip status={entry.status} /> : ''}
                </OpsTableCell>
                <OpsTableCell className="font-mono text-xs text-muted-foreground">
                  {entry.campaign_id ?? ''}
                </OpsTableCell>
                <OpsTableCell>{entry.event_type ?? ''}</OpsTableCell>
                <OpsTableCell className="max-w-0 truncate text-muted-foreground">
                  {entry.error ?? ''}
                </OpsTableCell>
                <OpsTableCell>{displayTimestamp(entry.failed_at, entry.failed_at_display)}</OpsTableCell>
                <OpsTableCell numeric>{entry.retry_count ?? ''}</OpsTableCell>
                <OpsTableCell className="w-10 text-center">
                  {canRetry ? (
                    <RowActionsMenu
                      ariaLabel="DLQ entry actions"
                      disabled={fetching || retryingId === entry.id}
                    >
                      <DropdownMenuItem
                        disabled={fetching || retryingId === entry.id}
                        onClick={() => onRetry(entry)}
                      >
                        {retryingId === entry.id ? 'Retrying...' : 'Retry'}
                      </DropdownMenuItem>
                    </RowActionsMenu>
                  ) : null}
                </OpsTableCell>
              </OpsTableRow>
            );
          })}
        </OpsTable>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
