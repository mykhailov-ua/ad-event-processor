import { useCallback, useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { api, ApiError } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import type { IncidentSnapshot, ShardHealthStatus } from '../types/index.js';
import { formatYesNo } from '../helpers/display_labels.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';

type ShardHealthReport = IncidentSnapshot;

type CatchupMetricResponse = {
  points?: Array<{ ts?: string; value?: number }>;
};

export function OpsShardsPage() {
  const user = auth.getUser();
  const canCatchup = can(user?.permissions ?? [], 'shards:write');

  const [report, setReport] = useState<ShardHealthReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown | null>(null);
  const [catchupLoading, setCatchupLoading] = useState(false);
  const [catchupLastSuccess, setCatchupLastSuccess] = useState<string | null>(null);

  const loadCatchupMetric = useCallback(async () => {
    const [res] = await to(
      api<CatchupMetricResponse>(
        '/api/v1/ops/dashboard/metrics?range=24h&name=ad_shard0_catchup_last_success_timestamp'
      )
    );
    const points = res?.data?.points ?? [];
    let latest = 0;
    for (const point of points) {
      const value = Number(point?.value ?? 0);
      if (value > latest) latest = value;
    }
    setCatchupLastSuccess(latest > 0 ? new Date(latest * 1000).toLocaleString() : null);
  }, []);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { data } = await api<ShardHealthReport>('/api/v1/ops/shards');
      setReport(data ?? null);
    } catch (err) {
      if (err instanceof ApiError && err.status === 503 && err.payload) {
        setReport(err.payload as ShardHealthReport);
      } else {
        setError(err);
      }
    } finally {
      setLoading(false);
      await loadCatchupMetric();
    }
  }, [loadCatchupMetric]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const shards = report?.shards ?? [];
  const shard0 = shards.find((s) => s.shard_id === 0);
  const catchupTarget = shard0 && shard0.config_version_synced === false ? shard0 : null;

  const runCatchup = async () => {
    if (!canCatchup || catchupLoading) return;
    setCatchupLoading(true);
    const [, err] = await to(
      apiConfirmed('/api/v1/ops/shards/0/catchup', { method: 'POST', body: '{}' })
    );
    setCatchupLoading(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    pushToastMessage({ title: 'Catch-up started', message: 'Shard 0 config sync worker running.' });
    void reload();
  };

  if (loading) return <span className="text-muted">Loading...</span>;
  if (error) return <ErrorBlock error={error} />;

  return (
    <>
      <div className="page-header">
        <Breadcrumbs items={[{ label: 'Operations', href: '/ops' }, { label: 'Redis shards' }]} />
        <div className="page-header__row">
          <h1 className="page-header__title">Redis shards</h1>
          {catchupTarget && canCatchup ? (
            <Button
              label={catchupLoading ? 'Running...' : 'Run catch-up'}
              variant="danger"
              size="sm"
              className="ml-auto"
              data-testid="shard0-catchup-btn"
              loading={catchupLoading}
              disabled={catchupLoading}
              onClick={() => void runCatchup()}
            />
          ) : null}
        </div>
      </div>

      {catchupTarget ? (
        <div className="stub-banner mb-4" role="status" data-testid="shard0-catchup-banner">
          {`Shard 0 config is out of sync (lag ${catchupTarget.config_version_lag ?? 0}). Run catch-up to reconcile pub/sub keys.`}
        </div>
      ) : null}

      {catchupLastSuccess ? (
        <p className="text-muted text-sm mb-4" data-testid="shard0-catchup-metric">
          {`Last successful shard 0 catch-up: ${catchupLastSuccess}`}
        </p>
      ) : null}

      {(report?.errors?.length ?? 0) > 0 ? (
        <div className="stub-banner mb-4">
          {`Partial: ${(report?.errors ?? []).map((e) => e.source).join(', ')}`}
        </div>
      ) : null}

      <div className="table-wrapper elevation-raised">
        <table className="data-table">
          <thead>
            <tr>
              <th scope="col">Shard</th>
              <th scope="col">Ping OK</th>
              <th scope="col">Latency ms</th>
              <th scope="col">Config lag</th>
              <th scope="col">Synced</th>
            </tr>
          </thead>
          <tbody>
            {shards.length === 0 ? (
              <tr>
                <td colSpan={5} className="text-muted text-center p-6">
                  No data
                </td>
              </tr>
            ) : null}
            {shards.map((s: ShardHealthStatus) => (
              <tr key={s.shard_id} className={!s.ping_ok ? 'data-table__row--danger' : undefined}>
                <td>{String(s.shard_id)}</td>
                <td>{formatYesNo(s.ping_ok)}</td>
                <td>{s.ping_latency_ms?.toFixed(1) ?? '-'}</td>
                <td>{String(s.config_version_lag ?? 0)}</td>
                <td>{formatYesNo(s.config_version_synced)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
