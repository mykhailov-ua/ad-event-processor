import { EmptyState } from '@/shell/empty_state';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import type { DLQEntry } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

export type OpsShardDlqProps = {
  items?: DLQEntry[];
  nextCursor?: string;
  fetching: boolean;
  retryingId?: string;
  retryError?: Error;
  onPrev: () => void;
  onNext: () => void;
  canGoPrev: boolean;
  onRetry: (entry: DLQEntry) => void;
};

export function OpsShardDlq({
  items,
  nextCursor,
  fetching,
  retryingId,
  retryError,
  onPrev,
  onNext,
  canGoPrev,
  onRetry,
}: OpsShardDlqProps) {
  return (
    <>
      {retryError ? opsPanelError(retryError, 'Retry failed') : null}
      {(items ?? []).length === 0 ? (
        <EmptyState description="No shard fan-out DLQ entries on this page." title="Shard DLQ empty" />
      ) : (
        <OpsTable
          horizontalScroll
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>Shard</OpsTableHead>
              <OpsTableHead>Stream</OpsTableHead>
              <OpsTableHead>Entry</OpsTableHead>
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
            const rowKey = entry.id ?? `${entry.shard_id}-${entry.entry_id}`;
            const canRetry = Boolean(entry.id);
            return (
              <OpsTableRow key={rowKey}>
                <OpsTableCell numeric>{entry.shard_id ?? ''}</OpsTableCell>
                <OpsTableCell>{entry.stream_id ?? ''}</OpsTableCell>
                <OpsTableCell>{entry.entry_id ?? ''}</OpsTableCell>
                <OpsTableCell>{entry.campaign_id ?? ''}</OpsTableCell>
                <OpsTableCell>{entry.event_type ?? ''}</OpsTableCell>
                <OpsTableCell>{entry.error ?? ''}</OpsTableCell>
                <OpsTableCell>
                  {displayTimestamp(entry.failed_at, undefined)}
                </OpsTableCell>
                <OpsTableCell numeric>{entry.retry_count ?? ''}</OpsTableCell>
                <OpsTableCell>
                  {canRetry ? (
                    <RowActionsMenu
                      ariaLabel="Shard DLQ entry actions"
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
