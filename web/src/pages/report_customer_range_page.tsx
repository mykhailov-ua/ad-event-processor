import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import type { DataFreshness, ReportEnvelope, ReportRow } from '../types/report.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { validateReportRange } from '../helpers/validators.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { tenantReportQueryString } from '../helpers/tenant_url.js';
import { formatMoney } from '../helpers/money.js';
import { reportCompareLabel, formatSpendDelta } from '../helpers/report_compare.js';
import { ReportRowActions } from '../components/report_row_actions.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FormField } from '../components/form_field.js';
import { FreshnessBadge } from '../components/freshness_badge.js';

export type CustomerRangeColumn = {
  header: string;
  render: (row: ReportRow) => ReactNode;
};

export type CustomerRangeReportPageProps = {
  title: string;
  endpoint: string;
  urlPath: string;
  columns: CustomerRangeColumn[];
  enableCompare?: boolean;
  enableActions?: boolean;
  requireCustomer?: boolean;
};

export function CustomerRangeReportPage({
  title,
  endpoint,
  urlPath,
  columns,
  enableCompare = true,
  enableActions = true,
  requireCustomer = true,
}: CustomerRangeReportPageProps) {
  const [searchParams] = useSearchParams();
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];

  const [customerInput, setCustomerInput] = useState(
    searchParams.get('customer_id') || (sessionScoped ? boundCustomerId(user) : '')
  );
  const [from, setFrom] = useState(searchParams.get('from') || preset.from());
  const [rangeTo, setRangeTo] = useState(searchParams.get('to') || preset.to());
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<ReportRow[]>([]);
  const [freshness, setFreshness] = useState<DataFreshness | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [comparePeriod, setComparePeriod] = useState(false);

  const load = useCallback(async () => {
    const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();
    const rangeErr = validateReportRange(from, rangeTo);
    if (requireCustomer && !cid) {
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
    const params = new URLSearchParams({ from, to: rangeTo, limit: '50' });
    if (cid) params.set('customer_id', cid);
    if (comparePeriod) params.set('compare', 'previous');
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
  }, [
    sessionScoped,
    user,
    customerInput,
    from,
    rangeTo,
    endpoint,
    urlPath,
    comparePeriod,
    requireCustomer,
  ]);

  useEffect(() => {
    void load();
  }, []);

  if (error) return <ErrorBlock error={error} />;

  const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">{title}</h1>
        {freshness ? (
          <FreshnessBadge stale={freshness.stale} lagSeconds={freshness.ch_lag_seconds} />
        ) : null}
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          void load();
        }}
      >
        {!requireCustomer || sessionScoped ? null : (
          <FormField label="Customer ID" htmlFor={`${endpoint}-customer`}>
            <input
              id={`${endpoint}-customer`}
              className="form-input"
              placeholder="Customer UUID..."
              value={customerInput}
              onChange={(e) => setCustomerInput(e.target.value)}
            />
          </FormField>
        )}
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
        {enableCompare ? (
          <label className="form-checkbox form-checkbox--block">
            <input
              type="checkbox"
              checked={comparePeriod}
              onChange={(e) => setComparePeriod(e.target.checked)}
            />
            <span>{reportCompareLabel()}</span>
          </label>
        ) : null}
        <Button label="Load" variant="primary" type="submit" loading={loading} disabled={loading} />
      </form>

      {validationError ? <AlertBanner variant="error" message={validationError} /> : null}
      {requireCustomer && !cid && !sessionScoped ? (
        <AlertBanner variant="info" message="Enter a customer UUID to load report data." />
      ) : null}

      {loading ? <p className="text-muted mt-4">Loading...</p> : null}

      {rows.length > 0 ? (
        <div className="table-wrapper elevation-raised mt-4">
          <table className="data-table">
            <thead>
              <tr>
                {columns.map((col) => (
                  <th key={col.header} scope="col">
                    {col.header}
                  </th>
                ))}
                {comparePeriod ? <th scope="col">delta spend</th> : null}
                {enableActions ? <th scope="col">Actions</th> : null}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={`row-${index}`}>
                  {columns.map((col) => (
                    <td key={col.header}>{col.render(row)}</td>
                  ))}
                  {comparePeriod ? <td className="font-mono">{formatSpendDelta(row)}</td> : null}
                  {enableActions ? (
                    <td>
                      <ReportRowActions row={row} customerId={cid} reportEndpoint={endpoint} />
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {(!requireCustomer || cid) && !loading && rows.length === 0 ? (
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
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>
        {String(row.campaign_id ?? '')}
      </Link>
    ),
  },
  { header: 'Sub1', render: (row) => <span className="font-mono">{String(row.sub1 ?? '-')}</span> },
  { header: 'Sub2', render: (row) => <span className="font-mono">{String(row.sub2 ?? '-')}</span> },
  { header: 'Country', render: (row) => String(row.country ?? '-') },
  { header: 'Impr.', render: (row) => String(row.impressions ?? 0) },
  { header: 'IVT', render: (row) => String(row.ivt_events ?? 0) },
  {
    header: 'IVT %',
    render: (row) => (row.ivt_rate != null ? `${(Number(row.ivt_rate) * 100).toFixed(2)}%` : '-'),
  },
];

export const TRAFFIC_REPORT_COLUMNS: CustomerRangeColumn[] = [
  { header: 'Channel', render: (row) => String(row.channel ?? '-') },
  { header: 'Impr.', render: (row) => String(row.impressions ?? 0) },
  { header: 'Clicks', render: (row) => String(row.clicks ?? 0) },
  { header: 'Spend', render: (row) => formatMoney(row.spend_micro as string | number) },
  {
    header: 'ROI %',
    render: (row) => (row.roi_pct != null ? `${Number(row.roi_pct).toFixed(2)}%` : '-'),
  },
];

export const GEO_REPORT_COLUMNS: CustomerRangeColumn[] = [
  { header: 'Country', render: (row) => String(row.country ?? '-') },
  { header: 'Clicks', render: (row) => String(row.clicks ?? 0) },
  {
    header: 'IVT %',
    render: (row) => (row.ivt_rate != null ? `${(Number(row.ivt_rate) * 100).toFixed(2)}%` : '-'),
  },
  { header: 'Spend', render: (row) => formatMoney(row.spend_micro as string | number) },
  {
    header: 'ROI %',
    render: (row) => (row.roi_pct != null ? `${Number(row.roi_pct).toFixed(2)}%` : '-'),
  },
];

export const DATA_QUALITY_REPORT_COLUMNS: CustomerRangeColumn[] = [
  {
    header: 'Campaign',
    render: (row) => (
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>
        {String(row.campaign_id ?? '')}
      </Link>
    ),
  },
  { header: 'Date', render: (row) => String(row.date ?? '-') },
  { header: 'PG total', render: (row) => String(row.pg_total ?? 0) },
  { header: 'CH total', render: (row) => String(row.ch_total ?? 0) },
  {
    header: 'Diff %',
    render: (row) => (row.diff_pct != null ? `${(Number(row.diff_pct) * 100).toFixed(2)}%` : '-'),
  },
  { header: 'Severity', render: (row) => String(row.severity ?? '-') },
];

export const COST_SYNC_COVERAGE_REPORT_COLUMNS: CustomerRangeColumn[] = [
  {
    header: 'Campaign',
    render: (row) => (
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>
        {String(row.campaign_id ?? '')}
      </Link>
    ),
  },
  { header: 'Clicks', render: (row) => String(row.clicks ?? 0) },
  { header: 'Spend', render: (row) => formatMoney(row.spend_micro as string | number) },
  { header: 'Gap', render: (row) => String(row.coverage_gap ?? '-') },
  { header: 'Network', render: (row) => String(row.network ?? '-') },
  { header: 'Last sync', render: (row) => String(row.last_sync_status ?? '-') },
];

export const FILTER_REJECT_REPORT_COLUMNS: CustomerRangeColumn[] = [
  { header: 'Reject kind', render: (row) => String(row.reject_kind ?? '-') },
  { header: 'Count', render: (row) => String(row.reject_count ?? 0) },
];

export const FRAUD_BREAKDOWN_REPORT_COLUMNS: CustomerRangeColumn[] = [
  {
    header: 'Campaign',
    render: (row) => (
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>
        {String(row.campaign_id ?? '')}
      </Link>
    ),
  },
  { header: 'Placement', render: (row) => String(row.placement_id ?? '-') },
  { header: 'Reason', render: (row) => String(row.fraud_reason ?? '-') },
  { header: 'Events', render: (row) => String(row.event_count ?? 0) },
  { header: 'Silent reject', render: (row) => String(row.silent_reject_count ?? 0) },
  {
    header: 'Silent reject %',
    render: (row) =>
      row.silent_reject_ratio != null
        ? `${(Number(row.silent_reject_ratio) * 100).toFixed(2)}%`
        : '-',
  },
];

export const SILENT_REJECT_IMPRESSION_FUNNEL_COLUMNS: CustomerRangeColumn[] = [
  {
    header: 'Campaign',
    render: (row) => (
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>
        {String(row.campaign_id ?? '')}
      </Link>
    ),
  },
  { header: 'Placement', render: (row) => String(row.placement_id ?? '-') },
  { header: 'Billable imps', render: (row) => String(row.billable_impressions ?? 0) },
  { header: 'Silent reject imps', render: (row) => String(row.silent_reject_impressions ?? 0) },
  { header: 'IVT imps', render: (row) => String(row.ivt_impressions ?? 0) },
  {
    header: 'Silent reject %',
    render: (row) =>
      row.silent_reject_rate != null
        ? `${(Number(row.silent_reject_rate) * 100).toFixed(2)}%`
        : '-',
  },
  {
    header: 'IVT %',
    render: (row) =>
      row.ivt_impression_rate != null
        ? `${(Number(row.ivt_impression_rate) * 100).toFixed(2)}%`
        : '-',
  },
];

export const RTB_OVERVIEW_REPORT_COLUMNS: CustomerRangeColumn[] = [
  { header: 'Deal', render: (row) => String(row.deal_id ?? '-') },
  { header: 'Bids', render: (row) => String(row.bids ?? 0) },
  { header: 'Wins', render: (row) => String(row.wins ?? 0) },
  {
    header: 'Win rate',
    render: (row) => (row.win_rate != null ? `${(Number(row.win_rate) * 100).toFixed(2)}%` : '-'),
  },
  { header: 'Spend', render: (row) => formatMoney(row.spend_micro as string | number) },
];

export const RTB_NO_BID_REPORT_COLUMNS: CustomerRangeColumn[] = [
  { header: 'Reason', render: (row) => String(row.no_bid_reason ?? '-') },
  { header: 'Count', render: (row) => String(row.bid_count ?? 0) },
];

export const RTB_GEO_DEVICE_REPORT_COLUMNS: CustomerRangeColumn[] = [
  { header: 'Country', render: (row) => String(row.country ?? '-') },
  { header: 'Device OS', render: (row) => String(row.device_os ?? '-') },
  { header: 'Bids', render: (row) => String(row.bids ?? 0) },
  { header: 'Wins', render: (row) => String(row.wins ?? 0) },
  {
    header: 'Win rate',
    render: (row) => (row.win_rate != null ? `${(Number(row.win_rate) * 100).toFixed(2)}%` : '-'),
  },
  { header: 'Spend', render: (row) => formatMoney(row.spend_micro as string | number) },
];

export const CONVERSION_TYPE_PAYOUT_REPORT_COLUMNS: CustomerRangeColumn[] = [
  {
    header: 'Campaign',
    render: (row) => (
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>
        {String(row.campaign_id ?? '')}
      </Link>
    ),
  },
  { header: 'Goal / type', render: (row) => String(row.goal_name ?? '-') },
  { header: 'Conversions', render: (row) => String(row.conversions ?? '-') },
  { header: 'Payout', render: (row) => formatMoney(row.payout_micro as string | number) },
];

export const POSTBACK_RECON_REPORT_COLUMNS: CustomerRangeColumn[] = [
  {
    header: 'Campaign',
    render: (row) => (
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>
        {String(row.campaign_id ?? '')}
      </Link>
    ),
  },
  {
    header: 'Click ID',
    render: (row) => <span className="font-mono">{String(row.click_id ?? '-')}</span>,
  },
  { header: 'Conversion', render: (row) => String(row.conversion_at ?? '-') },
  { header: 'Value', render: (row) => formatMoney(row.conversion_value_micro as string | number) },
  { header: 'Day fee', render: (row) => formatMoney(row.ledger_day_fee_micro as string | number) },
  { header: 'Postback', render: (row) => String(row.postback_status ?? '-') },
  { header: 'Status', render: (row) => String(row.reconcile_status ?? '-') },
];

export const PACING_DRIFT_REPORT_COLUMNS: CustomerRangeColumn[] = [
  {
    header: 'Campaign',
    render: (row) => (
      <Link to={`/campaigns/${String(row.campaign_id ?? '')}`}>
        {String(row.campaign_id ?? '')}
      </Link>
    ),
  },
  { header: 'Date', render: (row) => String(row.date ?? '-') },
  { header: 'Planned', render: (row) => formatMoney(row.planned_spend_micro as string | number) },
  { header: 'Actual', render: (row) => formatMoney(row.actual_spend_micro as string | number) },
  {
    header: 'Drift %',
    render: (row) => (row.drift_pct != null ? `${(Number(row.drift_pct) * 100).toFixed(2)}%` : '-'),
  },
  { header: 'Pacing', render: (row) => String(row.pacing_mode ?? '-') },
];
