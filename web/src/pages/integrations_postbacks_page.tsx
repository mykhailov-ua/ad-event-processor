import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  fetchPostbackCampaignStatus,
  fetchPostbackDlq,
  retryPostbackDlq,
  type PostbackCampaignStatusRow,
  type PostbackDlqRow,
} from '../helpers/postback_api.js';
import { to } from '../lib/to.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';

function TableSkeleton({ cols, rows = 4 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <tr key={`sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`sk-${i}-${j}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

function formatTs(iso?: string): string {
  if (!iso) return '-';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
}

export function IntegrationsPostbacksPage() {
  const canWrite = can(auth.getUser()?.permissions ?? [], 'campaigns:write');
  const [dlq, setDlq] = useState<PostbackDlqRow[]>([]);
  const [status, setStatus] = useState<PostbackCampaignStatusRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [retryingId, setRetryingId] = useState<string | number | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [dlqRes, statusRes] = await Promise.all([
      to(fetchPostbackDlq()),
      to(fetchPostbackCampaignStatus()),
    ]);
    setLoading(false);
    if (dlqRes[1]) {
      setError(dlqRes[1]);
      return;
    }
    if (statusRes[1]) {
      setError(statusRes[1]);
      return;
    }
    setDlq(dlqRes[0] ?? []);
    setStatus(statusRes[0] ?? []);
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const retry = async (rowId: string | number) => {
    if (!canWrite) return;
    setRetryingId(rowId);
    const [, err] = await to(retryPostbackDlq(rowId));
    setRetryingId(null);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Retry failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Retry queued', message: `DLQ #${rowId}` });
    void reload();
  };

  return (
    <div className="page stack" data-testid="integrations-postbacks-page">
      <header className="page-header">
        <h1 className="page-title">Postbacks & CAPI</h1>
        <p className="text-muted text-sm">
          Failed outbound dispatches and per-campaign CAPI health. Configure providers on each
          campaign&apos;s <Link to="/campaigns">CAPI &amp; Postbacks</Link> tab.
        </p>
      </header>

      {error ? <ErrorBlock error={error} /> : null}

      <section className="section-card stack" data-testid="postback-campaign-status-panel">
        <h2 className="subsection-title">Dispatch status</h2>
        <p className="text-muted text-sm">
          Last successful outbound dispatch and pending DLQ count per configured campaign.
        </p>
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Campaign</th>
                <th scope="col">Provider</th>
                <th scope="col">Last success</th>
                <th scope="col">DLQ pending</th>
                <th scope="col" />
              </tr>
            </thead>
            <tbody>
              {loading ? <TableSkeleton cols={5} rows={3} /> : null}
              {!loading && status.length === 0 ? (
                <tr>
                  <td colSpan={5} className="text-muted">
                    No postback configs yet.
                  </td>
                </tr>
              ) : null}
              {!loading &&
                status.map((row) => (
                  <tr key={row.campaign_id} data-testid={`postback-status-${row.campaign_id}`}>
                    <td className="font-mono text-sm">{row.campaign_id}</td>
                    <td>{row.provider}</td>
                    <td>{formatTs(row.last_success_at)}</td>
                    <td>{row.dlq_pending_count}</td>
                    <td>
                      <Link to={`/campaigns/${row.campaign_id}?tab=postbacks`} className="text-sm">
                        Open {'->'}
                      </Link>
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="section-card stack" data-testid="postback-dlq-panel">
        <h2 className="subsection-title">Dead letter queue</h2>
        <p className="text-muted text-sm">
          Retry re-queues a failed outbound postback via the outbox worker.
        </p>
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">ID</th>
                <th scope="col">Campaign</th>
                <th scope="col">Event</th>
                <th scope="col">Failures</th>
                <th scope="col">Status</th>
                <th scope="col">Error</th>
                <th scope="col" />
              </tr>
            </thead>
            <tbody>
              {loading ? <TableSkeleton cols={7} rows={4} /> : null}
              {!loading && dlq.length === 0 ? (
                <tr>
                  <td colSpan={7} className="text-muted">
                    No failed postbacks.
                  </td>
                </tr>
              ) : null}
              {!loading &&
                dlq.map((row) => {
                  const rowId = row.id;
                  const rowStatus = typeof row.status === 'string' ? row.status : '';
                  const canRetry =
                    canWrite &&
                    rowStatus !== 'RETRIED' &&
                    (typeof rowId === 'string' || typeof rowId === 'number');
                  return (
                    <tr key={String(rowId)} data-testid={`postback-dlq-row-${rowId}`}>
                      <td>{String(rowId ?? '')}</td>
                      <td className="font-mono text-sm">{String(row.campaign_id ?? '-')}</td>
                      <td>{typeof row.event_type === 'string' ? row.event_type : '-'}</td>
                      <td>{String(row.failures_count ?? 0)}</td>
                      <td>{rowStatus || '-'}</td>
                      <td className="text-sm text-muted">{String(row.last_error ?? '-')}</td>
                      <td>
                        {canRetry ? (
                          <Button
                            label="Retry"
                            variant="secondary"
                            size="sm"
                            loading={retryingId === rowId}
                            disabled={retryingId === rowId}
                            data-testid={`postback-dlq-retry-${rowId}`}
                            onClick={() => void retry(rowId!)}
                          />
                        ) : null}
                      </td>
                    </tr>
                  );
                })}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
