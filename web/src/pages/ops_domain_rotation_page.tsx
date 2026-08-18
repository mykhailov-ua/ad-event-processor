import { useCallback, useEffect, useState } from 'react';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';
import { api } from '../helpers/api_client.js';
import { displayLabel } from '../helpers/display_labels.js';

type DomainRotationHost = {
  hostname: string;
  role: string;
  health_status: string;
  ssl_status?: string;
  pool_id?: string;
  pool_domain_status?: string;
  dmr_campaign_count: number;
  active_campaign_count: number;
};

type DomainRotationResponse = {
  hosts?: DomainRotationHost[];
};

export function OpsDomainRotationPage() {
  const [hosts, setHosts] = useState<DomainRotationHost[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api<DomainRotationResponse>('/api/v1/ops/domains/rotation');
      setHosts(res.data?.hosts ?? []);
    } catch (e) {
      setError(e);
      setHosts([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <>
      <header className="page-header">
        <h1 className="h2">Domain rotation</h1>
        <p className="text-muted">
          Tracking host health with pool membership and campaigns using DMR referer hiding.
        </p>
      </header>

      {error ? <ErrorBlock error={error} /> : null}

      <section className="section-block" data-testid="ops-domain-rotation">
        {loading ? <p className="text-muted">Loading…</p> : null}
        <div className="table-wrapper elevation-raised">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Hostname</th>
                <th scope="col">Role</th>
                <th scope="col">Health</th>
                <th scope="col">SSL</th>
                <th scope="col">Pool status</th>
                <th scope="col">DMR campaigns</th>
                <th scope="col">Active campaigns</th>
              </tr>
            </thead>
            <tbody>
              {hosts.map((row) => (
                <tr key={row.hostname}>
                  <td className="font-mono">{row.hostname}</td>
                  <td>{displayLabel(row.role)}</td>
                  <td>
                    <StatusBadge
                      status={row.health_status}
                      kind="service"
                      label={displayLabel(row.health_status)}
                    />
                  </td>
                  <td>{row.ssl_status ? displayLabel(row.ssl_status) : '—'}</td>
                  <td>{row.pool_domain_status ? displayLabel(row.pool_domain_status) : '—'}</td>
                  <td>{String(row.dmr_campaign_count)}</td>
                  <td>{String(row.active_campaign_count)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!loading && hosts.length === 0 ? (
          <p className="text-muted text-sm mt-3">No tracking domains registered.</p>
        ) : null}
      </section>
    </>
  );
}
