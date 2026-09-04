import { EmptyState } from '@/shell/empty_state';
import type { IncidentSnapshot } from '@/api/types';
import { opsPanelError } from '@/domains/ops/ops_nav';
import {
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import { OpsBlock, OpsTable, OpsTableCell, OpsTableHead, OpsTableHeaderRow, OpsTableRow } from '@/domains/ops/ops_table';

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
              <OpsTableHeaderRow>
                <OpsTableHead>Shard</OpsTableHead>
                <OpsTableHead>Ping</OpsTableHead>
                <OpsTableHead numeric>Latency (ms)</OpsTableHead>
                <OpsTableHead numeric>Config lag</OpsTableHead>
                <OpsTableHead>Synced</OpsTableHead>
              </OpsTableHeaderRow>
            }
          >
            {shards.map((shard) => (
              <OpsTableRow key={shard.shard_id ?? shard.ping_error}>
                <OpsTableCell numeric>{shard.shard_id ?? ''}</OpsTableCell>
                <OpsTableCell>{shard.ping_ok ? 'ok' : (shard.ping_error ?? 'fail')}</OpsTableCell>
                <OpsTableCell numeric>{shard.ping_latency_ms ?? ''}</OpsTableCell>
                <OpsTableCell numeric>{shard.config_version_lag ?? ''}</OpsTableCell>
                <OpsTableCell>{shard.config_version_synced ? 'yes' : 'no'}</OpsTableCell>
              </OpsTableRow>
            ))}
          </OpsTable>
        </OpsBlock>
      ) : null}

      {campaigns.length > 0 ? (
        <OpsBlock title="Affected campaigns">
          <OpsTable
            head={
              <OpsTableHeaderRow>
                <OpsTableHead>Campaign ID</OpsTableHead>
                <OpsTableHead>Name</OpsTableHead>
              </OpsTableHeaderRow>
            }
          >
            {campaigns.map((row) => (
              <OpsTableRow key={row.campaign_id ?? row.name}>
                <OpsTableCell className="font-mono text-xs text-muted-foreground">
                  {row.campaign_id ?? ''}
                </OpsTableCell>
                <OpsTableCell>{row.name ?? ''}</OpsTableCell>
              </OpsTableRow>
            ))}
          </OpsTable>
        </OpsBlock>
      ) : null}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
