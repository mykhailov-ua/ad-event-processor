import { Button } from '@/components/ui/button';
import { EmptyState } from '@/shell/empty_state';
import type { OpsShardsResponse } from '@/api/types';
import { opsPanelError } from '@/domains/ops/ops_nav';
import {
  OpsActionGroup,
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import { OpsTable } from '@/domains/ops/ops_table';

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
  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return <OpsPageBlockingError error={error} pageTitle="Shards" title="Could not load shards" />;
  }

  const shards = snapshot?.shards ?? [];

  return (
    <OpsPageShell
      badge={
        snapshot?.emergency_breaker ? (
          <OpsStatusChip status={snapshot.emergency_breaker} />
        ) : undefined
      }
      title="Shards"
      actions={
        <OpsActionGroup label="Shard maintenance">
          <Button disabled={catchingUp} loading={catchingUp} type="button" onClick={onCatchup}>
            Shard 0 catch-up
          </Button>
        </OpsActionGroup>
      }
    >
      {catchupStatus ? (
        <p className="admin-muted" role="status">
          Catch-up status: {catchupStatus}
        </p>
      ) : null}
      {catchupError ? opsPanelError(catchupError, 'Catch-up failed') : null}

      {shards.length === 0 ? (
        <EmptyState description="Shard health matrix is empty." title="No shard rows" />
      ) : (
        <OpsTable
          head={
            <tr>
              <th>Shard</th>
              <th>Ping</th>
              <th className="num">Latency (ms)</th>
              <th className="num">Config version</th>
              <th className="num">Lag</th>
              <th>Synced</th>
            </tr>
          }
        >
          {shards.map((shard) => (
            <tr key={shard.shard_id ?? shard.ping_error}>
              <td className="num">{shard.shard_id ?? ''}</td>
              <td>{shard.ping_ok ? 'ok' : (shard.ping_error ?? 'fail')}</td>
              <td className="num">{shard.ping_latency_ms ?? ''}</td>
              <td className="num">{shard.config_version ?? ''}</td>
              <td className="num">{shard.config_version_lag ?? ''}</td>
              <td>{shard.config_version_synced ? 'yes' : 'no'}</td>
            </tr>
          ))}
        </OpsTable>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
