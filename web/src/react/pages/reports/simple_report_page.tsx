import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
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
import type { SimpleReportColumn } from './report_configs.js';

function formatCell(row: ReportRow, col: SimpleReportColumn) {
  const v = row[col.key];
  if (v == null || v === '') return '—';
  if (col.format === 'money') return formatMoney(v as string | number);
  if (col.format === 'rate') return `${(Number(v) * 100).toFixed(2)}%`;
  if (col.format === 'pct') return `${Number(v).toFixed(2)}%`;
  if (col.format === 'number') return String(v);
  if (typeof v === 'boolean') return v ? 'Yes' : 'No';
  return String(v);
}

export type SimpleReportPageProps = {
  title: string;
  endpoint: string;
  columns: SimpleReportColumn[];
};

/**
 * Generic customer + date range report table.
 */
export function SimpleReportPage({ title, endpoint, columns }: SimpleReportPageProps) {
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
    const params = new URLSearchParams({ customer_id: cid, from, to: rangeTo, limit: '100' });
    const [res, err] = await to(api<ReportEnvelope>(`/api/v1/reports/${endpoint}?${params.toString()}`));
    setLoading(false);
    if (err) {
      setError(err);
      return;
    }
    const data = res?.data ?? null;
    setRows(Array.isArray(data?.rows) ? data.rows : []);
    setFreshness(data?.freshness ?? null);
    if (!sessionScoped && cid) {
      const qs = tenantReportQueryString({ customer_id: cid, from, to: rangeTo });
      window.history.replaceState(null, '', `/reports/${endpoint}?${qs}`);
    }
  }, [sessionScoped, user, customerInput, from, rangeTo, endpoint]);

  useEffect(() => {
    void load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  if (error) return <ErrorBlock error={error} />;

  const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">{title}</h1>
        <p className="text-muted text-sm">
          <a href="/reports">← Reports hub</a>
        </p>
        {freshness ? <ImperativeDomHost build={buildFreshness} deps={[freshness]} /> : null}
      </div>

      <form
        className="mb-4"
        onSubmit={(e) => {
          e.preventDefault();
          void load();
        }}
      >
        {!sessionScoped ? (
          <FormField label="Customer ID" htmlFor={`report-${endpoint}-customer`}>
            <input
              id={`report-${endpoint}-customer`}
              className="form-input"
              value={customerInput}
              onChange={(e) => setCustomerInput(e.target.value)}
            />
          </FormField>
        ) : null}
        <FormField label="From" htmlFor={`report-${endpoint}-from`}>
          <input
            id={`report-${endpoint}-from`}
            className="form-input"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
        </FormField>
        <FormField label="To" htmlFor={`report-${endpoint}-to`}>
          <input
            id={`report-${endpoint}-to`}
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

      {loading ? (
        <div className="table-wrapper">
          <table className="data-table">
            <tbody>
              {Array.from({ length: 5 }, (_, i) => (
                <tr key={`sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
                  {columns.map((col) => (
                    <td key={col.key}><span className="skeleton-bar" /></td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {!loading && rows.length > 0 ? (
        <div className="table-wrapper elevation-raised mt-4">
          <table className="data-table">
            <thead>
              <tr>
                {columns.map((c) => <th key={c.key} scope="col">{c.label}</th>)}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={`row-${index}`}>
                  {columns.map((c) => (
                    <td key={c.key}>{formatCell(row, c)}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {!loading && cid && rows.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state__title">No rows</div>
          <div className="empty-state__desc text-muted text-sm">
            Try a different date range or filters.
          </div>
        </div>
      ) : null}
    </>
  );
}
