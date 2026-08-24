import { useCallback, useEffect, useState } from 'react';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  type DomainHealthRow,
  addCustomDomain,
  deleteCustomDomain,
  fetchDomains,
  healthStatusLabel,
  parkDomain,
  probeDomain,
  setupDomainSSL,
  sslStatusLabel,
} from '../helpers/domains_api.js';
import { checkTlsAllowed } from '../helpers/ops_compliance_api.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

function healthBadgeClass(status: string): string {
  switch (status) {
    case 'healthy':
      return 'ACTIVE';
    case 'degraded':
      return 'PAUSED';
    case 'down':
      return 'FAILED';
    default:
      return 'UNKNOWN';
  }
}

function TableSkeleton({ cols, rows = 3 }: { cols: number; rows?: number }) {
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

export function SettingsDomainsPage() {
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'settings:write');

  const [rows, setRows] = useState<DomainHealthRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [customHost, setCustomHost] = useState('');
  const [tlsHost, setTlsHost] = useState('');
  const [tlsAskToken, setTlsAskToken] = useState('');
  const [tlsResult, setTlsResult] = useState<'allowed' | 'denied' | null>(null);
  const [parkDomainInput, setParkDomainInput] = useState('');
  const [parkZoneId, setParkZoneId] = useState('');
  const [parkPoolId, setParkPoolId] = useState('');

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setRows(await fetchDomains());
    } catch (e) {
      setError(e instanceof Error ? e : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const addCustom = async () => {
    if (!canWrite || !customHost.trim()) return;
    setBusy(true);
    try {
      await addCustomDomain(customHost.trim());
      setCustomHost('');
      pushToastMessage({ title: 'Domain added', message: 'Custom domain registered for probing' });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Add failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  const removeHost = async (hostname: string) => {
    if (!canWrite) return;
    setBusy(true);
    try {
      await deleteCustomDomain(hostname);
      pushToastMessage({ title: 'Domain removed', message: hostname });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Delete failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  const probeNow = async (hostname: string) => {
    if (!canWrite) return;
    setBusy(true);
    try {
      await probeDomain(hostname);
      pushToastMessage({ title: 'Probe complete', message: hostname });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Probe failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  const setupSSL = async (hostname: string) => {
    if (!canWrite) return;
    setBusy(true);
    try {
      const result = await setupDomainSSL(hostname);
      pushToastMessage({
        title: result.status === 'ok' ? 'SSL setup started' : 'SSL setup failed',
        message: result.message,
      });
      await reload();
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'SSL setup failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  const checkTls = async () => {
    if (!tlsHost.trim()) return;
    setBusy(true);
    setTlsResult(null);
    try {
      const allowed = await checkTlsAllowed(tlsHost.trim(), tlsAskToken);
      setTlsResult(allowed ? 'allowed' : 'denied');
      pushToastMessage({
        title: allowed ? 'TLS allowed' : 'TLS denied',
        message: tlsHost.trim(),
      });
    } catch (e) {
      pushToastMessage({ title: 'TLS check failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  const parkHost = async () => {
    if (!canWrite || !parkDomainInput.trim() || !parkZoneId.trim()) return;
    setBusy(true);
    try {
      const req = {
        domain: parkDomainInput.trim(),
        cloudflare_zone_id: parkZoneId.trim(),
        ...(parkPoolId.trim() ? { pool_id: parkPoolId.trim() } : {}),
      };
      const result = await parkDomain(req);
      pushToastMessage({
        title: result.success ? 'Domain parked' : 'Park failed',
        message: result.dns_record_id || result.ssl_status || parkDomainInput.trim(),
      });
      if (result.success) {
        setParkDomainInput('');
        setParkZoneId('');
        setParkPoolId('');
        await reload();
      }
    } catch (e) {
      if (e instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Park failed', message: mapServiceError(e).message });
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <header className="page-header">
        <h1 className="h2">Domains</h1>
        <p className="text-muted">
          Health probes every 5 minutes (HTTP + TLS). Tracking and admin hosts sync from platform
          config.
        </p>
      </header>

      {error ? <ErrorBlock error={error} /> : null}

      {canWrite ? (
        <section className="card stack" data-testid="domains-add-custom">
          <h2 className="h3">Add custom domain</h2>
          <div className="row gap-sm">
            <input
              type="text"
              className="form-input"
              placeholder="lander.example.com"
              value={customHost}
              disabled={busy}
              onChange={(e) => setCustomHost(e.target.value)}
            />
            <Button
              label="Add"
              variant="primary"
              disabled={busy || !customHost.trim()}
              onClick={() => void addCustom()}
            />
          </div>
        </section>
      ) : null}

      <section className="card stack" data-testid="domains-tls-check">
        <h2 className="h3">TLS on-demand allowlist</h2>
        <p className="text-muted text-sm">
          Probe Caddy ask endpoint (<code>GET /api/v1/ops/domains/.../tls-allowed</code>). Pass ask
          token when local bypass is disabled.
        </p>
        <div className="row gap-sm">
          <input
            type="text"
            className="form-input"
            placeholder="buyer.example.com"
            data-testid="tls-check-host"
            value={tlsHost}
            disabled={busy}
            onChange={(e) => setTlsHost(e.target.value)}
          />
          <input
            type="password"
            className="form-input"
            placeholder="Caddy ask token (optional)"
            data-testid="tls-check-token"
            value={tlsAskToken}
            disabled={busy}
            onChange={(e) => setTlsAskToken(e.target.value)}
          />
          <Button
            label="Check TLS"
            variant="ghost"
            disabled={busy || !tlsHost.trim()}
            data-testid="tls-check-submit"
            onClick={() => void checkTls()}
          />
        </div>
        {tlsResult ? (
          <p className="text-sm" data-testid="tls-check-result">
            Result: <strong>{tlsResult === 'allowed' ? 'allowed' : 'denied'}</strong>
          </p>
        ) : null}
      </section>

      {canWrite ? (
        <section className="card stack" data-testid="domains-park">
          <h2 className="h3">Park domain (Cloudflare)</h2>
          <p className="text-muted text-sm">
            Create DNS for a buyer subdomain via <code>POST /api/v1/domains/park</code>.
          </p>
          <input
            type="text"
            className="form-input"
            placeholder="track.buyer.example.com"
            data-testid="park-domain"
            value={parkDomainInput}
            disabled={busy}
            onChange={(e) => setParkDomainInput(e.target.value)}
          />
          <input
            type="text"
            className="form-input"
            placeholder="Cloudflare zone ID"
            data-testid="park-zone-id"
            value={parkZoneId}
            disabled={busy}
            onChange={(e) => setParkZoneId(e.target.value)}
          />
          <input
            type="text"
            className="form-input"
            placeholder="Pool ID (optional)"
            data-testid="park-pool-id"
            value={parkPoolId}
            disabled={busy}
            onChange={(e) => setParkPoolId(e.target.value)}
          />
          <Button
            label="Park domain"
            variant="primary"
            disabled={busy || !parkDomainInput.trim() || !parkZoneId.trim()}
            data-testid="park-submit"
            onClick={() => void parkHost()}
          />
        </section>
      ) : null}

      <section className={`card stack${loading ? '' : ''}`} data-testid="domains-table">
        <h2 className="h3">Monitored domains</h2>
        {loading ? (
          <table className="data-table">
            <tbody>
              <TableSkeleton cols={7} />
            </tbody>
          </table>
        ) : rows.length === 0 ? (
          <p className="text-muted">
            No domains configured yet. Set tracking domain in Platform settings.
          </p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Hostname</th>
                <th>Role</th>
                <th>Health</th>
                <th>SSL</th>
                <th>Latency</th>
                <th>Last probe</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.hostname} data-testid={`domain-row-${row.hostname}`}>
                  <td>{row.hostname}</td>
                  <td>{row.role}</td>
                  <td>
                    <StatusBadge
                      status={healthBadgeClass(row.health_status)}
                      label={healthStatusLabel(row.health_status)}
                    />
                  </td>
                  <td>
                    <StatusBadge
                      status={row.ssl_status.toUpperCase()}
                      label={sslStatusLabel(row.ssl_status)}
                      kind="service"
                    />
                    {row.ssl_not_after ? (
                      <div className="text-muted text-sm">
                        until {new Date(row.ssl_not_after).toLocaleDateString()}
                      </div>
                    ) : null}
                  </td>
                  <td>{row.probe_latency_ms != null ? `${row.probe_latency_ms} ms` : '-'}</td>
                  <td>{row.last_probe_at ? new Date(row.last_probe_at).toLocaleString() : '-'}</td>
                  <td className="row gap-xs">
                    {canWrite ? (
                      <Button
                        label="Probe"
                        variant="ghost"
                        size="sm"
                        disabled={busy}
                        data-testid="domain-probe"
                        onClick={() => void probeNow(row.hostname)}
                      />
                    ) : null}
                    {canWrite ? (
                      <Button
                        label="Setup SSL"
                        variant="ghost"
                        size="sm"
                        disabled={busy}
                        data-testid="domain-ssl-setup"
                        onClick={() => void setupSSL(row.hostname)}
                      />
                    ) : null}
                    {canWrite && row.role === 'custom' ? (
                      <Button
                        label="Remove"
                        variant="ghost"
                        size="sm"
                        disabled={busy}
                        onClick={() => void removeHost(row.hostname)}
                      />
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </>
  );
}
