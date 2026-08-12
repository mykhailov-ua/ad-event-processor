import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import type { DataFreshness, ReportEnvelope, ReportRow } from '../types/api/report.js';
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
import { renderEmptyState } from '../ui/data_table.js';
import { renderButton } from '../ui/button.js';

/**
 * Mount traffic-sources report view.
 *
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  let customerInput = ctx.query.get('customer_id') || (sessionScoped ? boundCustomerId(user) : '');
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];
  let from = ctx.query.get('from') || preset.from();
  let rangeTo = ctx.query.get('to') || preset.to();
  let loading = false;
  /** @type {ReportRow[]} */
  let rows: ReportRow[] = [];
  let freshness: DataFreshness | null = null;
  /** @type {Error|null} */
  let error: any = null;
  let validationError: any = null;

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
    const params = new URLSearchParams({ customer_id: cid, from, to: rangeTo, limit: '50' });
    const [res, err] = await to(api(`/api/v1/reports/traffic-sources?${params.toString()}`));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
    } else {
      const data = (res?.data as ReportEnvelope | null) ?? null;
      rows = data?.rows ?? [];
      freshness = data?.freshness ?? null;
      if (!sessionScoped && cid) {
        const qs = tenantReportQueryString({ customer_id: cid, from, to: rangeTo });
        window.history.replaceState(null, '', `/reports/traffic-sources?${qs}`);
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
        el('h1', { className: 'page-header__title' }, 'Traffic sources'),
        freshness ? renderFreshnessBadge({ stale: freshness.stale, lagSeconds: freshness.ch_lag_seconds }) : null,
      ),
      el('form', {
        onSubmit: (e: Event) => {
          e.preventDefault();
          load();
        },
      },
        !sessionScoped
          ? renderFormField({
            label: 'Customer ID',
            htmlFor: 'traffic-customer',
            children: el('input', {
              id: 'traffic-customer',
              className: 'form-input',
              placeholder: 'Customer UUID…',
              value: customerInput,
              onInput: (e: Event) => { customerInput = eventTargetValue(e); },
            }),
          })
          : null,
        renderFormField({
          label: 'From',
          htmlFor: 'traffic-from',
          children: el('input', { id: 'traffic-from', className: 'form-input', value: from, onInput: (e: Event) => { from = eventTargetValue(e); } }),
        }),
        renderFormField({
          label: 'To',
          htmlFor: 'traffic-to',
          children: el('input', { id: 'traffic-to', className: 'form-input', value: rangeTo, onInput: (e: Event) => { rangeTo = eventTargetValue(e); } }),
        }),
        renderButton({ label: 'Load', variant: 'primary', type: 'submit', loading, disabled: loading }),
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
                el('th', { scope: 'col' }, 'Channel'),
                el('th', { scope: 'col' }, 'Impr.'),
                el('th', { scope: 'col' }, 'Clicks'),
                el('th', { scope: 'col' }, 'Spend'),
                el('th', { scope: 'col' }, 'ROI %'),
              ),
            ),
            el('tbody', null,
              rows.map((row: any) => el('tr', null,
                el('td', null, row.channel ?? '—'),
                el('td', null, String(row.impressions ?? 0)),
                el('td', null, String(row.clicks ?? 0)),
                el('td', null, formatMoney(row.spend_micro)),
                el('td', null, row.roi_pct != null ? `${row.roi_pct.toFixed(2)}%` : '—'),
              )),
            ),
          ),
        )
        : (cid && !loading
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
