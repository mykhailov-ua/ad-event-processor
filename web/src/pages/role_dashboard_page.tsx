import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { useParams } from 'react-router-dom';
import type { BuyerCampaignPortfolioRow } from '../types/api/campaign.js';
import { CommercialMetrics } from '../components/commercial_metrics.js';
import type { MetricsBlockDTO } from '../types/metrics.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { fraudTierBandRows } from '../helpers/edge_fraud_tier.js';
import { formatAmountMicro } from '../helpers/money.js';
import { t } from '../helpers/i18n.js';
import { validateCustomerIdField } from '../helpers/validators.js';
import { pollReportJob, downloadReportExport } from '../helpers/report_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { Button } from '../components/button.js';
import { ButtonLink } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

type SourceRow = {
  campaign_id?: string;
  sub1?: string;
  sub2?: string;
  ivt_rate?: number;
  quality_score?: number;
  spend_micro?: number;
};

type ExportJobRow = {
  id: string;
  status?: string;
  format?: string;
};

type AccountantClose = {
  billing_month?: string;
  invariant_ok?: boolean;
};

type FraudTierThresholds = {
  pass_max?: number;
  suspect_max?: number;
  ivt_max?: number;
};

type FraudGeoHint = {
  country?: string;
  ivt_rate?: number;
  ivt_events?: number;
};

type FraudLabelRow = {
  ip_hash?: string;
  label?: number;
  reason?: string;
  created_at?: string;
};

type RoleDashboardData = {
  customer_id?: string;
  kpis?: MetricsBlockDTO | null;
  campaigns?: BuyerCampaignPortfolioRow[];
  worst_sources?: SourceRow[];
  billed_micro?: number;
  ar_aging_micro?: number;
  dispute_exposure_micro?: number;
  close?: AccountantClose;
  tax_country?: string;
  tax_scheme?: string;
  tax_vat_id?: string;
  export_jobs?: ExportJobRow[];
  ghost_ivt_campaigns?: number;
  labels_pending?: number;
  edge_blocked_fraud?: number;
  ml_active_version_id?: string;
  ml_artifact_hash?: string;
  ml_precision?: number;
  ml_recall?: number;
  ml_drift_detected?: boolean;
  fraud_tier_thresholds?: FraudTierThresholds;
  geo_hints?: FraudGeoHint[];
  recent_labels?: FraudLabelRow[];
};

const TITLES: Record<string, string> = {
  adops: 'AdOps dashboard',
  cfo: 'CFO dashboard',
  accountant: 'Accountant dashboard',
  fraud: 'Fraud dashboard',
};

const ENDPOINTS: Record<string, string> = {
  adops: '/api/v1/dashboards/adops',
  cfo: '/api/v1/dashboards/cfo',
  accountant: '/api/v1/dashboards/accountant',
  fraud: '/api/v1/dashboards/fraud',
};

function EmptyBlock({ title, description }: { title: string; description: string }) {
  return (
    <div className="empty-state">
      <div className="empty-state__title">{title}</div>
      <div className="empty-state__desc text-muted text-sm">{description}</div>
    </div>
  );
}

function Subsection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="stack">
      <h2 className="subsection-title">{title}</h2>
      {children}
    </section>
  );
}

type ExportJobState = Record<string, { status: string; error?: string }>;

function syncExportJobState(
  prev: ExportJobState,
  jobs: ExportJobRow[],
): ExportJobState {
  const next = { ...prev };
  for (let i = 0; i < jobs.length; i++) {
    const job = jobs[i];
    if (!job?.id) continue;
    next[job.id] = {
      status: job.status ?? next[job.id]?.status ?? 'PENDING',
      error: next[job.id]?.error,
    };
  }
  return next;
}

function hasPendingExportJobs(state: ExportJobState): boolean {
  const ids = Object.keys(state);
  for (let i = 0; i < ids.length; i++) {
    const status = state[ids[i]]?.status ?? '';
    if (status === 'PENDING' || status === 'RUNNING') return true;
  }
  return false;
}

