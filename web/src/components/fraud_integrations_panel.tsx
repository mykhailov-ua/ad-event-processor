import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  fetchFraudIntegrations,
  fraudIntegrationBadgeStatus,
  fraudIntegrationStatusLabel,
  type FraudIntegrationRow,
} from '../helpers/fraud_integrations_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { StatusBadge } from './status_badge.js';
import { ButtonLink } from './button.js';

export type FraudIntegrationsPanelProps = {
  customerId: string;
};

/**
 * Format ISO timestamp for integration health table cells.
 */
function formatTs(iso?: string): string {
  if (!iso) return '—';
  const parsed = new Date(iso);
  return Number.isNaN(parsed.getTime()) ? iso : parsed.toLocaleString();
}

/**
 * Read-only postback/CAPI health table for the fraud dashboard.
 */
export function FraudIntegrationsPanel({ customerId }: FraudIntegrationsPanelProps) {
  const [rows, setRows] = useState<FraudIntegrationRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!customerId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await fetchFraudIntegrations(customerId);
      setRows(data);
    } catch (err) {
      setError(mapServiceError(err).message);
    } finally {
      setLoading(false);
    }
  }, [customerId]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="stack stack--lg" data-testid="fraud-integrations-panel">
      <p className="text-muted text-sm">
        Postback and CAPI delivery health per campaign. Configure providers on each campaign&apos;s{' '}
        <Link to="/campaigns">CAPI &amp; Postbacks</Link> tab.
      </p>
      {error ? <p className="text-danger text-sm">{error}</p> : null}
      <div className="table-wrapper">
        <table className="data-table">
          <thead>
            <tr>
              <th scope="col">Campaign</th>
              <th scope="col">Provider</th>
              <th scope="col">Status</th>
              <th scope="col">Last success</th>
              <th scope="col">DLQ</th>
              <th scope="col">Last error</th>
              <th scope="col" />
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={7} className="text-muted">
                  Loading…
                </td>
              </tr>
            ) : null}
            {!loading && rows.length === 0 ? (
              <tr>
                <td colSpan={7} className="text-muted">
                  No campaigns for this customer.
                </td>
              </tr>
            ) : null}
            {!loading
              ? rows.map((row) => (
                  <tr key={row.campaign_id}>
                    <td>{row.name}</td>
                    <td className="font-mono text-sm">{row.provider || '—'}</td>
                    <td>
                      <StatusBadge
                        status={fraudIntegrationBadgeStatus(row.health_status)}
                        label={fraudIntegrationStatusLabel(row.health_status)}
                      />
                    </td>
                    <td>{formatTs(row.last_success_at)}</td>
                    <td className="font-mono">{String(row.dlq_count ?? 0)}</td>
                    <td className="text-sm text-muted">{row.last_error || '—'}</td>
                    <td>
                      <ButtonLink
                        href={`/campaigns/${encodeURIComponent(row.campaign_id)}?tab=postbacks`}
                        label="Fix"
                        variant="ghost"
                        size="sm"
                      />
                    </td>
                  </tr>
                ))
              : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
