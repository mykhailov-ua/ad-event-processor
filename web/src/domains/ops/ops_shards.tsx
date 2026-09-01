import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { OpsShardsResponse } from '@/api/types';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

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
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Shards">
        <OpsNav />
        {opsPanelError(error, 'Could not load shards')}
      </PageChrome>
    );
  }

  const shards = snapshot?.shards ?? [];

  return (
    <PageChrome
      title="Shards"
      badge={
        snapshot?.emergency_breaker ? (
          <Badge variant="destructive">{snapshot.emergency_breaker}</Badge>
        ) : undefined
      }
    >
      <OpsNav />

      <div className="flex flex-wrap gap-2">
        <Button disabled={catchingUp} onClick={onCatchup} type="button" variant="secondary">
          {catchingUp ? 'Starting catch-up...' : 'Shard 0 catch-up'}
        </Button>
      </div>

      {catchupStatus ? (
        <p className="text-sm text-muted-foreground" role="status">
          Catch-up status: {catchupStatus}
        </p>
      ) : null}
      {catchupError ? opsPanelError(catchupError, 'Catch-up failed') : null}

      {shards.length === 0 ? (
        <EmptyState title="No shard rows" description="Shard health matrix is empty." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Shard</TableHead>
                <TableHead>Ping</TableHead>
                <TableHead>Latency (ms)</TableHead>
                <TableHead>Config version</TableHead>
                <TableHead>Lag</TableHead>
                <TableHead>Synced</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {shards.map((shard) => (
                <TableRow key={shard.shard_id ?? shard.ping_error}>
                  <TableCell>{shard.shard_id ?? ''}</TableCell>
                  <TableCell>{shard.ping_ok ? 'ok' : shard.ping_error ?? 'fail'}</TableCell>
                  <TableCell>{shard.ping_latency_ms ?? ''}</TableCell>
                  <TableCell>{shard.config_version ?? ''}</TableCell>
                  <TableCell>{shard.config_version_lag ?? ''}</TableCell>
                  <TableCell>{shard.config_version_synced ? 'yes' : 'no'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
