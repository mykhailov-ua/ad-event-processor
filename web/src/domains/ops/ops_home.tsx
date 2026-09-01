import { PageChrome } from '@/components/system/page_chrome';
import { PageToolbar } from '@/components/system/page_toolbar';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { PanelSection, StatPanel, StatRow } from '@/components/system/stat_panel';
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
import type { OpsHomeSnapshot } from '@/api/types';
import { OpsNav } from '@/domains/ops/ops_nav';
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

function statusBadgeVariant(
  status: string | undefined,
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = (status ?? '').toLowerCase();
  if (normalized === 'ok' || normalized === 'healthy' || normalized === 'pass') {
    return 'default';
  }
  if (normalized === 'degraded' || normalized === 'warn' || normalized === 'warning') {
    return 'secondary';
  }
  if (normalized === 'critical' || normalized === 'fail' || normalized === 'down') {
    return 'destructive';
  }
  return 'outline';
}

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
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load ops snapshot" message={error.message} />;
  }

  if (!snapshot) {
    return <EmptyState title="No ops data" description="Ops health snapshot is unavailable." />;
  }

  const { doctor, stackHealth, dashboardSummary } = snapshot;
  const checks = doctor.checks ?? [];
  const services = dashboardSummary.services ?? [];

  return (
    <PageChrome
      title="Ops"
      badge={
        doctor.overall ? (
          <Badge variant={statusBadgeVariant(doctor.overall)}>{doctor.overall}</Badge>
        ) : undefined
      }
    >
      <OpsNav />

      <PageToolbar>
        <Button disabled={reloadingRoles} onClick={onReloadRoles} type="button" variant="outline">
          {reloadingRoles ? 'Reloading...' : 'Reload roles'}
        </Button>
        <Button
          disabled={downloadingBundle}
          onClick={onDownloadSupportBundle}
          type="button"
          variant="outline"
        >
          {downloadingBundle ? 'Downloading...' : 'Download support bundle'}
        </Button>
        {rolesReloadMessage ? (
          <p className="text-sm text-muted-foreground" role="status">
            {rolesReloadMessage}
          </p>
        ) : null}
      </PageToolbar>
      {rolesReloadError ? (
        <ErrorBlock title="Could not reload roles" message={rolesReloadError.message} />
      ) : null}
      {bundleDownloadError ? (
        <ErrorBlock title="Could not download support bundle" message={bundleDownloadError.message} />
      ) : null}

      <div className="grid gap-4 lg:grid-cols-[repeat(auto-fit,minmax(300px,1fr))]">
        <StatPanel
          meta={<Badge variant={statusBadgeVariant(stackHealth.status)}>{stackHealth.status}</Badge>}
          title="Stack health"
        >
          <StatRow label="ClickHouse lag" value={formatSeconds(stackHealth.clickhouse_lag_seconds)} />
          <StatRow
            label="Outbox oldest pending"
            value={formatSeconds(stackHealth.outbox_oldest_pending_seconds)}
          />
          <StatRow
            label="Redis shards"
            value={`${stackHealth.redis_shards_reachable}/${stackHealth.redis_shards_total}`}
          />
          <StatRow label="License" value={stackHealth.license_state} />
          {stackHealth.cost_sync_last_success_seconds != null ? (
            <StatRow
              label="Cost sync last success"
              value={formatSeconds(stackHealth.cost_sync_last_success_seconds)}
            />
          ) : null}
        </StatPanel>

        <StatPanel title="Dashboard summary">
          {dashboardSummary.generated_at ? (
            <StatRow
              label="Generated"
              value={displayTimestamp(
                dashboardSummary.generated_at,
                dashboardSummary.generated_at_display,
              )}
            />
          ) : null}
          {dashboardSummary.rps_estimate != null ? (
            <StatRow label="RPS estimate" value={dashboardSummary.rps_estimate.toFixed(2)} />
          ) : null}
          {dashboardSummary.outbox_pending != null ? (
            <StatRow label="Outbox pending" value={dashboardSummary.outbox_pending} />
          ) : null}
          {dashboardSummary.drift_micro_max != null ? (
            <StatRow label="Drift max (micro)" value={dashboardSummary.drift_micro_max} />
          ) : null}
          {dashboardSummary.drift_alert != null ? (
            <StatRow label="Drift alert" value={dashboardSummary.drift_alert ? 'yes' : 'no'} />
          ) : null}
          {dashboardSummary.emergency_breaker ? (
            <StatRow label="Emergency breaker" value={dashboardSummary.emergency_breaker} />
          ) : null}
          {doctor.rtb_mode ? <StatRow label="RTB mode" value={doctor.rtb_mode} /> : null}
          {doctor.tracking_domain ? (
            <StatRow label="Tracking domain" value={doctor.tracking_domain} />
          ) : null}
        </StatPanel>
      </div>

      {services.length > 0 ? (
        <PanelSection title="Services">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Detail</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {services.map((service) => (
                  <TableRow key={service.id ?? service.name}>
                    <TableCell>{service.name ?? service.id}</TableCell>
                    <TableCell>
                      {service.status ? (
                        <Badge variant={statusBadgeVariant(service.status)}>{service.status}</Badge>
                      ) : (
                        ''
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">{service.detail ?? ''}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </PanelSection>
      ) : null}

      <PanelSection title="Doctor checks">
        {checks.length === 0 ? (
          <p className="px-5 py-4 text-sm text-muted-foreground">No doctor checks returned.</p>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Check</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Message</TableHead>
                  <TableHead>Hint</TableHead>
                  <TableHead className="text-right">Latency</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {checks.map((check) => (
                  <TableRow key={check.id ?? check.message}>
                    <TableCell>{check.id ?? ''}</TableCell>
                    <TableCell>
                      {check.status ? (
                        <Badge variant={statusBadgeVariant(check.status)}>{check.status}</Badge>
                      ) : (
                        ''
                      )}
                    </TableCell>
                    <TableCell>{check.message ?? ''}</TableCell>
                    <TableCell className="text-muted-foreground">{check.hint ?? ''}</TableCell>
                    <TableCell className="text-right">
                      {check.latency_ms != null ? `${check.latency_ms} ms` : ''}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </PanelSection>

      {error && hasSnapshot ? (
        <ErrorBlock title="Refresh failed" message={error.message} />
      ) : null}
    </PageChrome>
  );
}
