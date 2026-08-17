import { useCallback, useEffect, useState } from 'react';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FilterToolbar } from '../components/filter_toolbar.js';
import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { displayLabel } from '../helpers/display_labels.js';
import {
  fetchDlqInboxPage,
  isDlqInboxEntryRetryable,
  retryDlqInboxEntry,
  type DLQInboxEntryDTO,
  type DLQInboxSource,
} from '../helpers/ops_dlq_inbox_api.js';

const SOURCE_OPTIONS: Array<{ value: DLQInboxSource; label: string }> = [
  { value: '', label: 'All sources' },
  { value: 'stream', label: 'Stream (Redis)' },
  { value: 'postback', label: 'Postback webhook' },
  { value: 'capi', label: 'CAPI' },
];

/**
 * CPA-M8 unified DLQ inbox — stream + postback/CAPI with source-aware retry.
 */
export function OpsDlqPage() {
  const user = auth.getUser();
  const canRetry = can(user?.permissions ?? [], 'shards:write');

  const [source, setSource] = useState<DLQInboxSource>('');
  const [items, setItems] = useState<DLQInboxEntryDTO[]>([]);
  const [cursor, setCursor] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [partialErrors, setPartialErrors] = useState<Array<{ source?: string; code?: string }>>([]);

  const load = useCallback(async (nextSource: DLQInboxSource, reset = true) => {
    setLoading(true);
    setError(null);
    const page = await fetchDlqInboxPage(nextSource);
    if (page.error) {
      setError(page.error);
      setItems([]);
      setCursor('');
      setPartialErrors([]);
    } else {
      setItems(page.items);
      setCursor(page.nextCursor);
      setPartialErrors(page.partialErrors);
    }
    setLoading(false);
    if (!reset) return;
  }, []);

  useEffect(() => {
    void load(source);
  }, [source, load]);

  const loadMore = async () => {
    if (!cursor || loading) return;
    setLoading(true);
    const page = await fetchDlqInboxPage(source, cursor);
    setLoading(false);
    if (page.error) {
      pushToastMessage({ title: 'Load failed', message: mapServiceError(page.error).message });
      return;
    }
    setItems((prev) => [...prev, ...page.items]);
    setCursor(page.nextCursor);
    if (page.partialErrors.length > 0) {
      setPartialErrors((prev) => [...prev, ...page.partialErrors]);
    }
  };

  const retry = async (row: DLQInboxEntryDTO) => {
    setLoading(true);
    try {
      await retryDlqInboxEntry(row);
      pushToastMessage({ title: 'Retry queued', message: `${row.source}:${row.id}` });
      await load(source);
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Retry failed', message: mapServiceError(e).message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <header className="page-header">
        <h1 className="h2">Unified DLQ inbox</h1>
        <p className="text-muted">
          Stream shard DLQ merged with postback and CAPI dead letters. Retry routes to the correct backend per source.
        </p>
      </header>

      {error ? <ErrorBlock error={error} /> : null}

      <section className="section-block" data-testid="ops-dlq-inbox">
        <FilterToolbar
          leading={(
            <div className="cluster cluster--sm items-center">
              <span className="text-muted text-sm">Source</span>
              <select
                className="form-input form-input--sm min-w-44"
                aria-label="DLQ source filter"
                data-testid="dlq-inbox-source"
                value={source}
                disabled={loading}
                onChange={(e) => setSource(e.target.value as DLQInboxSource)}
              >
                {SOURCE_OPTIONS.map((opt) => (
                  <option key={opt.value || 'all'} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          )}
          pagination={cursor ? (
            <Button
              label="Load more"
              variant="secondary"
              size="sm"
              loading={loading}
              disabled={loading}
              onClick={() => void loadMore()}
            />
          ) : undefined}
        />

        {partialErrors.length > 0 ? (
          <div className="stub-banner mb-4">
            {`Partial source errors: ${partialErrors.map((e) => `${e.source ?? '?'} (${e.code ?? 'err'})`).join('; ')}`}
          </div>
        ) : null}

        {loading && items.length === 0 ? (
          <p className="text-muted">Loading…</p>
        ) : null}

        <div className="table-wrapper elevation-raised">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Source</th>
                <th scope="col">ID</th>
                <th scope="col">Campaign</th>
                <th scope="col">Type</th>
                <th scope="col">Error</th>
                <th scope="col">Failed</th>
                <th scope="col">Status</th>
                {canRetry ? <th scope="col">Action</th> : null}
              </tr>
            </thead>
            <tbody>
              {items.map((row) => (
                <tr key={`${row.source}-${row.id}`}>
                  <td>{displayLabel(row.source)}</td>
                  <td className="font-mono text-sm">{row.id}</td>
                  <td className="font-mono text-sm">{row.campaign_id ?? '—'}</td>
                  <td>{row.event_type ?? '—'}</td>
                  <td>{row.error ?? '—'}</td>
                  <td className="text-muted text-sm">
                    {row.failed_at ? new Date(row.failed_at).toLocaleString() : '—'}
                  </td>
                  <td>{row.status ?? '—'}</td>
                  {canRetry ? (
                    <td>
                      {isDlqInboxEntryRetryable(row) ? (
                        <Button
                          label="Retry"
                          variant="ghost"
                          size="sm"
                          data-testid={`dlq-inbox-retry-${row.source}-${row.id}`}
                          disabled={loading}
                          onClick={() => void retry(row)}
                        />
                      ) : null}
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!loading && items.length === 0 ? (
          <p className="text-muted text-sm mt-3">No DLQ entries for this filter.</p>
        ) : null}
      </section>
    </>
  );
}
