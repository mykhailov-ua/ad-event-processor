import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderFormField } from '../ui/form_field.js';
import { validateCustomerIdField, validateReportRange } from '../helpers/validators.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { renderFreshnessBadge } from '../ui/freshness_badge.js';
import { tenantReportQueryString } from '../helpers/tenant_url.js';
import { formatMoney } from '../helpers/money.js';
import { renderAlertBanner } from '../ui/alert_banner.js';

/**
 * Mount geo ROI report view.
 *
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  let customerInput = ctx.query.get('customer_id') || (sessionScoped ? boundCustomerId(user) : '');
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];
  let from = ctx.query.get('from') || preset.from();
  let to = ctx.query.get('to') || preset.to();
  let loading = false;
  /** @type {object[]} */
  let rows = [];
  let freshness = null;
  /** @type {Error|null} */
  let error = null;
  let validationError = null;

  async function load() {
    const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();
    const rangeErr = validateReportRange(from, to);
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
    const params = new URLSearchParams({ customer_id: cid, from, to, limit: '50' });
    const [res, err] = await to(api(`/api/v1/reports/geo-roi?${params.toString()}`));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
    } else {
      rows = res?.data?.rows ?? [];
      freshness = res?.data?.freshness ?? null;
      if (!sessionScoped && cid) {
        const qs = tenantReportQueryString({ customer_id: cid, from, to });
        window.history.replaceState(null, '', `/reports/geo-roi?${qs}`);
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
        el('h1', { className: 'page-header__title' }, 'Geo ROI'),
        freshness ? renderFreshnessBadge({ stale: freshness.stale, lagSeconds: freshness.ch_lag_seconds }) : null,
      ),
      el('form', {
        onSubmit: (e) => {
          e.preventDefault();
          load();
        },
      },
        !sessionScoped
          ? renderFormField({
            label: 'Customer ID',
            htmlFor: 'geo-customer',
            children: el('input', {
              id: 'geo-customer',
              className: 'form-input',
              placeholder: 'Customer UUID…',
              value: customerInput,
              onInput: (e) => { customerInput = e.target.value; },
            }),
          })
          : null,
        renderFormField({
          label: 'From',
          htmlFor: 'geo-from',
          children: el('input', { id: 'geo-from', className: 'form-input', value: from, onInput: (e) => { from = e.target.value; } }),
        }),
        renderFormField({
          label: 'To',
          htmlFor: 'geo-to',
          children: el('input', { id: 'geo-to', className: 'form-input', value: to, onInput: (e) => { to = e.target.value; } }),
        }),
        el('button', { type: 'submit', className: 'btn btn--primary', disabled: loading }, 'Load'),
      ),
      validationError ? renderAlertBanner({ variant: 'error', message: validationError }) : null,
      !cid && !sessionScoped
        ? renderAlertBanner({ variant: 'info', message: 'Enter a customer UUID to load report data.' })
        : null,
      loading ? el('p', { className: 'text-muted mt-4' }, 'Loading…') : null,
      rows.length > 0
        ? el('div', { className: 'table-wrapper elevation-raised mt-4' },
          el('table', { className: 'data-table' },
            el('thead', null,
              el('tr', null,
                el('th', { scope: 'col' }, 'Country'),
                el('th', { scope: 'col' }, 'Clicks'),
                el('th', { scope: 'col' }, 'IVT %'),
                el('th', { scope: 'col' }, 'Spend'),
                el('th', { scope: 'col' }, 'ROI %'),
              ),
            ),
            el('tbody', null,
              rows.map((row) => el('tr', null,
                el('td', null, row.country ?? '—'),
                el('td', null, String(row.clicks ?? 0)),
                el('td', null, row.ivt_rate != null ? `${(row.ivt_rate * 100).toFixed(2)}%` : '—'),
                el('td', null, formatMoney(row.spend_micro)),
                el('td', null, row.roi_pct != null ? `${row.roi_pct.toFixed(2)}%` : '—'),
              )),
            ),
          ),
        )
        : (cid && !loading ? el('p', { className: 'text-muted mt-4' }, 'No rows.') : null),
    );
  }

  load();
  return { destroy() { destroyed = true; } };
}
