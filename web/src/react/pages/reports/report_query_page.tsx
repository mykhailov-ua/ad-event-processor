import { useCallback, useEffect, useRef, useState } from 'react';
import { to } from '../../../lib/to.js';
import { api } from '../../../helpers/api_client.js';
import { createGenerationGuard, shouldCommitAsyncResult } from '../../../lib/async_guard.js';
import * as auth from '../../../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../../../helpers/buyer_session.js';
import { formatAmountMicro } from '../../../helpers/money.js';
import { mergeReportRows, visibleReportRows, MAX_REPORT_ROWS } from '../../../helpers/report_rows.js';
import { canShowReportFinancials } from '../../../helpers/report_mask.js';
import { validateReportRange, validateCustomerIdField } from '../../../helpers/validators.js';
import { touchCustomerContext } from '../../../helpers/customer_context.js';
import { REPORT_DATE_PRESETS } from '../../../helpers/date_presets.js';
import * as storage from '../../../helpers/storage.js';
import {
  isPageBlockingError,
  mapServiceError,
} from '../../../helpers/service_error.js';
import { surfaceServiceErrorToast } from '../../../helpers/service_error_toast.js';
import { t } from '../../../helpers/i18n.js';
import type { DataFreshness, ReportEnvelope, ReportRow } from '../../../types/api/report.js';
import { Button } from '../../components/button.js';
import { DatePickerHost } from '../../components/date_picker_host.js';
import { ErrorBlock } from '../../components/error_block.js';
import { FormField } from '../../components/form_field.js';
import { Icon } from '../../components/icon.js';
import { ImperativeDomHost } from '../../components/legacy_panel_host.js';
import { renderFreshnessBadge } from '../../../ui/freshness_badge.js';
import { renderStubBanner } from '../../../ui/stub_banner.js';

export type ReportQueryPageProps = {
  endpoint: 'placements' | 'keywords';
  title: string;
};

function defaultTo() {
  return new Date().toISOString();
}

function defaultFrom() {
  const d = new Date();
  d.setDate(d.getDate() - 7);
  return d.toISOString();
}

/**
 * Placements / keywords report with cursor pagination and compare period.
 */
