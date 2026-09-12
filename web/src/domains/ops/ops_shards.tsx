import { useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import type { OpsShardsResponse, ShardHealthStatus } from '@/api/types';
import { EmptyState } from '@/shell/empty_state';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { TableHost } from '@/shell/ui_bands';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageWithLoad } from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';

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

function shardRowId(shard: ShardHealthStatus): string | undefined {
  if (shard.shard_id != null) {
    return String(shard.shard_id);
  }
  return shard.ping_error ?? undefined;
}

function shardRowLabel(shard: ShardHealthStatus): string {
  if (shard.shard_id != null) {
    return `Shard ${shard.shard_id}`;
  }
  return shard.ping_error ?? 'Shard';
}

function buildShardOverviewFields(shard: ShardHealthStatus): DirectoryOverviewField[] {
  return [
    { label: 'Shard', value: shard.shard_id ?? '-' },
    {
      label: 'Ping',
      value: shard.ping_ok ? 'ok' : (shard.ping_error ?? 'fail'),
    },
    { label: 'Latency (ms)', value: shard.ping_latency_ms ?? '-' },
    { label: 'Config version', value: shard.config_version ?? '-' },
    { label: 'Config lag', value: shard.config_version_lag ?? '-' },
    { label: 'Synced', value: shard.config_version_synced ? 'yes' : 'no' },
  ];
}

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
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const shards = snapshot?.shards ?? [];
  const recordById = useMemo(() => directoryRecordMap(shards, shardRowId), [shards]);
  const rows = useMemo(() => directoryOperateRows(shards, shardRowId, shardRowLabel), [shards]);

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
          {catchupStatus ? <p role="status">Catch-up status: {catchupStatus}</p> : null}
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
        <TableHost>
          <DirectorySelectOverviewTable
            buildOverviewFields={buildShardOverviewFields}
            disabled={fetching || catchingUp}
            nameColumnLabel="Shard"
            overviewTitle={(shard) => shardRowLabel(shard)}
            recordById={recordById}
            rows={rows}
            selectedId={selectedId}
            onSelectedIdChange={setSelectedId}
          />
        </TableHost>
      )}
    </OpsPageWithLoad>
  );
}
