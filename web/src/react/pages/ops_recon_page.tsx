import { useCallback, useEffect, useState } from 'react';
import { to } from '../../lib/to.js';
import { fetchReconRuns } from '../../helpers/ops_recon_api.js';
import type { ReconRunDTO } from '../../types/api/ops_extra.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { PaginationBar } from '../components/pagination_bar.js';
import { StatusBadge } from '../components/status_badge.js';

const PAGE_SIZE = 50;

type ServiceFilter = 'all' | 'management' | 'payment';

function formatPeriod(value: string | undefined): string {
  if (!value) return '—';
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toISOString().slice(0, 16).replace('T', ' ');
}

function statusTone(status: string): 'ok' | 'warn' | 'error' | 'neutral' {
  const s = status.toUpperCase();
  if (s === 'COMPLETED' || s === 'OK') return 'ok';
  if (s === 'FAILED' || s === 'ERROR') return 'error';
  if (s === 'RUNNING' || s === 'PENDING') return 'warn';
  return 'neutral';
}

function TableSkeleton({ cols }: { cols: number }) {
  return (
    <>
      {Array.from({ length: 5 }, (_, rowIndex) => (
        <tr key={`sk-${rowIndex}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, colIndex) => (
            <td key={`sk-${rowIndex}-${colIndex}`}><span className="skeleton-bar" /></td>
          ))}
        </tr>
      ))}
    </>
  );
}

/**
 * Reconciliation runs list for operators.
 */
export function OpsReconPage() {
  const [page, setPage] = useState(0);
  const [service, setService] = useState<ServiceFilter>('all');
  const [rows, setRows] = useState<ReconRunDTO[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const load = useCallback(async () => {
    setLoading(true);
    const [res, err] = await to(fetchReconRuns(service, PAGE_SIZE, page * PAGE_SIZE));
    setLoading(false);
    if (err) {
      setError(err);
      setRows([]);
      setTotal(0);
      return;
    }
    setError(null);
    setRows(res?.items ?? []);
    setTotal(res?.total ?? (res?.items?.length ?? 0));
  }, [service, page]);

  useEffect(() => {
    void load();
  }, [load]);

  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load reconciliation runs" />;

  const totalPages = Math.ceil(total / PAGE_SIZE) || 1;

  return (
    <section data-testid="ops-recon-view">
      <div className="page-header">
        <h1 className="page-header__title">Reconciliation runs</h1>
        <p className="text-muted text-sm">
          Management ledger vs spend checks and payment intent reconciliation.
        </p>
      </div>

      <label className="form-field mb-3" htmlFor="recon-service">
        Service
        <select
          id="recon-service"
          className="form-input form-input--sm"
          value={service}
          onChange={(e) => {
            setService(e.target.value as ServiceFilter);
            setPage(0);
          }}
        >
          <option value="all">All</option>
          <option value="management">Management</option>
          <option value="payment">Payment</option>
        </select>
      </label>

      <FilterToolbar
        pagination={totalPages > 1 ? (
          <PaginationBar
            label={`${page + 1} / ${totalPages}`}
            prevDisabled={page === 0}
            nextDisabled={page >= totalPages - 1}
            onPrev={() => setPage((p) => Math.max(0, p - 1))}
            onNext={() => setPage((p) => p + 1)}
          />
        ) : null}
      />

      <div className="table-wrapper elevation-raised">
        <table className="data-table" data-testid="ops-recon-table">
          <thead>
            <tr>
              <th scope="col">Service</th>
              <th scope="col">Period</th>
              <th scope="col">Status</th>
              <th scope="col">Discrepancies</th>
              <th scope="col">Created</th>
            </tr>
          </thead>
          <tbody>
            {loading && rows.length === 0 ? <TableSkeleton cols={5} /> : null}
            {!loading && rows.length === 0 ? (
              <tr>
                <td colSpan={5} className="text-center p-6">
                  <div className="empty-state">
                    <div className="empty-state__title">No reconciliation runs</div>
                    <div className="empty-state__desc text-muted text-sm">
                      Scheduled recon jobs will appear here after they complete.
                    </div>
                  </div>
                </td>
              </tr>
            ) : null}
            {rows.map((row) => (
              <tr key={`${row.service}-${row.created_at}-${row.period_start}`}>
                <td>{row.service}</td>
                <td className="font-mono text-xs">
                  {`${formatPeriod(row.period_start)} → ${formatPeriod(row.period_end)}`}
                </td>
                <td>
                  <StatusBadge status={statusTone(row.status)} label={row.status} kind="service" />
                </td>
                <td>{String(row.discrepancies_found ?? row.findings_count ?? '—')}</td>
                <td className="text-muted text-xs">
                  {row.created_at ? new Date(row.created_at).toLocaleString() : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
