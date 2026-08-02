import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { renderFreshnessBadge } from '../ui/freshness_badge.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderStubBanner } from '../ui/stub_banner.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { api } from '../helpers/api_client.js';
import { formatAmountMicro } from '../helpers/money.js';
import { mergeReportRows, visibleReportRows, MAX_REPORT_ROWS } from '../helpers/report_rows.js';
import { canShowReportFinancials } from '../helpers/report_mask.js';
import { validateReportRange, validateCustomerIdField } from '../helpers/validators.js';
import { touchCustomerContext } from '../helpers/customer_context.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { renderFormField } from '../ui/form_field.js';
import { tableSkeletonRows } from '../ui/data_table.js';
import * as storage from '../helpers/storage.js';
import { renderIcon } from '../ui/icon.js';
import {
  isPageBlockingError,
  mapServiceError,
} from '../helpers/service_error.js';
import { t } from '../helpers/i18n.js';

/**
 * Mount a report query view with date presets and cursor pagination.
 *
 * @param {HTMLElement} container
 * @param {import('../lib/router.js').RouteContext} ctx
 * @param {{ endpoint: 'placements'|'keywords', title: string, rowKey: (row: object) => string }} opts
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mountReportQuery(container, ctx, opts) {
  let destroyed = false;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  const permissions = user?.permissions ?? [];
  const showFinancials = canShowReportFinancials(permissions);
  const savedRange = storage.getReportRange();

  let customerInput = sessionScoped ? boundCustomerId(user) : '';
  let from = savedRange?.from ?? defaultFrom();
  let rangeTo = savedRange?.to ?? defaultTo();
  let rangeError = null;
  let customerError = null;
  let activePreset = '';
  let comparePeriod = false;
  /** @type {object[]} */
  let rows = [];
  let freshness = null;
  let nextCursor = '';
  let loading = false;
  let fetchError = null;
  let lastFetchError = null;
  const fetchGuard = createGenerationGuard();
  /** @type {AbortController|null} */
  let fetchAbort = null;

  const customerId = () => (sessionScoped ? boundCustomerId(user) : customerInput.trim());

  async function fetchReport(cursor = '') {
    customerError = sessionScoped ? null : validateCustomerIdField(customerInput);
    const rangeErr = validateReportRange(from, rangeTo);
    if (rangeErr) {
      rangeError = rangeErr;
      render();
      return;
    }
    if (customerError) {
      rangeError = null;
      render();
      return;
    }
    rangeError = null;
    const opGen = fetchGuard.next();
    if (fetchAbort) fetchAbort.abort();
    const ctrl = new AbortController();
    fetchAbort = ctrl;
    loading = true;
    fetchError = null;
    storage.setReportRange({ from, to: rangeTo });
    if (!sessionScoped) touchCustomerContext(customerInput.trim());
    render();

    const params = new URLSearchParams({
      customer_id: customerId(),
      from,
      to: rangeTo,
      limit: '50',
    });
    if (cursor) params.set('cursor', cursor);
    if (comparePeriod && !cursor) params.set('compare', 'previous');

    const [apiRes, apiErr] = await to(api(`/api/v1/reports/${opts.endpoint}?${params.toString()}`, {
      signal: ctrl.signal,
    }));
    if (!shouldCommitAsyncResult(opGen, fetchGuard.current(), destroyed)) return;
    if (apiErr) {
      if (apiErr.name === 'AbortError') return;
      fetchError = apiErr;
    } else {
      const data = apiRes?.data;
      const batch = data?.rows ?? [];
      const [mergedRows, mergeErr] = await to(
        cursor ? mergeReportRows(rows, batch) : mergeReportRows([], batch),
      );
      if (!shouldCommitAsyncResult(opGen, fetchGuard.current(), destroyed)) return;
      if (mergeErr) {
        fetchError = mergeErr;
      } else {
        rows = mergedRows;
        freshness = data?.freshness;
        nextCursor = rows.length >= MAX_REPORT_ROWS ? '' : (data?.next_cursor ?? '');
      }
    }
    loading = false;
    if (fetchError !== lastFetchError) {
      lastFetchError = fetchError;
      if (fetchError) surfaceServiceErrorToast(fetchError);
    }
    render();
  }

  function handleSearch(e) {
    e.preventDefault();
    rows = [];
    nextCursor = '';
    fetchReport('');
  }

  function applyPreset(preset) {
    activePreset = preset.id;
    from = preset.from();
    rangeTo = preset.to();
    rangeError = validateReportRange(from, rangeTo);
    render();
  }

  function renderDatePresets() {
    return el('div', { className: 'date-presets' },
      el('span', { className: 'date-presets__label' }, 'Range'),
      REPORT_DATE_PRESETS.map((preset) =>
        el('button', {
          type: 'button',
          className: 'date-preset' + (activePreset === preset.id ? ' date-preset--active' : ''),
          onClick: () => applyPreset(preset),
        }, preset.label),
      ),
    );
  }

  function render() {
    if (destroyed) return;

    if (fetchError) {
      const view = mapServiceError(fetchError);
      if (view.kind === 'stub') {
        replaceChildren(container,
          el('div', { className: 'page-header' },
            el('h1', { className: 'page-header__title' }, opts.title),
          ),
          renderStubBanner({
            message: view.message,
            linkTo: '/reports/placements',
            linkLabel: 'Report: placements',
          }),
        );
        return;
      }
      if (isPageBlockingError(view) || view.kind === 'empty') {
        replaceChildren(container, renderErrorBlock(fetchError));
        return;
      }
    }

    const isPlacements = opts.endpoint === 'placements';
    const { visible: tableRows, truncated } = visibleReportRows(rows);

    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            renderIcon('file-spreadsheet', { size: 20, className: 'text-muted' }),
            el('h1', { className: 'page-header__title' }, opts.title),
          ),
          freshness ? renderFreshnessBadge({
            stale: freshness.stale,
            lagSeconds: freshness.ch_lag_seconds,
          }) : null,
        ),
      ),
      el('form', {
        onSubmit: handleSearch,
        className: 'mb-4',
      },
        !sessionScoped
          ? renderFormField({
            label: 'Customer ID',
            htmlFor: 'report-customer-id',
            error: customerError,
            hint: 'UUID of the customer to query',
            children: el('input', {
              id: 'report-customer-id',
              className: 'form-input',
              value: customerInput,
              onInput: (e) => {
                customerInput = e.target.value;
                customerError = validateCustomerIdField(customerInput);
              },
              required: true,
            }),
          })
          : null,
        sessionScoped && customerId()
          ? el('p', { className: 'text-muted', style: { fontSize: 13, marginBottom: 12 } },
            'Customer: ',
            el('span', { className: 'font-mono' }, customerId()),
          )
          : null,
        renderDatePresets(),
        el('label', { className: 'form-checkbox', style: { display: 'block', marginBottom: 12 } },
          el('input', {
            type: 'checkbox',
            checked: comparePeriod,
            onChange: (e) => { comparePeriod = e.target.checked; },
          }),
          ' ',
          t('report.compare', 'Compare with previous period'),
        ),
        el('div', { className: 'form-row' },
          renderFormField({
            label: 'From (ISO)',
            htmlFor: 'report-from',
            children: el('input', {
              id: 'report-from',
              className: 'form-input',
              value: from,
              onInput: (e) => {
                from = e.target.value;
                activePreset = '';
                rangeError = validateReportRange(from, rangeTo);
              },
              required: true,
            }),
          }),
          renderFormField({
            label: 'To (ISO)',
            htmlFor: 'report-to',
            error: rangeError,
            children: el('input', {
              id: 'report-to',
              className: 'form-input',
              value: rangeTo,
              onInput: (e) => {
                rangeTo = e.target.value;
                activePreset = '';
                rangeError = validateReportRange(from, rangeTo);
              },
              required: true,
            }),
          }),
          el('button', {
            type: 'submit',
            className: 'btn btn--primary',
            disabled: loading,
          },
            renderIcon('search', { size: 14 }),
            'Load',
          ),
        ),
      ),
      loading && rows.length === 0
        ? el('div', { className: 'table-wrapper elevation-raised' },
          el('table', { className: 'data-table' },
            el('tbody', null, tableSkeletonRows(8, 6)),
          ),
        )
        : null,
      rows.length > 0
        ? el('div', { className: 'table-wrapper elevation-raised' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, isPlacements ? 'Placement' : 'Keyword'),
                el('th', { scope: 'col' }, 'Campaign'),
                el('th', { scope: 'col' }, 'Impr.'),
                el('th', { scope: 'col' }, 'Clicks'),
                el('th', { scope: 'col' }, 'Conv.'),
                showFinancials ? el('th', { scope: 'col' }, 'Spend') : null,
                showFinancials ? el('th', { scope: 'col' }, 'Revenue') : null,
                showFinancials ? el('th', { scope: 'col' }, 'ROI %') : null,
                el('th', { scope: 'col' }, 'CTR %'),
                showFinancials ? el('th', { scope: 'col' }, 'CPA') : null,
                el('th', { scope: 'col' }, 'IVT %'),
                comparePeriod ? el('th', { scope: 'col' }, 'Δ spend') : null,
              ),
            ),
            el('tbody', null,
              tableRows.map((row) => {
                const spendDelta = comparePeriod && showFinancials
                  ? row.compare?.spend_micro_delta
                  : null;
                return el('tr', null,
                  el('td', { className: 'font-mono' },
                    isPlacements ? row.placement_id : row.keyword,
                  ),
                  el('td', { className: 'font-mono text-muted' }, row.campaign_id),
                  el('td', null, String(row.impressions)),
                  el('td', null, String(row.clicks)),
                  el('td', null, String(row.conversions)),
                  showFinancials
                    ? el('td', { className: 'font-mono' },
                      formatAmountMicro(row.spend_micro ?? 0),
                    )
                    : null,
                  showFinancials
                    ? el('td', { className: 'font-mono' },
                      formatAmountMicro(row.revenue_micro ?? 0),
                    )
                    : null,
                  showFinancials
                    ? el('td', null,
                      row.roi_pct != null ? row.roi_pct.toFixed(2) : '—',
                    )
                    : null,
                  el('td', null, row.ctr != null ? (row.ctr * 100).toFixed(2) : '—'),
                  showFinancials
                    ? el('td', { className: 'font-mono' },
                      row.cpa_micro != null ? formatAmountMicro(row.cpa_micro) : '—',
                    )
                    : null,
                  el('td', null, row.ivt_rate != null ? (row.ivt_rate * 100).toFixed(2) : '—'),
                  comparePeriod && spendDelta != null
                    ? el('td', { className: 'font-mono' }, formatAmountMicro(spendDelta))
                    : (comparePeriod ? el('td', null, '—') : null),
                );
              }),
            ),
          ),
        )
        : null,
      nextCursor && rows.length < MAX_REPORT_ROWS
        ? el('button', {
          type: 'button',
          className: 'btn btn--secondary btn--sm mt-4',
          disabled: loading,
          onClick: () => fetchReport(nextCursor),
        }, 'Load more')
        : null,
    );
  }

  render();

  return {
    destroy() {
      destroyed = true;
      fetchGuard.invalidate();
      fetchAbort?.abort();
    },
  };
}

/**
 * Return the default report range end timestamp.
 *
 * @returns {string}
 */
function defaultTo() {
  return new Date().toISOString();
}

/**
 * Return the default report range start timestamp seven days ago.
 *
 * @returns {string}
 */
function defaultFrom() {
  const d = new Date();
  d.setDate(d.getDate() - 7);
  return d.toISOString();
}
