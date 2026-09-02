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
import { OpsTable } from '@/domains/ops/ops_table';

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
            <tr>
              <th>Source</th>
              <th>Status</th>
              <th>Campaign</th>
              <th>Event</th>
              <th>Error</th>
              <th>Failed</th>
              <th className="num">Retries</th>
              <th />
            </tr>
          }
        >
          {items.map((entry) => {
            const rowKey = entry.id ?? `${entry.source}-${entry.failed_at}`;
            const canRetry = Boolean(entry.id && entry.source);
            return (
              <tr key={rowKey}>
                <td>{entry.source ?? ''}</td>
                <td>
                  {entry.status ? <OpsStatusChip status={entry.status} /> : ''}
                </td>
                <td className="admin-table-td--id">{entry.campaign_id ?? ''}</td>
                <td>{entry.event_type ?? ''}</td>
                <td className="admin-muted admin-table-td--truncate">{entry.error ?? ''}</td>
                <td>{displayTimestamp(entry.failed_at, entry.failed_at_display)}</td>
                <td className="num">{entry.retry_count ?? ''}</td>
                <td className="admin-table-td--actions">
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
                </td>
              </tr>
            );
          })}
        </OpsTable>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
