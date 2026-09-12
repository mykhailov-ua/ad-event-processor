import { useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import { EmptyState } from '@/shell/empty_state';
import type { DashboardSummary, DoctorSummary, OpsHomeSnapshot } from '@/api/types';
import { OpsKvRow, OpsStatGrid, OpsStatPanel } from '@/domains/ops/ops_stat_panel';
import { OpsFraudPresetPanelWithWorkspace } from '@/domains/ops/ops_fraud_preset_panel';
import { OpsActionGroup, OpsPageWithLoad } from '@/domains/ops/ops_page_shell';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import { OpsBlock } from '@/domains/ops/ops_table';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { TableHost } from '@/shell/ui_bands';
import { displayTimestamp } from '@/lib/display';

export type OpsHomeProps = {
  snapshot: OpsHomeSnapshot | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  reloadingRoles: boolean;
  rolesReloadError: Error | undefined;
  rolesReloadMessage: string | undefined;
  onReloadRoles: () => void;
  downloadingBundle: boolean;
  bundleDownloadError: Error | undefined;
  onDownloadSupportBundle: () => void;
};

type DashboardService = NonNullable<DashboardSummary['services']>[number];
type DoctorCheck = NonNullable<DoctorSummary['checks']>[number];

function formatSeconds(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(1)}s`;
}

function serviceRowId(service: DashboardService): string | undefined {
  return service.id ?? service.name ?? undefined;
}

function serviceRowLabel(service: DashboardService): string {
  return service.name ?? service.id ?? 'Service';
}

function buildServiceOverviewFields(service: DashboardService): DirectoryOverviewField[] {
  return [
    { label: 'Name', value: service.name ?? service.id ?? '-' },
    {
      label: 'Status',
      value: service.status ? <OpsStatusChip status={service.status} /> : '-',
    },
    { label: 'Detail', value: service.detail ?? '-' },
  ];
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

function OpsHomeServicesTable({ services }: { services: DashboardService[] }) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const recordById = useMemo(() => directoryRecordMap(services, serviceRowId), [services]);
  const rows = useMemo(
    () => directoryOperateRows(services, serviceRowId, serviceRowLabel),
    [services]
  );

  return (
    <TableHost>
      <DirectorySelectOverviewTable
        buildOverviewFields={buildServiceOverviewFields}
        nameColumnLabel="Service"
        overviewTitle={(service) => serviceRowLabel(service)}
        recordById={recordById}
        rows={rows}
        selectedId={selectedId}
        onSelectedIdChange={setSelectedId}
      />
    </TableHost>
  );
}

function OpsHomeDoctorChecksTable({ checks }: { checks: DoctorCheck[] }) {
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

export function OpsHome({
  snapshot,
  fetching,
  error,
  hasSnapshot,
  reloadingRoles,
  rolesReloadError,
  rolesReloadMessage,
  onReloadRoles,
  downloadingBundle,
  bundleDownloadError,
  onDownloadSupportBundle,
}: OpsHomeProps) {
  const doctor = snapshot?.doctor ?? { checks: [] };
  const stackHealthRaw = snapshot?.stackHealth;
  const stackHealth =
    stackHealthRaw && 'clickhouse_lag_seconds' in stackHealthRaw ? stackHealthRaw : undefined;
  const dashboardSummary = snapshot?.dashboardSummary ?? { services: [] };
  const checks = doctor.checks ?? [];
  const services = dashboardSummary.services ?? [];

  return (
    <OpsPageWithLoad
      badge={snapshot && doctor.overall ? <OpsStatusChip status={doctor.overall} /> : undefined}
      blockingErrorTitle="Could not load ops snapshot"
      fetchState={{ fetching, error, hasSnapshot }}
      title="Ops"
      alerts={
        <>
          {rolesReloadError ? opsPanelError(rolesReloadError, 'Could not reload roles') : null}
          {bundleDownloadError
            ? opsPanelError(bundleDownloadError, 'Could not download support bundle')
            : null}
        </>
      }
      actions={
        <OpsActionGroup label="Support">
          <Button type="button" variant="outline" disabled={reloadingRoles} onClick={onReloadRoles}>
            {reloadingRoles ? 'Reloading...' : 'Reload roles'}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={downloadingBundle}
            onClick={onDownloadSupportBundle}
          >
            {downloadingBundle ? 'Downloading...' : 'Download support bundle'}
          </Button>
          {rolesReloadMessage ? (
            <span role="status">{rolesReloadMessage}</span>
          ) : null}
        </OpsActionGroup>
      }
    >
      {!snapshot ? (
        <EmptyState description="Ops health snapshot is unavailable." title="No ops data" />
      ) : (
        <>
          <OpsStatGrid>
            {stackHealth ? (
              <OpsStatPanel status={stackHealth.status} title="Stack health">
                <OpsKvRow
                  label="ClickHouse lag"
                  value={formatSeconds(stackHealth.clickhouse_lag_seconds)}
                />
                <OpsKvRow
                  label="Outbox oldest pending"
                  value={formatSeconds(stackHealth.outbox_oldest_pending_seconds)}
                />
                <OpsKvRow
                  label="Redis shards"
                  value={`${stackHealth.redis_shards_reachable}/${stackHealth.redis_shards_total}`}
                />
                <OpsKvRow label="License" value={stackHealth.license_state} />
                {stackHealth.cost_sync_last_success_seconds != null ? (
                  <OpsKvRow
                    label="Cost sync last success"
                    value={formatSeconds(stackHealth.cost_sync_last_success_seconds)}
                  />
                ) : null}
              </OpsStatPanel>
            ) : null}

            <OpsStatPanel title="Dashboard summary">
              {dashboardSummary.generated_at ? (
                <OpsKvRow
                  label="Generated"
                  value={displayTimestamp(
                    dashboardSummary.generated_at,
                    dashboardSummary.generated_at_display
                  )}
                />
              ) : null}
              {dashboardSummary.rps_estimate != null ? (
                <OpsKvRow label="RPS estimate" value={dashboardSummary.rps_estimate.toFixed(2)} />
              ) : null}
              {dashboardSummary.outbox_pending != null ? (
                <OpsKvRow label="Outbox pending" value={dashboardSummary.outbox_pending} />
              ) : null}
              {dashboardSummary.drift_micro_max != null ? (
                <OpsKvRow label="Drift max (micro)" value={dashboardSummary.drift_micro_max} />
              ) : null}
              {dashboardSummary.drift_alert != null ? (
                <OpsKvRow label="Drift alert" value={dashboardSummary.drift_alert ? 'yes' : 'no'} />
              ) : null}
              {dashboardSummary.emergency_breaker ? (
                <OpsKvRow label="Emergency breaker" value={dashboardSummary.emergency_breaker} />
              ) : null}
              {doctor.rtb_mode ? <OpsKvRow label="RTB mode" value={doctor.rtb_mode} /> : null}
              {doctor.tracking_domain ? (
                <OpsKvRow label="Tracking domain" value={doctor.tracking_domain} />
              ) : null}
            </OpsStatPanel>
          </OpsStatGrid>

          {services.length > 0 ? (
            <OpsBlock title="Services">
              <OpsHomeServicesTable services={services} />
            </OpsBlock>
          ) : null}

          <OpsBlock title="Doctor checks">
            {checks.length === 0 ? (
              <p>No doctor checks returned.</p>
            ) : (
              <OpsHomeDoctorChecksTable checks={checks} />
            )}
          </OpsBlock>

          <OpsFraudPresetPanelWithWorkspace />
        </>
      )}
    </OpsPageWithLoad>
  );
}
