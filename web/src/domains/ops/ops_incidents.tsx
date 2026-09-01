import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { IncidentSnapshot } from '@/api/types';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

export type OpsIncidentsProps = {
  snapshot: IncidentSnapshot | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function OpsIncidents({ snapshot, fetching, error, hasSnapshot }: OpsIncidentsProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Incidents">
        <OpsNav />
        {opsPanelError(error, 'Could not load incidents')}
      </PageChrome>
    );
  }

  if (!snapshot) {
    return (
      <PageChrome title="Incidents">
        <OpsNav />
        <EmptyState title="No incidents" description="Incident snapshot returned no data." />
      </PageChrome>
    );
  }

  const shards = snapshot.shards ?? [];
  const campaigns = snapshot.affected_campaigns ?? [];

  return (
    <PageChrome
      title="Incidents"
      badge={
        snapshot.emergency_breaker ? (
          <Badge variant="destructive">{snapshot.emergency_breaker}</Badge>
        ) : undefined
      }
    >
      <OpsNav />

      {snapshot.partial ? <Badge variant="outline">Partial snapshot</Badge> : null}
      {snapshot.stale_dashboard ? <Badge variant="secondary">Stale dashboard</Badge> : null}

      {shards.length > 0 ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Shard health</h2>
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Shard</TableHead>
                  <TableHead>Ping</TableHead>
                  <TableHead>Latency (ms)</TableHead>
                  <TableHead>Config lag</TableHead>
                  <TableHead>Synced</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {shards.map((shard) => (
                  <TableRow key={shard.shard_id ?? shard.ping_error}>
                    <TableCell>{shard.shard_id ?? ''}</TableCell>
                    <TableCell>{shard.ping_ok ? 'ok' : shard.ping_error ?? 'fail'}</TableCell>
                    <TableCell>{shard.ping_latency_ms ?? ''}</TableCell>
                    <TableCell>{shard.config_version_lag ?? ''}</TableCell>
                    <TableCell>{shard.config_version_synced ? 'yes' : 'no'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </section>
      ) : null}

      {campaigns.length > 0 ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Affected campaigns</h2>
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Campaign ID</TableHead>
                  <TableHead>Name</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {campaigns.map((row) => (
                  <TableRow key={row.campaign_id ?? row.name}>
                    <TableCell className="font-mono text-xs">{row.campaign_id ?? ''}</TableCell>
                    <TableCell>{row.name ?? ''}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </section>
      ) : null}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
