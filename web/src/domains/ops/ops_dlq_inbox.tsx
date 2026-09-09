import { EmptyState } from '@/shell/empty_state';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import type { DLQInboxEntry } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import { OpsPageWithLoad } from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

export type OpsDlqInboxProps = {
  items?: DLQInboxEntry[];
  nextCursor?: string;
  partial?: boolean;
  limit: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  retryingId?: string;
  retryError?: Error;
  onPrev: () => void;
  onNext: () => void;
  canGoPrev: boolean;
  onRetry: (entry: DLQInboxEntry) => void;
  embedded?: boolean;
};

export function OpsDlqInbox({
  items,
  nextCursor,
  partial,
  fetching,
  error,
  hasSnapshot,
  retryingId,
  retryError,
  onPrev,
  onNext,
  canGoPrev,
  onRetry,
  embedded = false,
}: OpsDlqInboxProps) {
  const body = (
    <>
      {(items ?? []).length === 0 ? (
        <EmptyState description="No failed deliveries are queued." title="DLQ inbox empty" />
      ) : (
        <OpsTable
          horizontalScroll
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
          {(items ?? []).map((entry) => {
            const rowKey = entry.id ?? `${entry.source}-${entry.failed_at}`;
            const canRetry = Boolean(entry.id && entry.source);
            return (
              <OpsTableRow key={rowKey}>
                <OpsTableCell>{entry.source ?? ''}</OpsTableCell>
                <OpsTableCell>
                  {entry.status ? <OpsStatusChip status={entry.status} /> : ''}
                </OpsTableCell>
                <OpsTableCell >
                  {entry.campaign_id ?? ''}
                </OpsTableCell>
                <OpsTableCell>{entry.event_type ?? ''}</OpsTableCell>
                <OpsTableCell >
                  {entry.error ?? ''}
                </OpsTableCell>
                <OpsTableCell>
                  {displayTimestamp(entry.failed_at, entry.failed_at_display)}
                </OpsTableCell>
                <OpsTableCell numeric>{entry.retry_count ?? ''}</OpsTableCell>
                <OpsTableCell >
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
    </>
  );

  if (embedded) {
    return (
      <>
        {retryError ? opsPanelError(retryError, 'Retry failed') : null}
        {body}
        <OpsListFooter
          canGoNext={Boolean(nextCursor)}
          canGoPrev={canGoPrev}
          disabled={fetching}
          summary={`${(items ?? []).length} entries on this page${nextCursor ? '  /  more pages available' : ''}`}
          onNext={onNext}
          onPrev={onPrev}
        />
      </>
    );
  }

  return (
    <OpsPageWithLoad
      badge={partial ? <OpsStatusChip status="partial" /> : undefined}
      blockingErrorTitle="Could not load DLQ inbox"
      fetchState={{ fetching, error, hasSnapshot }}
      title="DLQ inbox"
      alerts={retryError ? opsPanelError(retryError, 'Retry failed') : null}
      footer={
        <OpsListFooter
          canGoNext={Boolean(nextCursor)}
          canGoPrev={canGoPrev}
          disabled={fetching}
          summary={`${(items ?? []).length} entries on this page${nextCursor ? '  /  more pages available' : ''}`}
          onNext={onNext}
          onPrev={onPrev}
        />
      }
    >
      {body}
    </OpsPageWithLoad>
  );
}
