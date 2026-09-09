import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/shell/empty_state';
import type { DomainHealth, DoctorSummary, StackHealthSnapshot } from '@/api/types';
import { OpsKvRow, OpsStatGrid, OpsStatPanel } from '@/domains/ops/ops_stat_panel';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageWithLoad } from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import {
  OpsBlock,
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';
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

function formatSeconds(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(1)}s`;
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
              <OpsTable
                horizontalScroll
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
                    <OpsTableCell>{check.hint ?? ''}</OpsTableCell>
                    <OpsTableCell numeric>
                      {check.latency_ms != null ? `${check.latency_ms} ms` : ''}
                    </OpsTableCell>
                  </OpsTableRow>
                ))}
              </OpsTable>
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
              <OpsTable
                head={
                  <OpsTableHeaderRow>
                    <OpsTableHead>Field</OpsTableHead>
                    <OpsTableHead>Value</OpsTableHead>
                  </OpsTableHeaderRow>
                }
              >
                <OpsTableRow>
                  <OpsTableCell>Hostname</OpsTableCell>
                  <OpsTableCell>{probeResult.hostname}</OpsTableCell>
                </OpsTableRow>
                <OpsTableRow>
                  <OpsTableCell>Health</OpsTableCell>
                  <OpsTableCell>
                    <OpsStatusChip status={probeResult.health_status} />
                  </OpsTableCell>
                </OpsTableRow>
                <OpsTableRow>
                  <OpsTableCell>SSL</OpsTableCell>
                  <OpsTableCell>
                    <OpsStatusChip status={probeResult.ssl_status} />
                  </OpsTableCell>
                </OpsTableRow>
                <OpsTableRow>
                  <OpsTableCell>Role</OpsTableCell>
                  <OpsTableCell>{probeResult.role}</OpsTableCell>
                </OpsTableRow>
                {probeResult.http_status != null ? (
                  <OpsTableRow>
                    <OpsTableCell>HTTP status</OpsTableCell>
                    <OpsTableCell numeric>{probeResult.http_status}</OpsTableCell>
                  </OpsTableRow>
                ) : null}
                {probeResult.probe_latency_ms != null ? (
                  <OpsTableRow>
                    <OpsTableCell>Probe latency</OpsTableCell>
                    <OpsTableCell numeric>{probeResult.probe_latency_ms} ms</OpsTableCell>
                  </OpsTableRow>
                ) : null}
                {probeResult.probe_detail ? (
                  <OpsTableRow>
                    <OpsTableCell>Detail</OpsTableCell>
                    <OpsTableCell>{probeResult.probe_detail}</OpsTableCell>
                  </OpsTableRow>
                ) : null}
                {probeResult.last_probe_at ? (
                  <OpsTableRow>
                    <OpsTableCell>Last probe</OpsTableCell>
                    <OpsTableCell>
                      {displayTimestamp(probeResult.last_probe_at)}
                    </OpsTableCell>
                  </OpsTableRow>
                ) : null}
              </OpsTable>
            )}
          </OpsBlock>
        </>
      )}
    </OpsPageWithLoad>
  );
}
