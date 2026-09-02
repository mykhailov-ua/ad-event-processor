import { EmptyState } from '@/shell/empty_state';
import type { IncidentSnapshot } from '@/api/types';
import { opsPanelError } from '@/domains/ops/ops_nav';
import {
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import { OpsTable, OpsBlock } from '@/domains/ops/ops_table';

export type OpsIncidentsProps = {
  snapshot: IncidentSnapshot | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function OpsIncidents({ snapshot, fetching, error, hasSnapshot }: OpsIncidentsProps) {
  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError error={error} pageTitle="Incidents" title="Could not load incidents" />
    );
  }

  if (!snapshot) {
    return (
      <OpsPageShell title="Incidents">
        <EmptyState description="Incident snapshot returned no data." title="No incidents" />
      </OpsPageShell>
    );
  }

  const shards = snapshot.shards ?? [];
  const campaigns = snapshot.affected_campaigns ?? [];

  return (
    <OpsPageShell
      badge={
        <>
          {snapshot.emergency_breaker ? (
            <OpsStatusChip status={snapshot.emergency_breaker} />
          ) : null}
          {snapshot.partial ? <OpsStatusChip status="partial" /> : null}
          {snapshot.stale_dashboard ? <OpsStatusChip status="stale" /> : null}
        </>
      }
      title="Incidents"
    >
      {shards.length > 0 ? (
        <OpsBlock title="Shard health">
          <OpsTable
            head={
              <tr>
                <th>Shard</th>
                <th>Ping</th>
                <th className="num">Latency (ms)</th>
                <th className="num">Config lag</th>
                <th>Synced</th>
              </tr>
            }
          >
            {shards.map((shard) => (
              <tr key={shard.shard_id ?? shard.ping_error}>
                <td className="num">{shard.shard_id ?? ''}</td>
                <td>{shard.ping_ok ? 'ok' : (shard.ping_error ?? 'fail')}</td>
                <td className="num">{shard.ping_latency_ms ?? ''}</td>
                <td className="num">{shard.config_version_lag ?? ''}</td>
                <td>{shard.config_version_synced ? 'yes' : 'no'}</td>
              </tr>
            ))}
          </OpsTable>
        </OpsBlock>
      ) : null}

      {campaigns.length > 0 ? (
        <OpsBlock title="Affected campaigns">
          <OpsTable
            head={
              <tr>
                <th>Campaign ID</th>
                <th>Name</th>
              </tr>
            }
          >
            {campaigns.map((row) => (
              <tr key={row.campaign_id ?? row.name}>
                <td className="admin-table-td--id">{row.campaign_id ?? ''}</td>
                <td>{row.name ?? ''}</td>
              </tr>
            ))}
          </OpsTable>
        </OpsBlock>
      ) : null}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
