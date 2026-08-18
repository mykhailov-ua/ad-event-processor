import { useCallback, useEffect, useState } from 'react';
import type { PublisherDashboard, PublisherStatement } from '../types/publisher.js';
import { fetchPublisherDashboard, fetchPublisherStatements } from '../helpers/publisher_api.js';
import { fetchSupplyValidation } from '../helpers/supply_api.js';
import type { SupplyValidation } from '../types/publisher.js';
import { formatAmountMicro } from '../helpers/money.js';
import { to } from '../lib/to.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

type PublisherTab = 'dashboard' | 'statements' | 'supply';

function pct(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`;
}

export function PublisherPage() {
  const [tab, setTab] = useState<PublisherTab>('dashboard');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [dashboard, setDashboard] = useState<PublisherDashboard | null>(null);
  const [statements, setStatements] = useState<PublisherStatement[]>([]);
  const [validation, setValidation] = useState<SupplyValidation | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [dashRes, stmtRes, valRes] = await Promise.all([
      to(fetchPublisherDashboard()),
      to(fetchPublisherStatements()),
      to(fetchSupplyValidation()),
    ]);
    if (dashRes[1]) {
      setError(dashRes[1]);
      setLoading(false);
      return;
    }
    setDashboard(dashRes[0] ?? null);
    setStatements(stmtRes[1] ? [] : (stmtRes[0]?.items ?? []));
    setValidation(valRes[1] ? null : (valRes[0] ?? null));
    setLoading(false);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Publisher portal unavailable" />;
  }

  return (
    <section className="stack" data-testid="publisher-portal">
      <div className="page-header">
        <h1 className="page-header__title">Publisher dashboard</h1>
        <p className="page-header__desc">
          Seller scope: <code className="code-inline">{dashboard?.seller_id || '—'}</code>
          {dashboard?.publisher_account_id ? (
            <>
              {' '}
              · account <code className="code-inline">{dashboard.publisher_account_id}</code>
            </>
          ) : null}
        </p>
      </div>

      <div className="filter-row cluster--actions">
        {(['dashboard', 'statements', 'supply'] as PublisherTab[]).map((key) => (
          <button
            key={key}
            type="button"
            className={`btn btn--sm ${tab === key ? 'btn--primary' : 'btn--secondary'}`}
            onClick={() => setTab(key)}
          >
            {key === 'dashboard'
              ? 'Performance'
              : key === 'statements'
                ? 'Statements'
                : 'Supply validation'}
          </button>
        ))}
      </div>

      {tab === 'dashboard' ? (
        <div className="stack">
          <div className="metric-grid">
            <div className="metric-card">
              <span className="metric-card__label">Impressions</span>
              <span className="metric-card__value">
                {loading ? '…' : String(dashboard?.kpis.impressions ?? 0)}
              </span>
            </div>
            <div className="metric-card">
              <span className="metric-card__label">Fill rate</span>
              <span className="metric-card__value">
                {loading ? '…' : pct(dashboard?.kpis.fill_rate ?? 0)}
              </span>
            </div>
            <div className="metric-card">
              <span className="metric-card__label">eCPM</span>
              <span className="metric-card__value">
                {loading ? '…' : `$${formatAmountMicro(dashboard?.kpis.ecpm_micro ?? 0)}`}
              </span>
            </div>
            <div className="metric-card">
              <span className="metric-card__label">IVT rate</span>
              <span className="metric-card__value">
                {loading ? '…' : pct(dashboard?.kpis.ivt_rate ?? 0)}
              </span>
            </div>
          </div>

          <div className="section-card">
            <h2 className="subsection-title">Placements</h2>
            <div className="table-wrapper">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Placement</th>
                    <th>Impressions</th>
                    <th>Clicks</th>
                    <th>Fill</th>
                    <th>Revenue</th>
                    <th>eCPM</th>
                  </tr>
                </thead>
                <tbody>
                  {loading ? (
                    <tr>
                      <td colSpan={6}>Loading…</td>
                    </tr>
                  ) : (
                    (dashboard?.placements ?? []).map((row) => (
                      <tr key={row.placement_id}>
                        <td className="font-mono">{row.placement_id}</td>
                        <td>{row.impressions}</td>
                        <td>{row.clicks}</td>
                        <td>{pct(row.fill_rate)}</td>
                        <td>${formatAmountMicro(row.revenue_micro)}</td>
                        <td>${formatAmountMicro(row.ecpm_micro)}</td>
                      </tr>
                    ))
                  )}
                  {!loading && (dashboard?.placements?.length ?? 0) === 0 ? (
                    <tr>
                      <td colSpan={6} className="text-muted">
                        No placement traffic in range.
                      </td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      ) : null}

      {tab === 'statements' ? (
        <div className="section-card">
          <div className="cluster cluster--actions" style={{ justifyContent: 'space-between' }}>
            <h2 className="subsection-title">Payout statements</h2>
            <a
              className="btn btn--secondary btn--sm"
              href="/api/v1/publisher/statements?format=csv"
            >
              Export CSV
            </a>
          </div>
          <div className="table-wrapper">
            <table className="data-table" data-testid="publisher-statements-table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Amount</th>
                  <th>Campaign</th>
                  <th>Reference</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr>
                    <td colSpan={4}>Loading…</td>
                  </tr>
                ) : (
                  statements.map((row) => (
                    <tr key={row.id}>
                      <td>{row.created_at}</td>
                      <td>${formatAmountMicro(row.amount_micro)}</td>
                      <td className="font-mono">{row.campaign_id || '—'}</td>
                      <td className="font-mono text-sm">{row.idempotency_hash || '—'}</td>
                    </tr>
                  ))
                )}
                {!loading && statements.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="text-muted">
                      No publisher payouts in range.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      {tab === 'supply' ? (
        <div className="section-card stack" data-testid="publisher-supply-validation">
          <h2 className="subsection-title">ads.txt / sellers.json validation</h2>
          {loading ? <p className="text-muted">Loading…</p> : null}
          {!loading && validation ? (
            <dl className="definition-list">
              <dt>sellers.json</dt>
              <dd>
                <StatusBadge
                  status={validation.sellers_json_valid ? 'ok' : 'error'}
                  label={validation.sellers_json_valid ? 'valid' : 'invalid'}
                />{' '}
                {validation.sellers_count} sellers · SHA-256{' '}
                <code className="code-inline">
                  {validation.sellers_checksum_sha256.slice(0, 16)}…
                </code>
              </dd>
              <dt>ads.txt</dt>
              <dd>
                <StatusBadge
                  status={validation.ads_txt_valid ? 'ok' : 'error'}
                  label={validation.ads_txt_valid ? 'valid' : 'invalid'}
                />{' '}
                {validation.ads_txt_line_count} lines · SHA-256{' '}
                <code className="code-inline">
                  {validation.ads_txt_checksum_sha256.slice(0, 16)}…
                </code>
              </dd>
              {(validation.issues ?? []).map((issue) => (
                <dd key={issue} className="text-muted">
                  {issue}
                </dd>
              ))}
            </dl>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
