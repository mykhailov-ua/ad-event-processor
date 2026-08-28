import { useCallback, useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import type { BlacklistEntryDTO, BlacklistListResponse } from '../types/index.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { FormField } from '../components/form_field.js';
import { PaginationBar } from '../components/pagination_bar.js';

const PAGE_SIZE = 50;

function TableSkeleton({ cols }: { cols: number }) {
  return (
    <>
      {Array.from({ length: 5 }, (_, rowIndex) => (
        <tr key={`sk-${rowIndex}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, colIndex) => (
            <td key={`sk-${rowIndex}-${colIndex}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function OpsBlacklistPage() {
  const [page, setPage] = useState(0);
  const [items, setItems] = useState<BlacklistEntryDTO[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [ip, setIp] = useState('');
  const [source, setSource] = useState('manual');
  const [ttl, setTtl] = useState('');
  const [preview, setPreview] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    const offset = page * PAGE_SIZE;
    const [res, err] = await to(
      api<BlacklistListResponse>(`/api/v1/ops/blacklist?limit=${PAGE_SIZE}&offset=${offset}`)
    );
    setLoading(false);
    if (err) {
      setError(err);
      return;
    }
    setError(null);
    const data = res?.data ?? {};
    setItems(data.items ?? []);
    setTotal(data.total ?? data.items?.length ?? 0);
  }, [page]);

  useEffect(() => {
    void load();
  }, [load]);

  const dryRun = async () => {
    setPreview(null);
    const [res, err] = await to(
      api('/api/v1/ops/blacklist', {
        method: 'POST',
        headers: { 'X-Dry-Run': '1' },
        body: JSON.stringify({
          ip: ip.trim(),
          source: source.trim() || 'manual',
          ttl_seconds: ttl ? Number(ttl) : undefined,
        }),
      })
    );
    if (err) {
      pushToastMessage({ title: 'Preview failed', message: mapServiceError(err).message });
      return;
    }
    setPreview(res?.data);
  };

  const block = async () => {
    setBusy(true);
    const [, err] = await to(
      apiConfirmed('/api/v1/ops/blacklist', {
        method: 'POST',
        body: JSON.stringify({
          ip: ip.trim(),
          source: source.trim() || 'manual',
          ttl_seconds: ttl ? Number(ttl) : undefined,
        }),
      })
    );
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Block failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'IP blocked', message: ip });
    setIp('');
    setPreview(null);
    void load();
  };

  const unblock = async (rowIp: string | undefined, rowSource: string | undefined) => {
    const [, err] = await to(
      apiConfirmed('/api/v1/ops/blacklist', {
        method: 'DELETE',
        body: JSON.stringify({ ip: rowIp, source: rowSource }),
      })
    );
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Unblock failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'IP unblocked', message: rowIp ?? '' });
    void load();
  };

  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load blacklist" />;

  const totalPages = Math.ceil(total / PAGE_SIZE) || 1;

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">Blacklist</h1>
        <p className="text-muted text-sm">Edge + manual IP blocks</p>
      </div>

      <div className="section-card stack mb-4">
        <h2 className="subsection-title">Add block</h2>
        <FormField label="IP address" htmlFor="bl-ip">
          <input
            id="bl-ip"
            className="form-input"
            placeholder="203.0.113.10"
            value={ip}
            onChange={(e) => setIp(e.target.value)}
          />
        </FormField>
        <FormField label="Source" htmlFor="bl-source">
          <input
            id="bl-source"
            className="form-input form-input--sm"
            value={source}
            onChange={(e) => setSource(e.target.value)}
          />
        </FormField>
        <FormField label="TTL seconds (optional)" htmlFor="bl-ttl">
          <input
            id="bl-ttl"
            className="form-input form-input--sm"
            inputMode="numeric"
            value={ttl}
            onChange={(e) => setTtl(e.target.value)}
          />
        </FormField>
        <div className="cluster--actions">
          <Button
            label="Dry-run preview"
            variant="secondary"
            size="sm"
            onClick={() => void dryRun()}
          />
          <Button
            label={busy ? 'Blocking...' : 'Block IP'}
            variant="danger"
            size="sm"
            loading={busy}
            disabled={busy || !ip.trim()}
            onClick={() => void block()}
          />
        </div>
        {preview ? (
          <pre className="code-block text-sm mt-2">{JSON.stringify(preview, null, 2)}</pre>
        ) : null}
      </div>

      <FilterToolbar
        pagination={
          totalPages > 1 ? (
            <PaginationBar
              label={`${page + 1} / ${totalPages}`}
              prevDisabled={page === 0}
              nextDisabled={page >= totalPages - 1}
              onPrev={() => setPage((p) => Math.max(0, p - 1))}
              onNext={() => setPage((p) => p + 1)}
            />
          ) : null
        }
      />

      <div className="table-wrapper table-wrapper--scroll elevation-raised">
        <table className="data-table" aria-label="Blacklist entries">
          <thead>
            <tr>
              <th scope="col">IP</th>
              <th scope="col">Reason</th>
              <th scope="col">Created</th>
              <th scope="col">Expires</th>
              <th scope="col" />
            </tr>
          </thead>
          <tbody>
            {loading ? <TableSkeleton cols={5} /> : null}
            {!loading && items.length === 0 ? (
              <tr>
                <td colSpan={5} className="text-center p-6">
                  <div className="empty-state">
                    <div className="empty-state__title">No blacklist entries</div>
                    <div className="empty-state__desc text-muted text-sm">
                      Blocked IPs appear here after you add one above.
                    </div>
                  </div>
                </td>
              </tr>
            ) : null}
            {items.map((row) => (
              <tr key={`${row.ip}-${row.reason}-${row.created_at}`}>
                <td className="font-mono">{row.ip ?? '-'}</td>
                <td>{row.reason ?? '-'}</td>
                <td>{row.created_at ? new Date(row.created_at).toLocaleString() : '-'}</td>
                <td>{row.expires_at ? new Date(row.expires_at).toLocaleString() : '-'}</td>
                <td>
                  <Button
                    label="Unblock"
                    variant="secondary"
                    size="sm"
                    onClick={() => void unblock(row.ip, row.reason ?? 'manual')}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
