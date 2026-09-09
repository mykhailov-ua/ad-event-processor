import { EmptyState } from '@/shell/empty_state';
import type { OutboxEvent } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import { OpsPageWithLoad } from '@/domains/ops/ops_page_shell';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

export type OpsOutboxProps = {
  items?: OutboxEvent[];
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
  return (
    <OpsPageWithLoad
      blockingErrorTitle="Could not load outbox"
      fetchState={{ fetching, error, hasSnapshot }}
      title="Outbox"
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
    >
      {(items ?? []).length === 0 ? (
        <EmptyState description="Outbox tail is empty for this page." title="No outbox events" />
      ) : (
        <OpsTable
          horizontalScroll
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>ID</OpsTableHead>
              <OpsTableHead>Event type</OpsTableHead>
              <OpsTableHead>Status</OpsTableHead>
              <OpsTableHead>Created</OpsTableHead>
            </OpsTableHeaderRow>
          }
        >
          {(items ?? []).map((row) => (
            <OpsTableRow key={row.id ?? `${row.event_type}-${row.created_at}`}>
              <OpsTableCell >{row.id ?? ''}</OpsTableCell>
              <OpsTableCell>{row.event_type ?? ''}</OpsTableCell>
              <OpsTableCell>{row.status ?? ''}</OpsTableCell>
              <OpsTableCell>{displayTimestamp(row.created_at)}</OpsTableCell>
            </OpsTableRow>
          ))}
        </OpsTable>
      )}

    </OpsPageWithLoad>
  );
}
