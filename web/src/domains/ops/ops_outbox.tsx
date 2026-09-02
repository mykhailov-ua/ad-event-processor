import { EmptyState } from '@/shell/empty_state';
import type { OutboxEvent } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import {
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import { OpsTable } from '@/domains/ops/ops_table';

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
  fetching,
  error,
  hasSnapshot,
  canGoPrev,
  onPrev,
  onNext,
}: OpsOutboxProps) {
  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return <OpsPageBlockingError error={error} pageTitle="Outbox" title="Could not load outbox" />;
  }

  return (
    <OpsPageShell
      footer={
        <OpsListFooter
          canGoNext={Boolean(nextCursor)}
          canGoPrev={canGoPrev}
          disabled={fetching}
          summary={total != null ? `${total} events total` : 'Outbox event tail'}
          onNext={onNext}
          onPrev={onPrev}
        />
      }
      title="Outbox"
    >
      {items.length === 0 ? (
        <EmptyState description="Outbox tail is empty for this page." title="No outbox events" />
      ) : (
        <OpsTable
          head={
            <tr>
              <th>ID</th>
              <th>Event type</th>
              <th>Status</th>
              <th>Created</th>
            </tr>
          }
        >
          {items.map((row) => (
            <tr key={row.id ?? `${row.event_type}-${row.created_at}`}>
              <td className="admin-table-td--id">{row.id ?? ''}</td>
              <td>{row.event_type ?? ''}</td>
              <td>{row.status ?? ''}</td>
              <td>{displayTimestamp(row.created_at)}</td>
            </tr>
          ))}
        </OpsTable>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
