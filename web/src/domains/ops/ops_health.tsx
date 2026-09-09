import { useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/shell/empty_state';
import type { DomainHealth, DoctorSummary, StackHealthSnapshot } from '@/api/types';
import { OpsKvRow, OpsStatGrid, OpsStatPanel } from '@/domains/ops/ops_stat_panel';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageWithLoad } from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import { OpsBlock } from '@/domains/ops/ops_table';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { TableHost } from '@/shell/ui_bands';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { displayTimestamp } from '@/lib/display';

export type OpsHealthProps = {
  stackHealth: StackHealthSnapshot | undefined;
  doctor: DoctorSummary | undefined;
  doctorFetching: boolean;
  doctorError: Error | undefined;
  hasDoctorSnapshot: boolean;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  refreshing: boolean;
  onRefresh: () => void;
  draftHostname: string;
  onDraftHostnameChange: (value: string) => void;
  probeResult: DomainHealth | undefined;
  probing: boolean;
  probeError: Error | undefined;
  onRunProbe: () => void;
};

type DoctorCheck = NonNullable<DoctorSummary['checks']>[number];

function formatSeconds(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(1)}s`;
}

function doctorCheckRowId(check: DoctorCheck): string | undefined {
  return check.id ?? check.message ?? undefined;
}

function doctorCheckRowLabel(check: DoctorCheck): string {
  return check.id ?? check.message ?? 'Doctor check';
}

function buildDoctorCheckOverviewFields(check: DoctorCheck): DirectoryOverviewField[] {
  return [
    { label: 'Check', value: check.id ?? '-' },
    {
      label: 'Status',
      value: check.status ? <OpsStatusChip status={check.status} /> : '-',
    },
    { label: 'Message', value: check.message ?? '-' },
    { label: 'Hint', value: check.hint ?? '-' },
    {
      label: 'Latency',
      value: check.latency_ms != null ? `${check.latency_ms} ms` : '-',
    },
  ];
}

function buildDomainProbeOverviewFields(probe: DomainHealth): DirectoryOverviewField[] {
  return [
    { label: 'Hostname', value: probe.hostname ?? '-' },
    {
      label: 'Health',
      value: probe.health_status ? <OpsStatusChip status={probe.health_status} /> : '-',
    },
    {
      label: 'SSL',
      value: probe.ssl_status ? <OpsStatusChip status={probe.ssl_status} /> : '-',
    },
    { label: 'Role', value: probe.role ?? '-' },
    { label: 'HTTP status', value: probe.http_status ?? '-' },
    {
      label: 'Probe latency',
      value: probe.probe_latency_ms != null ? `${probe.probe_latency_ms} ms` : '-',
    },
    { label: 'Detail', value: probe.probe_detail ?? '-' },
    {
      label: 'Last probe',
      value: probe.last_probe_at ? displayTimestamp(probe.last_probe_at) : '-',
    },
  ];
}

function OpsHealthDoctorChecksTable({ checks }: { checks: DoctorCheck[] }) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const recordById = useMemo(() => directoryRecordMap(checks, doctorCheckRowId), [checks]);
  const rows = useMemo(
    () => directoryOperateRows(checks, doctorCheckRowId, doctorCheckRowLabel),
    [checks]
  );

  return (
    <TableHost>
      <DirectorySelectOverviewTable
        buildOverviewFields={buildDoctorCheckOverviewFields}
        nameColumnLabel="Doctor check"
        overviewTitle={(check) => doctorCheckRowLabel(check)}
        recordById={recordById}
        rows={rows}
        selectedId={selectedId}
        onSelectedIdChange={setSelectedId}
      />
    </TableHost>
  );
}

function DomainProbeOverview({ probe }: { probe: DomainHealth }) {
  const fields = buildDomainProbeOverviewFields(probe);

  return (
    <dl className={cn('grid', adminSpacing.gap.md)}>
      {fields.map((field) => (
        <div key={field.label} className={cn('grid', adminSpacing.gap.xs)}>
          <dt className={adminTypography.labelMuted}>{field.label}</dt>
          <dd className={cn('m-0', adminTypography.body)}>{field.value}</dd>
        </div>
      ))}
    </dl>
  );
}

export function OpsHealth({
  stackHealth,
  doctor,
  doctorFetching,
  doctorError,
  hasDoctorSnapshot,
  fetching,
  error,
  hasSnapshot,
  refreshing,
  onRefresh,
  draftHostname,
  onDraftHostnameChange,
  probeResult,
  probing,
  probeError,
  onRunProbe,
}: OpsHealthProps) {
  const checks = doctor?.checks ?? [];

  return (
    <OpsPageWithLoad
      badge={
        <>
          {stackHealth ? <OpsStatusChip status={stackHealth.status} /> : null}
          {doctor?.overall ? <OpsStatusChip status={doctor.overall} /> : null}
        </>
      }
      blockingErrorTitle="Could not load stack health snapshot"
      fetchState={{ fetching, error, hasSnapshot }}
      refreshErrorTitle="Stack health refresh failed"
      title="Health"
      alerts={
        <>
          {doctorError && !hasDoctorSnapshot
            ? opsPanelError(doctorError, 'Could not load doctor checks')
            : null}
          {doctorError && hasDoctorSnapshot
            ? opsPanelError(doctorError, 'Doctor checks refresh failed')
            : null}
          {probeError ? opsPanelError(probeError, 'Domain probe failed') : null}
        </>
      }
      filters={
        <div>
          <Label htmlFor="ops-health-probe-hostname">Hostname</Label>
          <Input
            id="ops-health-probe-hostname"
            placeholder="track.example.com"
            value={draftHostname}
            onChange={(event) => onDraftHostnameChange(event.target.value)}
          />
        </div>
      }
      actions={
        <>
          <OpsActionGroup label="Snapshot">
            <Button
              disabled={refreshing}
              loading={refreshing}
              type="button"
              onClick={onRefresh}
            >
              Refresh
            </Button>
          </OpsActionGroup>
          <OpsActionGroup label="Domain probe">
            <Button disabled={probing} loading={probing} type="button" onClick={onRunProbe}>
              Run probe
            </Button>
          </OpsActionGroup>
        </>
      }
    >
      {!stackHealth ? (
        <EmptyState description="Stack health snapshot is unavailable." title="No health data" />
      ) : (
        <>
          <OpsStatGrid>
            <OpsStatPanel status={stackHealth.status} title="ClickHouse">
              <OpsKvRow label="Lag" value={formatSeconds(stackHealth.clickhouse_lag_seconds)} />
            </OpsStatPanel>

            <OpsStatPanel title="Outbox">
              <OpsKvRow
                label="Oldest pending"
                value={formatSeconds(stackHealth.outbox_oldest_pending_seconds)}
              />
            </OpsStatPanel>

            <OpsStatPanel title="Redis shards">
              <OpsKvRow
                label="Reachable"
                value={`${stackHealth.redis_shards_reachable}/${stackHealth.redis_shards_total}`}
              />
              <OpsKvRow
                label="Shard ping"
                value={stackHealth.redis_shard_reachable ? 'ok' : 'fail'}
              />
            </OpsStatPanel>

            <OpsStatPanel title="License">
              <OpsKvRow label="State" value={stackHealth.license_state} />
            </OpsStatPanel>

            <OpsStatPanel title="Cost sync">
              {stackHealth.cost_sync_last_success_seconds != null ? (
                <OpsKvRow
                  label="Last success"
                  value={formatSeconds(stackHealth.cost_sync_last_success_seconds)}
                />
              ) : (
                <OpsKvRow label="Last success" value="n/a" />
              )}
            </OpsStatPanel>

            {stackHealth.automation_worker_last_tick_seconds != null ? (
              <OpsStatPanel title="Automation worker">
                <OpsKvRow
                  label="Last tick"
                  value={formatSeconds(stackHealth.automation_worker_last_tick_seconds)}
                />
              </OpsStatPanel>
            ) : null}
          </OpsStatGrid>

          <OpsBlock title="Doctor checks">
            {doctorFetching && !hasDoctorSnapshot ? (
              <p>Loading doctor checks...</p>
            ) : checks.length === 0 ? (
              <p>No doctor checks returned.</p>
            ) : (
              <OpsHealthDoctorChecksTable checks={checks} />
            )}
            {doctor?.rtb_mode ? (
              <p>
                RTB mode: <span>{doctor.rtb_mode}</span>
              </p>
            ) : null}
            {doctor?.tracking_domain ? (
              <p>
                Tracking domain: <span>{doctor.tracking_domain}</span>
              </p>
            ) : null}
          </OpsBlock>

          <OpsBlock title="Domain probe">
            {!probeResult ? (
              <p>Enter a hostname and run probe to fetch live domain health.</p>
            ) : (
              <DomainProbeOverview probe={probeResult} />
            )}
          </OpsBlock>
        </>
      )}
    </OpsPageWithLoad>
  );
}