export function ReportQueryPage({ endpoint, title }: ReportQueryPageProps) {
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const permissions = user?.permissions ?? [];
  const showFinancials = canShowReportFinancials(permissions);
  const savedRange = storage.getReportRange();

  const [customerInput, setCustomerInput] = useState(sessionScoped ? boundCustomerId(user) : '');
  const [from, setFrom] = useState(savedRange?.from ?? defaultFrom());
  const [rangeTo, setRangeTo] = useState(savedRange?.to ?? defaultTo());
  const [rangeError, setRangeError] = useState<string | null>(null);
  const [customerError, setCustomerError] = useState<string | null>(null);
  const [activePreset, setActivePreset] = useState('');
  const [comparePeriod, setComparePeriod] = useState(false);
  const [rows, setRows] = useState<ReportRow[]>([]);
  const [freshness, setFreshness] = useState<DataFreshness | null>(null);
  const [nextCursor, setNextCursor] = useState('');
  const [loading, setLoading] = useState(false);
  const [fetchError, setFetchError] = useState<unknown>(null);
  const lastFetchErrorRef = useRef<unknown>(null);
  const rowsRef = useRef<ReportRow[]>([]);
  const fetchGuardRef = useRef(createGenerationGuard());
  const fetchAbortRef = useRef<AbortController | null>(null);

  rowsRef.current = rows;

  const customerId = () => (sessionScoped ? boundCustomerId(user) : customerInput.trim());

  const fetchReport = useCallback(async (cursor = '') => {
    const custErr = sessionScoped ? null : validateCustomerIdField(customerInput);
    const rangeErr = validateReportRange(from, rangeTo);
    if (rangeErr) {
      setRangeError(rangeErr);
      return;
    }
    if (custErr) {
      setCustomerError(custErr);
      setRangeError(null);
      return;
    }
    setRangeError(null);
    const opGen = fetchGuardRef.current.next();
    fetchAbortRef.current?.abort();
    const ctrl = new AbortController();
    fetchAbortRef.current = ctrl;
    setLoading(true);
    setFetchError(null);
    storage.setReportRange({ from, to: rangeTo });
    if (!sessionScoped) touchCustomerContext(customerInput.trim());

    const params = new URLSearchParams({
      customer_id: customerId(),
      from,
      to: rangeTo,
      limit: '50',
    });
    if (cursor) params.set('cursor', cursor);
    if (comparePeriod && !cursor) params.set('compare', 'previous');

    const [apiRes, apiErr] = await to(api(`/api/v1/reports/${endpoint}?${params.toString()}`, {
      signal: ctrl.signal,
    }));
    if (!shouldCommitAsyncResult(opGen, fetchGuardRef.current.current())) return;
    if (apiErr) {
      if (apiErr.name === 'AbortError') return;
      setFetchError(apiErr);
      setLoading(false);
      if (apiErr !== lastFetchErrorRef.current) {
        lastFetchErrorRef.current = apiErr;
        surfaceServiceErrorToast(apiErr);
      }
      return;
    }
    const data = (apiRes?.data as ReportEnvelope | null) ?? null;
    const batch = data?.rows ?? [];
    const [mergedRows, mergeErr] = await to(
      cursor ? mergeReportRows(rowsRef.current, batch) : mergeReportRows([], batch),
    );
    if (!shouldCommitAsyncResult(opGen, fetchGuardRef.current.current())) return;
    if (mergeErr) {
      setFetchError(mergeErr);
    } else {
      const nextRows = mergedRows ?? [];
      setRows(nextRows);
      setFreshness(data?.freshness ?? null);
      setNextCursor(nextRows.length >= MAX_REPORT_ROWS ? '' : (data?.next_cursor ?? ''));
    }
    setLoading(false);
  }, [sessionScoped, customerInput, from, rangeTo, comparePeriod, endpoint, user]);

  useEffect(() => () => {
    fetchGuardRef.current.invalidate();
    fetchAbortRef.current?.abort();
  }, []);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setRows([]);
    setNextCursor('');
    void fetchReport('');
  };

  const applyPreset = (preset: typeof REPORT_DATE_PRESETS[number]) => {
    setActivePreset(preset.id);
    setFrom(preset.from());
    setRangeTo(preset.to());
    setRangeError(validateReportRange(preset.from(), preset.to()));
  };

  if (fetchError) {
    const view = mapServiceError(fetchError);
    if (view.kind === 'stub') {
      return (
        <>
          <div className="page-header">
            <h1 className="page-header__title">{title}</h1>
          </div>
          <ImperativeDomHost
            build={() => renderStubBanner({
              message: view.message,
              linkTo: '/reports/placements',
              linkLabel: 'Report: placements',
            })}
            deps={[view.message]}
          />
        </>
      );
    }
    if (isPageBlockingError(view) || view.kind === 'empty') {
      return <ErrorBlock error={fetchError} />;
    }
  }

  const isPlacements = endpoint === 'placements';
  const { visible: tableRows } = visibleReportRows(rows);
  const buildFreshness = () => renderFreshnessBadge({
    stale: freshness?.stale,
    lagSeconds: freshness?.ch_lag_seconds,
  });

  return (
    <>
      <div className="page-header">
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="file-spreadsheet" size={20} className="text-muted" />
            <h1 className="page-header__title">{title}</h1>
          </div>
          {freshness ? <ImperativeDomHost build={buildFreshness} deps={[freshness]} /> : null}
        </div>
      </div>

      <form className="mb-4" onSubmit={handleSearch}>
        {!sessionScoped ? (
          <FormField
            label="Customer ID"
            htmlFor="report-customer-id"
            error={customerError}
            hint="UUID of the customer to query"
          >
            <input
              id="report-customer-id"
              className="form-input"
              value={customerInput}
              required
              onChange={(e) => {
                setCustomerInput(e.target.value);
                setCustomerError(validateCustomerIdField(e.target.value));
              }}
            />
          </FormField>
        ) : null}
        {sessionScoped && customerId() ? (
          <p className="text-muted text-sm mb-3">
            Customer: <span className="font-mono">{customerId()}</span>
          </p>
        ) : null}

        <div className="date-presets">
          <span className="date-presets__label">Range</span>
          {REPORT_DATE_PRESETS.map((preset) => (
            <button
              key={preset.id}
              type="button"
              className={`date-preset${activePreset === preset.id ? ' date-preset--active' : ''}`}
              onClick={() => applyPreset(preset)}
            >
              {preset.label}
            </button>
          ))}
        </div>

        <label className="form-checkbox form-checkbox--block">
          <input
            type="checkbox"
            checked={comparePeriod}
            onChange={(e) => setComparePeriod(e.target.checked)}
          />
          <span>{t('report.compare', 'Compare with previous period')}</span>
        </label>

        <div className="form-row">
          <FormField label="From date & time" htmlFor="report-from">
            <DatePickerHost
              id="report-from"
              value={from}
              onChange={(iso) => {
                setFrom(iso);
                setActivePreset('');
                setRangeError(validateReportRange(iso, rangeTo));
              }}
            />
          </FormField>
          <FormField label="To date & time" htmlFor="report-to" error={rangeError}>
            <DatePickerHost
              id="report-to"
              value={rangeTo}
              onChange={(iso) => {
                setRangeTo(iso);
                setActivePreset('');
                setRangeError(validateReportRange(from, iso));
              }}
            />
          </FormField>
          <FormField label=" ">
            <Button
              label="Load"
              variant="primary"
              type="submit"
              icon="search"
              className="form-submit-btn"
              loading={loading}
              disabled={loading}
            />
          </FormField>
        </div>
      </form>

      {loading && rows.length === 0 ? (
        <div className="table-wrapper elevation-raised">
          <table className="data-table">
            <tbody>
              {Array.from({ length: 8 }, (_, i) => (
                <tr key={`sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
                  {Array.from({ length: 6 }, (__, j) => (
                    <td key={j}><span className="skeleton-bar" /></td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {rows.length > 0 ? (
        <div className="table-wrapper elevation-raised">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">{isPlacements ? 'Placement' : 'Keyword'}</th>
                <th scope="col">Campaign</th>
                <th scope="col">Impr.</th>
                <th scope="col">Clicks</th>
                <th scope="col">Conv.</th>
                {showFinancials ? <th scope="col">Spend</th> : null}
                {showFinancials ? <th scope="col">Revenue</th> : null}
                {showFinancials ? <th scope="col">ROI %</th> : null}
                <th scope="col">CTR %</th>
                {showFinancials ? <th scope="col">CPA</th> : null}
                <th scope="col">IVT %</th>
                {comparePeriod ? <th scope="col">Δ spend</th> : null}
              </tr>
            </thead>
            <tbody>
              {tableRows.map((row, index) => {
                const spendDelta = comparePeriod && showFinancials
                  ? (row.compare as { spend_micro_delta?: number } | undefined)?.spend_micro_delta
                  : null;
                return (
                  <tr key={`${isPlacements ? row.placement_id : row.keyword}-${index}`}>
                    <td className="font-mono">
                      {String(isPlacements ? row.placement_id : row.keyword)}
                    </td>
                    <td className="font-mono text-muted">{String(row.campaign_id ?? '')}</td>
                    <td>{String(row.impressions ?? 0)}</td>
                    <td>{String(row.clicks ?? 0)}</td>
                    <td>{String(row.conversions ?? 0)}</td>
                    {showFinancials ? (
                      <td className="font-mono">{formatAmountMicro(Number(row.spend_micro ?? 0))}</td>
                    ) : null}
                    {showFinancials ? (
                      <td className="font-mono">{formatAmountMicro(Number(row.revenue_micro ?? 0))}</td>
                    ) : null}
                    {showFinancials ? (
                      <td>{row.roi_pct != null ? Number(row.roi_pct).toFixed(2) : '—'}</td>
                    ) : null}
                    <td>{row.ctr != null ? (Number(row.ctr) * 100).toFixed(2) : '—'}</td>
                    {showFinancials ? (
                      <td className="font-mono">
                        {row.cpa_micro != null ? formatAmountMicro(Number(row.cpa_micro)) : '—'}
                      </td>
                    ) : null}
                    <td>{row.ivt_rate != null ? (Number(row.ivt_rate) * 100).toFixed(2) : '—'}</td>
                    {comparePeriod ? (
                      <td className="font-mono">
                        {spendDelta != null ? formatAmountMicro(spendDelta) : '—'}
                      </td>
                    ) : null}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}

      {nextCursor && rows.length < MAX_REPORT_ROWS ? (
        <Button
          label="Load more"
          variant="secondary"
          size="sm"
          className="mt-4"
          loading={loading}
          disabled={loading}
          onClick={() => void fetchReport(nextCursor)}
        />
      ) : null}
    </>
  );
}
