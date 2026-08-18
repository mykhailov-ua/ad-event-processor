import { useCallback, useEffect, useState } from 'react';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FormField } from '../components/form_field.js';
import { fetchConsentProofs } from '../helpers/ops_compliance_api.js';
import type { ConsentProofDTO } from '../types/ops_compliance.js';

function purposeLabel(purposes: number): string {
  const bits: string[] = [];
  if ((purposes & 1) !== 0) bits.push('ad storage');
  if ((purposes & 2) !== 0) bits.push('analytics');
  return bits.length > 0 ? bits.join(', ') : 'none';
}

export function OpsConsentPage() {
  const [userFilter, setUserFilter] = useState('');
  const [activeUser, setActiveUser] = useState('');
  const [items, setItems] = useState<ConsentProofDTO[]>([]);
  const [cursor, setCursor] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const loadPage = useCallback(async (userId: string, pageCursor: string, append: boolean) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchConsentProofs(userId, pageCursor);
      setItems((prev) => (append ? [...prev, ...res.items] : res.items));
      setCursor(res.next_cursor ?? '');
    } catch (e) {
      setError(e);
      if (!append) setItems([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadPage('', '', false);
  }, [loadPage]);

  const search = () => {
    setActiveUser(userFilter.trim());
    void loadPage(userFilter.trim(), '', false);
  };

  return (
    <>
      <header className="page-header">
        <h1 className="h2">Consent proofs</h1>
        <p className="text-muted">
          Read-only browser for <code>POST /api/v1/consent</code> receipts. Records are immutable —
          no delete in UI.
        </p>
      </header>

      {error ? <ErrorBlock error={error} /> : null}

      <section className="card stack" data-testid="consent-browser">
        <FormField
          label="Search by user_id"
          htmlFor="consent-user-filter"
          hint="Server hashes user_id before lookup"
        >
          <div className="row gap-sm">
            <input
              id="consent-user-filter"
              className="form-input"
              data-testid="consent-user-filter"
              value={userFilter}
              disabled={loading}
              onChange={(e) => setUserFilter(e.target.value)}
            />
            <Button label="Search" variant="secondary" disabled={loading} onClick={search} />
          </div>
        </FormField>

        {loading && items.length === 0 ? (
          <p className="text-muted" data-testid="consent-loading">
            Loading proofs…
          </p>
        ) : null}

        <div className="table-wrapper elevation-raised">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">ID</th>
                <th scope="col">User hash</th>
                <th scope="col">Purposes</th>
                <th scope="col">Source</th>
                <th scope="col">Recorded</th>
                <th scope="col">State</th>
              </tr>
            </thead>
            <tbody>
              {items.map((row) => (
                <tr key={String(row.id)} data-testid={`consent-proof-row-${row.id}`}>
                  <td>{String(row.id)}</td>
                  <td className="font-mono text-sm">{row.user_id_hash}</td>
                  <td>{purposeLabel(row.purposes)}</td>
                  <td>{row.source}</td>
                  <td className="text-muted text-sm">
                    {row.recorded_at ? new Date(row.recorded_at).toLocaleString() : '—'}
                  </td>
                  <td className="text-sm text-muted">
                    {row.ad_storage ? 'ad' : ''}
                    {row.ad_storage && row.analytics_storage ? ' · ' : ''}
                    {row.analytics_storage ? 'analytics' : ''}
                    {!row.ad_storage && !row.analytics_storage ? '—' : ''}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {cursor ? (
          <Button
            label="Load more"
            variant="secondary"
            size="sm"
            disabled={loading}
            onClick={() => void loadPage(activeUser, cursor, true)}
          />
        ) : null}

        {!loading && items.length === 0 ? (
          <p className="text-muted text-sm">No consent proofs found.</p>
        ) : null}
      </section>
    </>
  );
}
