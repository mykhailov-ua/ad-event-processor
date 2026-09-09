import { Button } from '@/components/ui/button';
import { EmptyState } from '@/shell/empty_state';
import type { OpsShardsResponse } from '@/api/types';
import { opsPanelError } from '@/domains/ops/ops_nav';
import {
  OpsActionGroup,
  OpsPageWithLoad,
} from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

export type OpsShardsProps = {
  snapshot: OpsShardsResponse | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  catchingUp: boolean;
  catchupError: Error | undefined;
  catchupStatus: string | undefined;
  onCatchup: () => void;
};

export function OpsShards({
  snapshot,
  fetching,
  error,
  hasSnapshot,
  catchingUp,
  catchupError,
  catchupStatus,
  onCatchup,
}: OpsShardsProps) {
  const shards = snapshot?.shards ?? [];

  return (
    <OpsPageWithLoad
      badge={
        snapshot?.emergency_breaker ? (
          <OpsStatusChip status={snapshot.emergency_breaker} />
        ) : undefined
      }
      blockingErrorTitle="Could not load shards"
      fetchState={{ fetching, error, hasSnapshot }}
      title="Shards"
      alerts={
        <>
          {catchupStatus ? (
            <p  role="status">
              Catch-up status: {catchupStatus}
            </p>
          ) : null}
          {catchupError ? opsPanelError(catchupError, 'Catch-up failed') : null}
        </>
      }
      actions={
        <OpsActionGroup label="Shard maintenance">
          <Button disabled={catchingUp} loading={catchingUp} type="button" onClick={onCatchup}>
            Shard 0 catch-up
          </Button>
        </OpsActionGroup>
      }
    >
      {shards.length === 0 ? (
        <EmptyState description="Shard health matrix is empty." title="No shard rows" />
      ) : (
        <OpsTable
          horizontalScroll
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>Shard</OpsTableHead>
              <OpsTableHead>Ping</OpsTableHead>
              <OpsTableHead numeric>Latency (ms)</OpsTableHead>
              <OpsTableHead numeric>Config version</OpsTableHead>
              <OpsTableHead numeric>Lag</OpsTableHead>
              <OpsTableHead>Synced</OpsTableHead>
            </OpsTableHeaderRow>
          }
        >
          {shards.map((shard) => (
            <OpsTableRow key={shard.shard_id ?? shard.ping_error}>
              <OpsTableCell numeric>{shard.shard_id ?? ''}</OpsTableCell>
              <OpsTableCell>{shard.ping_ok ? 'ok' : (shard.ping_error ?? 'fail')}</OpsTableCell>
              <OpsTableCell numeric>{shard.ping_latency_ms ?? ''}</OpsTableCell>
              <OpsTableCell numeric>{shard.config_version ?? ''}</OpsTableCell>
              <OpsTableCell numeric>{shard.config_version_lag ?? ''}</OpsTableCell>
              <OpsTableCell>{shard.config_version_synced ? 'yes' : 'no'}</OpsTableCell>
            </OpsTableRow>
          ))}
        </OpsTable>
      )}

    </OpsPageWithLoad>
  );
}