function AdOpsBody({ data }: { data: RoleDashboardData }) {
  const campaigns = Array.isArray(data.campaigns) ? data.campaigns : [];
  const worst = Array.isArray(data.worst_sources) ? data.worst_sources : [];

  return (
    <div className="stack stack--lg section-block">
      <CommercialMetrics kpis={data.kpis} masked={false} />
      <Subsection title="Campaigns">
        {campaigns.length === 0 ? (
          <EmptyBlock
            title="No campaigns"
            description="Campaign utilization data will appear when campaigns are active."
          />
        ) : (
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Campaign</th>
                  <th scope="col">Util %</th>
                  <th scope="col">Drift %</th>
                  <th scope="col">Spend</th>
                </tr>
              </thead>
              <tbody>
                {campaigns.map((c) => (
                  <tr key={c.id}>
                    <td><a href={`/campaigns/${c.id}`}>{c.name ?? c.id}</a></td>
                    <td>{c.utilization_pct?.toFixed?.(1) ?? '—'}</td>
                    <td>{c.pacing_drift_pct?.toFixed?.(1) ?? '—'}</td>
                    <td className="font-mono">{formatAmountMicro(c.spend_micro ?? 0)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Subsection>
      <Subsection title="Worst sources">
        {worst.length === 0 ? (
          <EmptyBlock
            title="No quality flags"
            description="No placement issues from the last reporting window."
          />
        ) : (
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Source</th>
                  <th scope="col">Campaign</th>
                  <th scope="col">IVT %</th>
                  <th scope="col">Quality</th>
                  <th scope="col">Spend</th>
                  <th scope="col" />
                </tr>
              </thead>
              <tbody>
                {worst.map((s, idx) => {
                  const placementsHref = data.customer_id
                    ? `/reports/placements?customer_id=${encodeURIComponent(data.customer_id)}&campaign_id=${encodeURIComponent(s.campaign_id ?? '')}`
                    : '/reports/placements';
                  return (
                    <tr key={`${s.campaign_id ?? ''}-${s.sub1 ?? ''}-${idx}`}>
                      <td>{s.sub1 ?? s.sub2 ?? '—'}</td>
                      <td>
                        {s.campaign_id
                          ? <a href={`/campaigns/${s.campaign_id}`}>{s.campaign_id.slice(0, 8)}</a>
                          : '—'}
                      </td>
                      <td>{`${((s.ivt_rate ?? 0) * 100).toFixed(1)}%`}</td>
                      <td>{s.quality_score != null ? s.quality_score.toFixed(1) : '—'}</td>
                      <td className="font-mono">{formatAmountMicro(s.spend_micro ?? 0)}</td>
                      <td>
                        <ButtonLink href={placementsHref} label="Placements" variant="ghost" size="sm" />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Subsection>
    </div>
  );
}

function CFOBody({ data }: { data: RoleDashboardData }) {
  return (
    <dl className="definition-list section-block">
      <dt>Billed</dt>
      <dd className="font-mono">{formatAmountMicro(data.billed_micro ?? 0)}</dd>
      <dt>AR aging</dt>
      <dd className="font-mono">{formatAmountMicro(data.ar_aging_micro ?? 0)}</dd>
      <dt>Dispute exposure</dt>
      <dd className="font-mono">{formatAmountMicro(data.dispute_exposure_micro ?? 0)}</dd>
    </dl>
  );
}

function AccountantBody({
  data,
  exportJobState,
  onDownload,
}: {
  data: RoleDashboardData;
  exportJobState: ExportJobState;
  onDownload: (jobId: string) => void;
}) {
  const close = data.close ?? {};
  const jobs = Array.isArray(data.export_jobs) ? data.export_jobs : [];
  const customerId = data.customer_id ?? '';

  return (
    <div className="stack section-block">
      <dl className="definition-list">
        <dt>Billing month</dt>
        <dd>{close.billing_month ?? '—'}</dd>
        <dt>Invariant OK</dt>
        <dd>{close.invariant_ok ? 'Yes' : 'No'}</dd>
        <dt>Tax country</dt>
        <dd>{data.tax_country ?? '—'}</dd>
        <dt>Tax scheme</dt>
        <dd>{data.tax_scheme ?? data.tax_vat_id ?? '—'}</dd>
      </dl>
      {customerId ? (
        <p className="text-sm">
          <ButtonLink
            label="Open ledger exports"
            href={`/billing?customer_id=${encodeURIComponent(customerId)}&tab=exports`}
            variant="secondary"
            size="sm"
          />
        </p>
      ) : null}
      <Subsection title="Export jobs">
        {jobs.length === 0 ? (
          <EmptyBlock
            title="No export jobs"
            description="Scheduled or manual exports will appear here."
          />
        ) : (
          <div className="table-wrapper">
            <table className="data-table" data-testid="accountant-export-jobs">
              <thead>
                <tr>
                  <th scope="col">Job</th>
                  <th scope="col">Format</th>
                  <th scope="col">Status</th>
                  <th scope="col" />
                </tr>
              </thead>
              <tbody>
                {jobs.map((j) => {
                  const live = exportJobState[j.id] ?? { status: j.status };
                  const status = live.status ?? j.status ?? 'PENDING';
                  const badgeStatus = status === 'COMPLETED' ? 'ok'
                    : status === 'FAILED' ? 'failed'
                      : 'pending';
                  const badgeKind = status === 'COMPLETED' || status === 'FAILED' || status === 'RUNNING'
                    ? 'service'
                    : 'invoice';
                  return (
                    <tr key={j.id} data-testid={`export-job-${j.id}`}>
                      <td className="font-mono text-sm">{j.id.slice(0, 8)}</td>
                      <td>{j.format ?? 'csv'}</td>
                      <td>
                        <StatusBadge status={badgeStatus} kind={badgeKind} label={status} />
                      </td>
                      <td>
                        {status === 'COMPLETED' ? (
                          <Button
                            label="Download"
                            variant="secondary"
                            size="sm"
                            onClick={() => onDownload(j.id)}
                          />
                        ) : (status === 'PENDING' || status === 'RUNNING') ? (
                          <span className="text-muted text-sm">Polling…</span>
                        ) : live.error ? (
                          <span className="text-danger text-sm">{live.error}</span>
                        ) : null}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Subsection>
    </div>
  );
}

function FraudBody({ data }: { data: RoleDashboardData }) {
  const thresholds = data.fraud_tier_thresholds ?? {};
  const geoHints = Array.isArray(data.geo_hints) ? data.geo_hints : [];
  const recentLabels = Array.isArray(data.recent_labels) ? data.recent_labels : [];
  const customerQS = data.customer_id
    ? `?customer_id=${encodeURIComponent(data.customer_id)}`
    : '';

  return (
    <div className="stack stack--lg section-block" data-testid="fraud-dashboard">
      <dl className="definition-list">
        <dt>Ghost IVT campaigns</dt>
        <dd>{String(data.ghost_ivt_campaigns ?? 0)}</dd>
        <dt>Labels queue (7d)</dt>
        <dd>{String(data.labels_pending ?? 0)}</dd>
        <dt>Edge blocked (fraud tier)</dt>
        <dd className="font-mono">{String(data.edge_blocked_fraud ?? 0)}</dd>
        <dt>ML active version</dt>
        <dd className="font-mono">{data.ml_active_version_id ?? '—'}</dd>
        <dt>ML artifact hash</dt>
        <dd className="font-mono text-sm">
          {data.ml_artifact_hash ? `${data.ml_artifact_hash.slice(0, 12)}…` : '—'}
        </dd>
        <dt>Shadow eval precision</dt>
        <dd>
          {data.ml_precision != null && data.ml_precision > 0
            ? `${(data.ml_precision * 100).toFixed(1)}%`
            : '—'}
        </dd>
        <dt>Shadow eval recall</dt>
        <dd>
          {data.ml_recall != null && data.ml_recall > 0
            ? `${(data.ml_recall * 100).toFixed(1)}%`
            : '—'}
        </dd>
        <dt>Model drift</dt>
        <dd>
          {data.ml_drift_detected
            ? <StatusBadge status="warning" label="detected" />
            : 'No'}
        </dd>
      </dl>
      <Subsection title="Edge fraud tier thresholds">
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Tier</th>
                <th scope="col">Score</th>
              </tr>
            </thead>
            <tbody>
              {fraudTierBandRows().map((row) => (
                <tr key={row.tier}>
                  <td>{row.tier}</td>
                  <td className="font-mono">{row.range}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {thresholds.pass_max ? (
            <p className="text-muted text-xs">
              {`API: pass≤${thresholds.pass_max}, suspect≤${thresholds.suspect_max}, ivt≤${thresholds.ivt_max}`}
            </p>
          ) : null}
        </div>
      </Subsection>
      <Subsection title="Geo suggestions (read-only)">
        {geoHints.length === 0 ? (
          <EmptyBlock
            title="No high-IVT countries"
            description="No geo suggestions in the last 7 days."
          />
        ) : (
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Country</th>
                  <th scope="col">IVT %</th>
                  <th scope="col">Events</th>
                  <th scope="col" />
                </tr>
              </thead>
              <tbody>
                {geoHints.map((hint) => (
                  <tr key={hint.country ?? ''}>
                    <td>{hint.country ?? '—'}</td>
                    <td>{`${((hint.ivt_rate ?? 0) * 100).toFixed(1)}%`}</td>
                    <td className="font-mono">{String(hint.ivt_events ?? 0)}</td>
                    <td>
                      <ButtonLink
                        href={`/reports/ivt-by-source${customerQS}`}
                        label="IVT report"
                        variant="ghost"
                        size="sm"
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <p className="text-muted text-sm">
          Review suspicious IPs via <a href="/ops/blacklist">ops blacklist</a>
          {' '}(dry-run: POST with <code>dry_run=1</code>).
        </p>
      </Subsection>
      <Subsection title="Manual labels queue">
        {recentLabels.length === 0 ? (
          <EmptyBlock
            title="No manual labels in queue"
            description="Fraud label review queue is empty."
          />
        ) : (
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">IP hash</th>
                  <th scope="col">Label</th>
                  <th scope="col">Reason</th>
                  <th scope="col">Created</th>
                </tr>
              </thead>
              <tbody>
                {recentLabels.map((row, idx) => (
                  <tr key={`${row.ip_hash ?? ''}-${idx}`}>
                    <td className="font-mono text-sm">{`${row.ip_hash?.slice(0, 8) ?? ''}…`}</td>
                    <td>{row.label === 1 ? 'fraud' : 'legit'}</td>
                    <td>{row.reason ?? '—'}</td>
                    <td className="text-sm text-muted">{row.created_at ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Subsection>
    </div>
  );
}

/**
 * Role-specific dashboard (adops, cfo, accountant, fraud).
 */
export function RoleDashboardPage() {
  const { role = '' } = useParams();
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const [customerInput, setCustomerInput] = useState(
    () => (sessionScoped ? boundCustomerId(user) : ''),
  );
  const [requested, setRequested] = useState(sessionScoped);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<RoleDashboardData | null>(null);
  const [blockError, setBlockError] = useState<Error | string | null>(null);
  const [exportJobState, setExportJobState] = useState<ExportJobState>({});
  const pollAbortRef = useRef<AbortController | null>(null);

  const customerError = sessionScoped ? null : validateCustomerIdField(customerInput);
  const customerId = sessionScoped ? boundCustomerId(user) : customerInput.trim();
  const canFetch = requested && Boolean(customerId) && !customerError;

  const downloadExportJob = useCallback(async (jobId: string) => {
    const [, err] = await to(downloadReportExport(jobId, `export-${jobId.slice(0, 8)}.csv`));
    if (err) {
      pushToastMessage({ title: 'Download failed', message: err.message ?? String(err) });
    }
  }, []);

  useEffect(() => {
    if (!canFetch) {
      setData(null);
      setBlockError(null);
      setLoading(false);
      return undefined;
    }

    const path = ENDPOINTS[role];
    if (!path) {
      setBlockError(new Error('Unknown dashboard role'));
      setData(null);
      return undefined;
    }

    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setBlockError(null);

    void (async () => {
      const params = new URLSearchParams({ customer_id: customerId });
      const [res, err] = await to(api(`${path}?${params.toString()}`, { signal: ctrl.signal }));
      if (cancelled) return;
      setLoading(false);
      if (err) {
        if (err.name === 'AbortError') return;
        setBlockError(err);
        setData(null);
        return;
      }
      const payload = (res?.data ?? null) as RoleDashboardData | null;
      setData(payload);
      if (role === 'accountant' && payload?.export_jobs) {
        setExportJobState((prev) => syncExportJobState(prev, payload.export_jobs ?? []));
      }
    })();

    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [canFetch, customerId, role]);

  const exportJobStateRef = useRef(exportJobState);
  exportJobStateRef.current = exportJobState;

  useEffect(() => {
    if (role !== 'accountant') return undefined;

    const timer = setInterval(() => {
      void (async () => {
        const state = exportJobStateRef.current;
        if (!hasPendingExportJobs(state)) return;
        pollAbortRef.current?.abort();
        const abort = new AbortController();
        pollAbortRef.current = abort;
        const ids = Object.keys(state);
        let changed = false;
        const next = { ...state };
        for (let i = 0; i < ids.length; i++) {
          const jobId = ids[i];
          const current = next[jobId]?.status ?? '';
          if (current !== 'PENDING' && current !== 'RUNNING') continue;
          const result = await pollReportJob(jobId, {
            signal: abort.signal,
            maxAttempts: 1,
            intervalMs: 0,
          });
          if (abort.signal.aborted) return;
          next[jobId] = {
            status: result.status,
            error: result.ok ? undefined : result.message,
          };
          changed = true;
        }
        if (changed) setExportJobState(next);
      })();
    }, 2500);

    return () => {
      clearInterval(timer);
      pollAbortRef.current?.abort();
      pollAbortRef.current = null;
    };
  }, [role, data?.export_jobs]);

  const body = useMemo(() => {
    if (!data) return null;
    if (role === 'adops') return <AdOpsBody data={data} />;
    if (role === 'cfo') return <CFOBody data={data} />;
    if (role === 'accountant') {
      return (
        <AccountantBody
          data={data}
          exportJobState={exportJobState}
          onDownload={downloadExportJob}
        />
      );
    }
    if (role === 'fraud') return <FraudBody data={data} />;
    return (
      <EmptyBlock
        title="Unknown role"
        description="This dashboard role is not configured."
      />
    );
  }, [data, role, exportJobState, downloadExportJob]);

  if (blockError) {
    return <ErrorBlock error={blockError} />;
  }

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">{TITLES[role] ?? 'Dashboard'}</h1>
      </div>
      <form
        className="filter-form section-block"
        onSubmit={(e) => {
          e.preventDefault();
          setRequested(true);
        }}
      >
        {!sessionScoped ? (
          <label className="form-field" htmlFor="role-dash-customer">
            Customer ID
            <input
              id="role-dash-customer"
              className={`form-input${customerError ? ' form-input--error' : ''}`}
              value={customerInput}
              onChange={(e) => setCustomerInput(e.target.value)}
            />
            {customerError ? <span className="form-field__error">{customerError}</span> : null}
          </label>
        ) : null}
        <Button
          label={t('action.load')}
          variant="primary"
          type="submit"
          loading={loading}
          disabled={loading}
        />
      </form>
      {loading ? <p className="loading-hint">{t('status.loading')}</p> : body}
    </>
  );
}
