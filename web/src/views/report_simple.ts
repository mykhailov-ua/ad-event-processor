import type { DataFreshness, ReportEnvelope, ReportRow } from '../types/api/report.js';
import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderFormField } from '../ui/form_field.js';
import { validateReportRange } from '../helpers/validators.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { renderFreshnessBadge } from '../ui/freshness_badge.js';
import { tenantReportQueryString } from '../helpers/tenant_url.js';
import { formatMoney } from '../helpers/money.js';
import { renderAlertBanner } from '../ui/alert_banner.js';
import { tableSkeletonRows, renderEmptyState } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';

export type ReportColumn = {
  key: string;
  label: string;
  format?: 'money' | 'pct' | 'rate' | 'number' | 'text';
};

export type SimpleReportOpts = {
  title: string;
  endpoint: string;
  columns: ReportColumn[];
  perm?: 'customers' | 'campaigns';
};

export function mountSimpleReport(container: HTMLElement, ctx: RouteContext, opts: SimpleReportOpts): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  let customerInput = ctx.query.get('customer_id') || (sessionScoped ? boundCustomerId(user) : '');
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];
  let from = ctx.query.get('from') || preset.from();
  let rangeTo = ctx.query.get('to') || preset.to();
  let loading = false;
  let rows: ReportRow[] = [];
  let freshness: DataFreshness | null = null;
  let error: Error | string | null = null;
  let validationError: string | null = null;

  function formatCell(row: ReportRow, col: ReportColumn) {
    const v = row[col.key];
    if (v == null || v === '') return '—';
    if (col.format === 'money') return formatMoney(v as string | number);
    if (col.format === 'rate') return `${(Number(v) * 100).toFixed(2)}%`;
    if (col.format === 'pct') return `${Number(v).toFixed(2)}%`;
    if (col.format === 'number') return String(v);
    if (typeof v === 'boolean') return v ? 'Yes' : 'No';
    return String(v);
  }

  async function load() {
    const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();
    const rangeErr = validateReportRange(from, rangeTo);
    if (!cid) {
      validationError = null;
      rows = [];
      error = null;
      render();
      return;
    }
    if (rangeErr) {
      validationError = rangeErr;
      rows = [];
      error = null;
      render();
      return;
    }
    validationError = null;
    loading = true;
    error = null;
    render();
    const params = new URLSearchParams({ customer_id: cid, from, to: rangeTo, limit: '100' });
    const [res, err] = await to(
      api<ReportEnvelope>(`/api/v1/reports/${opts.endpoint}?${params.toString()}`),
    );
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
    } else {
      const data = res?.data ?? null;
      rows = Array.isArray(data?.rows) ? data.rows : [];
      freshness = data?.freshness ?? null;
      if (!sessionScoped && cid) {
        const qs = tenantReportQueryString({ customer_id: cid, from, to: rangeTo });
        window.history.replaceState(null, '', `/reports/${opts.endpoint}?${qs}`);
      }
    }
    render();
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error));
      return;
    }
    const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();

    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, opts.title),
        el('p', { className: 'text-muted text-sm' },
          el('a', { href: '/reports' }, '← Reports hub'),
        ),
        freshness ? renderFreshnessBadge({ stale: freshness.stale, lagSeconds: freshness.ch_lag_seconds }) : null,
      ),
      el('form', {
        onSubmit: (e: Event) => { e.preventDefault(); load(); },
        className: 'mb-4',
      },
        !sessionScoped
          ? renderFormField({
            label: 'Customer ID',
            htmlFor: `report-${opts.endpoint}-customer`,
            children: el('input', {
              id: `report-${opts.endpoint}-customer`,
              className: 'form-input',
              value: customerInput,
              onInput: (e: Event) => { customerInput = eventTargetValue(e); },
            }),
          })
          : null,
        renderFormField({
          label: 'From',
          htmlFor: `report-${opts.endpoint}-from`,
          children: el('input', { id: `report-${opts.endpoint}-from`, className: 'form-input', value: from, onInput: (e: Event) => { from = eventTargetValue(e); } }),
        }),
        renderFormField({
          label: 'To',
          htmlFor: `report-${opts.endpoint}-to`,
          children: el('input', { id: `report-${opts.endpoint}-to`, className: 'form-input', value: rangeTo, onInput: (e: Event) => { rangeTo = eventTargetValue(e); } }),
        }),
        renderButton({
          label: 'Load',
          variant: 'primary',
          type: 'submit',
          loading,
          disabled: loading,
        }),
      ),
      validationError ? renderAlertBanner({ variant: 'error', message: validationError }) : null,
      !cid && !sessionScoped
        ? renderAlertBanner({ variant: 'info', message: 'Enter a customer UUID to load report data.' })
        : null,
      loading ? el('div', { className: 'table-wrapper' }, el('table', { className: 'data-table' }, el('tbody', null, tableSkeletonRows(opts.columns.length)))) : null,
      !loading && rows.length > 0
        ? el('div', { className: 'table-wrapper elevation-raised mt-4' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null, opts.columns.map((c: ReportColumn) => el('th', { scope: 'col' }, c.label))),
            ),
            el('tbody', null,
              rows.map((row, i) => el('tr', { key: `row-${i}` },
                opts.columns.map((c: ReportColumn) => el('td', null, formatCell(row, c))),
              )),
            ),
          ),
        )
        : (!loading && cid
          ? renderEmptyState({
            title: 'No rows',
            description: 'Try a different date range or filters.',
            icon: 'grid-four',
          })
          : null),
    );
  }

  load();
  return { destroy() { destroyed = true; } };
}
