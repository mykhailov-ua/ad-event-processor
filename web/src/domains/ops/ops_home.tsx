import { Button } from '@/components/ui/button';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import type { OpsHomeSnapshot } from '@/api/types';
import { OpsKvRow, OpsStatGrid, OpsStatPanel } from '@/domains/ops/ops_stat_panel';
import {
  OpsActionGroup,
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import { OpsBlock, OpsTable, OpsTableCell, OpsTableHead, OpsTableHeaderRow, OpsTableRow } from '@/domains/ops/ops_table';
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

function formatSeconds(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(1)}s`;
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
  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError
        error={error}
        pageTitle="Ops"
        title="Could not load ops snapshot"
      />
    );
  }

  if (!snapshot) {
    return (
      <OpsPageShell title="Ops">
        <EmptyState description="Ops health snapshot is unavailable." title="No ops data" />
      </OpsPageShell>
    );
  }

  const doctor = snapshot.doctor ?? { checks: [] };
  const stackHealth = snapshot.stackHealth ?? { status: 'unknown' };
  const dashboardSummary = snapshot.dashboardSummary ?? { services: [] };
  const checks = doctor.checks ?? [];
  const services = dashboardSummary.services ?? [];

  return (
    <OpsPageShell
      badge={doctor.overall ? <OpsStatusChip status={doctor.overall} /> : undefined}
      title="Ops"
      actions={
        <OpsActionGroup label="Support">
          <Button
            type="button"
            variant="outline"
            disabled={reloadingRoles}
            onClick={onReloadRoles}
          >
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
            <span className="text-muted-foreground" role="status">
              {rolesReloadMessage}
            </span>
          ) : null}
        </OpsActionGroup>
      }
    >
      {rolesReloadError ? (
        <ErrorBlock error={rolesReloadError} title="Could not reload roles" />
      ) : null}
      {bundleDownloadError ? (
        <ErrorBlock error={bundleDownloadError} title="Could not download support bundle" />
      ) : null}

      <OpsStatGrid>
        <OpsStatPanel status={stackHealth.status} title="Stack health">
          <OpsKvRow label="ClickHouse lag" value={formatSeconds(stackHealth.clickhouse_lag_seconds)} />
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

        <OpsStatPanel title="Dashboard summary">
          {dashboardSummary.generated_at ? (
            <OpsKvRow
              label="Generated"
              value={displayTimestamp(
                dashboardSummary.generated_at,
                dashboardSummary.generated_at_display,
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
          <OpsTable
            head={
              <OpsTableHeaderRow>
                <OpsTableHead>Name</OpsTableHead>
                <OpsTableHead>Status</OpsTableHead>
                <OpsTableHead>Detail</OpsTableHead>
              </OpsTableHeaderRow>
            }
          >
            {services.map((service) => (
              <OpsTableRow key={service.id ?? service.name}>
                <OpsTableCell>{service.name ?? service.id}</OpsTableCell>
                <OpsTableCell>
                  <OpsStatusChip status={service.status} />
                </OpsTableCell>
                <OpsTableCell className="text-muted-foreground">
                  {service.detail ?? ''}
                </OpsTableCell>
              </OpsTableRow>
            ))}
          </OpsTable>
        </OpsBlock>
      ) : null}

      <OpsBlock title="Doctor checks">
        {checks.length === 0 ? (
          <p className="text-muted-foreground">No doctor checks returned.</p>
        ) : (
          <OpsTable
            head={
              <OpsTableHeaderRow>
                <OpsTableHead>Check</OpsTableHead>
                <OpsTableHead>Status</OpsTableHead>
                <OpsTableHead>Message</OpsTableHead>
                <OpsTableHead>Hint</OpsTableHead>
                <OpsTableHead numeric>Latency</OpsTableHead>
              </OpsTableHeaderRow>
            }
          >
            {checks.map((check) => (
              <OpsTableRow key={check.id ?? check.message}>
                <OpsTableCell>{check.id ?? ''}</OpsTableCell>
                <OpsTableCell>
                  <OpsStatusChip status={check.status} />
                </OpsTableCell>
                <OpsTableCell>{check.message ?? ''}</OpsTableCell>
                <OpsTableCell className="text-muted-foreground">
                  {check.hint ?? ''}
                </OpsTableCell>
                <OpsTableCell numeric>
                  {check.latency_ms != null ? `${check.latency_ms} ms` : ''}
                </OpsTableCell>
              </OpsTableRow>
            ))}
          </OpsTable>
        )}
      </OpsBlock>

      {error && hasSnapshot ? <ErrorBlock error={error} title="Refresh failed" /> : null}
    </OpsPageShell>
  );
}
