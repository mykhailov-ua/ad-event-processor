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
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

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
            <OpsTableHeaderRow>
              <OpsTableHead>ID</OpsTableHead>
              <OpsTableHead>Event type</OpsTableHead>
              <OpsTableHead>Status</OpsTableHead>
              <OpsTableHead>Created</OpsTableHead>
            </OpsTableHeaderRow>
          }
        >
          {items.map((row) => (
            <OpsTableRow key={row.id ?? `${row.event_type}-${row.created_at}`}>
              <OpsTableCell className="font-mono text-xs text-zinc-500 dark:text-zinc-400">
                {row.id ?? ''}
              </OpsTableCell>
              <OpsTableCell>{row.event_type ?? ''}</OpsTableCell>
              <OpsTableCell>{row.status ?? ''}</OpsTableCell>
              <OpsTableCell>{displayTimestamp(row.created_at)}</OpsTableCell>
            </OpsTableRow>
          ))}
        </OpsTable>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
