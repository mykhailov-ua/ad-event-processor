import { useCallback, useEffect, useState } from 'react';
import { to } from '../../lib/to.js';
import { api } from '../../helpers/api_client.js';
import { apiBlobResult } from '../../helpers/api_blob.js';
import { can } from '../../helpers/permissions.js';
import * as auth from '../../helpers/auth.js';
import { mapServiceError } from '../../helpers/service_error.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import type { AuditLogRow } from '../../types/api/index.js';
import { Button } from '../components/button.js';
import { Checkbox } from '../components/checkbox.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { PaginationBar } from '../components/pagination_bar.js';

const PAGE_SIZE = 50;

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
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
 * Audit log viewer with PII redaction toggle and CSV export.
 */
export function AuditPage() {
  const user = auth.getUser();
  const canExport = can(user?.permissions ?? [], 'audit:read');

  const [page, setPage] = useState(0);
  const [redactPii, setRedactPii] = useState(true);
  const [rows, setRows] = useState<AuditLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [exportBusy, setExportBusy] = useState(false);
  const [exportTruncated, setExportTruncated] = useState(false);
  const [exportNextCursor, setExportNextCursor] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({
      limit: String(PAGE_SIZE),
      offset: String(page * PAGE_SIZE),
      redact_pii: redactPii ? 'true' : 'false',
    });
    const [res, err] = await to(api<AuditLogRow[]>(`/api/v1/audit?${params.toString()}`));
    setLoading(false);
    if (err) {
      setError(err);
      return;
    }
    setError(null);
    setRows(Array.isArray(res?.data) ? res.data : []);
    const hdr = res?.headers?.get?.('X-Total-Count');
    setTotal(hdr ? Number(hdr) : (Array.isArray(res?.data) ? res.data.length : 0));
  }, [page, redactPii]);

  useEffect(() => {
    void load();
  }, [load]);

  const exportCsv = async () => {
    if (!canExport || exportBusy) return;
    setExportBusy(true);
    setExportTruncated(false);
    setExportNextCursor(null);
    const params = new URLSearchParams({
      format: 'csv',
      redact_pii: redactPii ? 'true' : 'false',
    });
    const [result, err] = await to(apiBlobResult(`/api/v1/audit/export?${params.toString()}`));
    setExportBusy(false);
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    setExportTruncated(result.truncated);
    setExportNextCursor(result.nextCursor);
    downloadBlob(result.blob, 'audit-export.csv');
  };

  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load audit log" />;

  const totalPages = Math.ceil(total / PAGE_SIZE) || 1;

  return (
    <>
      <div className="page-header">
        <div className="page-header__row">
          <h1 className="page-header__title">Audit log</h1>
          {canExport ? (
            <Button
              label="Export CSV"
              variant="secondary"
              size="sm"
              className="ml-auto"
              loading={exportBusy}
              data-testid="audit-export-csv"
              onClick={() => void exportCsv()}
            />
          ) : null}
        </div>
        <p className="text-muted text-sm">{total} entries</p>
      </div>

      {exportTruncated ? (
        <p className="text-warning text-sm mb-2" data-testid="audit-export-truncated">
          {`Last export was truncated at cursor ${exportNextCursor ?? '—'}.`}
        </p>
      ) : null}

      <Checkbox
        className="form-check mb-3"
        label="Redact PII in changes/metadata"
        checked={redactPii}
        onChange={(checked) => {
          setRedactPii(checked);
          setPage(0);
        }}
      />

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

      <div className="table-wrapper table-wrapper--scroll elevation-raised">
        <table className="data-table" aria-label="Audit log">
          <thead>
            <tr>
              <th scope="col">Time</th>
              <th scope="col">Action</th>
              <th scope="col">Target</th>
              <th scope="col">Admin</th>
            </tr>
          </thead>
          <tbody>
            {loading && rows.length === 0 ? <TableSkeleton cols={4} /> : null}
            {!loading && rows.length === 0 ? (
              <tr>
                <td colSpan={4} className="text-center p-6">
                  <div className="empty-state">
                    <div className="empty-state__title">No audit entries</div>
                    <div className="empty-state__desc text-muted text-sm">
                      Admin actions will appear here as they occur.
                    </div>
                  </div>
                </td>
              </tr>
            ) : null}
            {rows.map((row) => (
              <tr key={`${row.created_at}-${row.action}-${row.target_id}`}>
                <td>{row.created_at ? new Date(row.created_at).toLocaleString() : '—'}</td>
                <td>{row.action ?? '—'}</td>
                <td>
                  {row.target_type ?? '—'}
                  {row.target_id ? ` · ${row.target_id.slice(0, 8)}…` : ''}
                </td>
                <td className="font-mono text-hint">{row.admin_id?.slice(0, 8) ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
