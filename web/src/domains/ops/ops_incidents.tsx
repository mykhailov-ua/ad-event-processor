import { useMemo, useState } from 'react';

import type { IncidentSnapshot, ShardHealthStatus } from '@/api/types';

type AffectedCampaign = NonNullable<IncidentSnapshot['affected_campaigns']>[number];
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/shell/empty_state';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { TableHost } from '@/shell/ui_bands';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsBlock } from '@/domains/ops/ops_table';
import { OpsPageBlockingError, OpsPageLoading, OpsPageShell } from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';

export type OpsIncidentsProps = {
  snapshot: IncidentSnapshot | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

function incidentShardRowId(shard: ShardHealthStatus): string | undefined {
  if (shard.shard_id != null) {
    return String(shard.shard_id);
  }
  return shard.ping_error ?? undefined;
}

function incidentShardRowLabel(shard: ShardHealthStatus): string {
  if (shard.shard_id != null) {
    return `Shard ${shard.shard_id}`;
  }
  return shard.ping_error ?? 'Shard';
}

function buildIncidentShardOverviewFields(shard: ShardHealthStatus): DirectoryOverviewField[] {
  return [
    { label: 'Shard', value: shard.shard_id ?? '-' },
    {
      label: 'Ping',
      value: shard.ping_ok ? 'ok' : (shard.ping_error ?? 'fail'),
    },
    { label: 'Latency (ms)', value: shard.ping_latency_ms ?? '-' },
    { label: 'Config lag', value: shard.config_version_lag ?? '-' },
    { label: 'Synced', value: shard.config_version_synced ? 'yes' : 'no' },
  ];
}

function incidentCampaignRowId(row: AffectedCampaign): string | undefined {
  return row.campaign_id ?? row.name ?? undefined;
}

function incidentCampaignRowLabel(row: AffectedCampaign): string {
  return row.name ?? row.campaign_id ?? 'Campaign';
}

function buildIncidentCampaignOverviewFields(row: AffectedCampaign): DirectoryOverviewField[] {
  return [
    {
      label: 'Campaign ID',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {row.campaign_id ?? '-'}
        </span>
      ),
    },
    { label: 'Name', value: row.name ?? '-' },
  ];
}

function IncidentShardTable({ shards }: { shards: ShardHealthStatus[] }) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const recordById = useMemo(() => directoryRecordMap(shards, incidentShardRowId), [shards]);
  const rows = useMemo(
    () => directoryOperateRows(shards, incidentShardRowId, incidentShardRowLabel),
    [shards]
  );

  return (
    <TableHost>
      <DirectorySelectOverviewTable
        buildOverviewFields={buildIncidentShardOverviewFields}
        nameColumnLabel="Shard"
        overviewTitle={(shard) => incidentShardRowLabel(shard)}
        recordById={recordById}
        rows={rows}
        selectedId={selectedId}
        onSelectedIdChange={setSelectedId}
      />
    </TableHost>
  );
}

function IncidentCampaignTable({ campaigns }: { campaigns: AffectedCampaign[] }) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const recordById = useMemo(
    () => directoryRecordMap(campaigns, incidentCampaignRowId),
    [campaigns]
  );
  const rows = useMemo(
    () => directoryOperateRows(campaigns, incidentCampaignRowId, incidentCampaignRowLabel),
    [campaigns]
  );

  return (
    <TableHost>
      <DirectorySelectOverviewTable
        buildOverviewFields={buildIncidentCampaignOverviewFields}
        nameColumnLabel="Campaign"
        overviewTitle={(row) => incidentCampaignRowLabel(row)}
        recordById={recordById}
        rows={rows}
        selectedId={selectedId}
        onSelectedIdChange={setSelectedId}
      />
    </TableHost>
  );
}

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
          <IncidentShardTable shards={shards} />
        </OpsBlock>
      ) : null}

      {campaigns.length > 0 ? (
        <OpsBlock title="Affected campaigns">
          <IncidentCampaignTable campaigns={campaigns} />
        </OpsBlock>
      ) : null}

      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
