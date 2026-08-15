import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import type { DataFreshness, ReportEnvelope, ReportRow } from '../../../types/api/report.js';
import { to } from '../../../lib/to.js';
import { api } from '../../../helpers/api_client.js';
import * as auth from '../../../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../../../helpers/buyer_session.js';
import { validateReportRange } from '../../../helpers/validators.js';
import { REPORT_DATE_PRESETS } from '../../../helpers/date_presets.js';
import { tenantReportQueryString } from '../../../helpers/tenant_url.js';
import { formatMoney } from '../../../helpers/money.js';
import { AlertBanner } from '../../components/alert_banner.js';
import { Button } from '../../components/button.js';
import { ErrorBlock } from '../../components/error_block.js';
import { FormField } from '../../components/form_field.js';
import { ImperativeDomHost } from '../../components/legacy_panel_host.js';
import { renderFreshnessBadge } from '../../../ui/freshness_badge.js';

export type CustomerRangeColumn = {
  header: string;
  render: (row: ReportRow) => ReactNode;
};

export type CustomerRangeReportPageProps = {
  title: string;
  endpoint: string;
  urlPath: string;
  columns: CustomerRangeColumn[];
};

/**
 * Customer + date range report with custom column renderers (IVT, traffic, geo).
 */
export function CustomerRangeReportPage({
  title,
  endpoint,
  urlPath,
  columns,
}: CustomerRangeReportPageProps) {
  const [searchParams] = useSearchParams();
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];

  const [customerInput, setCustomerInput] = useState(
    searchParams.get('customer_id') || (sessionScoped ? boundCustomerId(user) : ''),
  );
  const [from, setFrom] = useState(searchParams.get('from') || preset.from());
  const [rangeTo, setRangeTo] = useState(searchParams.get('to') || preset.to());
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<ReportRow[]>([]);
  const [freshness, setFreshness] = useState<DataFreshness | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [validationError, setValidationError] = useState<string | null>(null);

  const buildFreshness = useCallback(
    () => renderFreshnessBadge({
      stale: freshness?.stale,
      lagSeconds: freshness?.ch_lag_seconds,
    }),
    [freshness],
  );

  const load = useCallback(async () => {
    const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();
    const rangeErr = validateReportRange(from, rangeTo);
    if (!cid) {
      setValidationError(null);
      setRows([]);
      setError(null);
      return;
    }
    if (rangeErr) {
      setValidationError(rangeErr);
      setRows([]);
      setError(null);
      return;
    }
    setValidationError(null);
    setLoading(true);
    setError(null);
    const params = new URLSearchParams({ customer_id: cid, from, to: rangeTo, limit: '50' });
    const [res, err] = await to(api(`/api/v1/reports/${endpoint}?${params.toString()}`));
    setLoading(false);
    if (err) {
      setError(err);
      return;
    }
    const data = (res?.data as ReportEnvelope | null) ?? null;
    setRows(data?.rows ?? []);
    setFreshness(data?.freshness ?? null);
    if (!sessionScoped && cid) {
      const qs = tenantReportQueryString({ customer_id: cid, from, to: rangeTo });
      window.history.replaceState(null, '', `${urlPath}?${qs}`);
    }
  }, [sessionScoped, user, customerInput, from, rangeTo, endpoint, urlPath]);

  useEffect(() => {
    void load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  if (error) return <ErrorBlock error={error} />;

  const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">{title}</h1>
        {freshness ? <ImperativeDomHost build={buildFreshness} deps={[freshness]} /> : null}
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          void load();
        }}
      >
        {!sessionScoped ? (
          <FormField label="Customer ID" htmlFor={`${endpoint}-customer`}>
            <input
              id={`${endpoint}-customer`}
              className="form-input"
              placeholder="Customer UUID…"
              value={customerInput}
              onChange={(e) => setCustomerInput(e.target.value)}
            />
          </FormField>
        ) : null}
        <FormField label="From" htmlFor={`${endpoint}-from`}>
          <input
            id={`${endpoint}-from`}
            className="form-input"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
        </FormField>
        <FormField label="To" htmlFor={`${endpoint}-to`}>
          <input
            id={`${endpoint}-to`}
            className="form-input"
            value={rangeTo}
            onChange={(e) => setRangeTo(e.target.value)}
          />
        </FormField>
        <Button label="Load" variant="primary" type="submit" loading={loading} disabled={loading} />
      </form>

      {validationError ? <AlertBanner variant="error" message={validationError} /> : null}
      {!cid && !sessionScoped ? (
        <AlertBanner variant="info" message="Enter a customer UUID to load report data." />
      ) : null}

      {loading ? <p className="text-muted mt-4">Loading…</p> : null}

      {rows.length > 0 ? (
        <div className="table-wrapper elevation-raised mt-4">
          <table className="data-table">
            <thead>
              <tr>
                {columns.map((col) => <th key={col.header} scope="col">{col.header}</th>)}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={`row-${index}`}>
                  {columns.map((col) => <td key={col.header}>{col.render(row)}</td>)}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {cid && !loading && rows.length === 0 ? (
        <div className="empty-state mt-4">
          <div className="empty-state__title">No rows</div>
          <div className="empty-state__desc text-muted text-sm">
            Try a different date range or filters.
          </div>
        </div>
      ) : null}
    </>
  );
}

export const IVT_REPORT_COLUMNS: CustomerRangeColumn[] = [
  {
    header: 'Campaign',
    render: (row) => (
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>{String(row.campaign_id ?? '')}</Link>
    ),
  },
  { header: 'Sub1', render: (row) => <span className="font-mono">{String(row.sub1 ?? '—')}</span> },
  { header: 'Sub2', render: (row) => <span className="font-mono">{String(row.sub2 ?? '—')}</span> },
  { header: 'Country', render: (row) => String(row.country ?? '—') },
  { header: 'Impr.', render: (row) => String(row.impressions ?? 0) },
  { header: 'IVT', render: (row) => String(row.ivt_events ?? 0) },
  {
    header: 'IVT %',
    render: (row) => (row.ivt_rate != null ? `${(Number(row.ivt_rate) * 100).toFixed(2)}%` : '—'),
  },
];

export const TRAFFIC_REPORT_COLUMNS: CustomerRangeColumn[] = [
  { header: 'Channel', render: (row) => String(row.channel ?? '—') },
  { header: 'Impr.', render: (row) => String(row.impressions ?? 0) },
  { header: 'Clicks', render: (row) => String(row.clicks ?? 0) },
  { header: 'Spend', render: (row) => formatMoney(row.spend_micro as string | number) },
  {
    header: 'ROI %',
    render: (row) => (row.roi_pct != null ? `${Number(row.roi_pct).toFixed(2)}%` : '—'),
  },
];

export const GEO_REPORT_COLUMNS: CustomerRangeColumn[] = [
  { header: 'Country', render: (row) => String(row.country ?? '—') },
  { header: 'Clicks', render: (row) => String(row.clicks ?? 0) },
  {
    header: 'IVT %',
    render: (row) => (row.ivt_rate != null ? `${(Number(row.ivt_rate) * 100).toFixed(2)}%` : '—'),
  },
  { header: 'Spend', render: (row) => formatMoney(row.spend_micro as string | number) },
  {
    header: 'ROI %',
    render: (row) => (row.roi_pct != null ? `${Number(row.roi_pct).toFixed(2)}%` : '—'),
  },
];
