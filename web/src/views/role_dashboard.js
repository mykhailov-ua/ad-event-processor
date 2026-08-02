import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderCommercialMetrics } from '../ui/commercial_metrics.js';
import { renderFormField } from '../ui/form_field.js';
import { validateCustomerIdField } from '../helpers/validators.js';
import { formatAmountMicro } from '../helpers/money.js';
import { t } from '../helpers/i18n.js';

/**
 * Mount a role-specific dashboard (adops, cfo, accountant, fraud).
 *
 * @param {HTMLElement} container
 * @param {{ params: { role: string } }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const role = ctx.params.role;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  let customerInput = sessionScoped ? boundCustomerId(user) : '';
  let customerError = null;
  let loading = false;
  /** @type {object|null} */
  let data = null;
  /** @type {Error|null} */
  let blockError = null;

  const titles = {
    adops: 'AdOps dashboard',
    cfo: 'CFO dashboard',
    accountant: 'Accountant dashboard',
    fraud: 'Fraud dashboard',
  };
  const endpoints = {
    adops: '/api/v1/dashboards/adops',
    cfo: '/api/v1/dashboards/cfo',
    accountant: '/api/v1/dashboards/accountant',
    fraud: '/api/v1/dashboards/fraud',
  };

  async function load() {
    customerError = sessionScoped ? null : validateCustomerIdField(customerInput);
    const cid = sessionScoped ? boundCustomerId(user) : customerInput.trim();
    if (!cid || customerError) {
      render();
      return;
    }
    loading = true;
    blockError = null;
    render();
    const path = endpoints[role];
    if (!path) {
      blockError = new Error('Unknown dashboard role');
      loading = false;
      render();
      return;
    }
    const params = new URLSearchParams({ customer_id: cid });
    const [res, err] = await to(api(`${path}?${params.toString()}`));
    if (destroyed) return;
    loading = false;
    if (err) {
      blockError = err;
    } else {
      data = res?.data ?? null;
    }
    render();
  }

  function renderAdOps() {
    if (!data) return el('p', null, t('status.loading'));
    const kpis = renderCommercialMetrics(data.kpis, { masked: false });
    const campaigns = Array.isArray(data.campaigns) ? data.campaigns : [];
    const worst = Array.isArray(data.worst_sources) ? data.worst_sources : [];
    return el('div', null,
      kpis,
      el('h3', null, 'Campaigns'),
      campaigns.length === 0
        ? el('p', null, 'No campaigns.')
        : el('table', { className: 'data-table' },
          el('thead', null,
            el('tr', null,
              el('th', { scope: 'col' }, 'Campaign'),
              el('th', { scope: 'col' }, 'Util %'),
              el('th', { scope: 'col' }, 'Drift %'),
              el('th', { scope: 'col' }, 'Spend'),
            ),
          ),
          el('tbody', null,
            campaigns.map((c) => el('tr', null,
              el('td', null, el('a', { href: `/campaigns/${c.id}` }, c.name ?? c.id)),
              el('td', null, c.utilization_pct?.toFixed?.(1) ?? '—'),
              el('td', null, c.pacing_drift_pct?.toFixed?.(1) ?? '—'),
              el('td', { className: 'font-mono' }, formatAmountMicro(c.spend_micro ?? 0)),
            )),
          ),
        ),
      el('h3', null, 'Worst sources'),
      worst.length === 0
        ? el('p', null, 'No quality flags.')
        : el('ul', null,
          worst.map((s) => el('li', null,
            `${s.sub1 ?? s.campaign_id} — IVT ${((s.ivt_rate ?? 0) * 100).toFixed(1)}%`,
          )),
        ),
    );
  }

  function renderCFO() {
    if (!data) return el('p', null, t('status.loading'));
    return el('dl', null,
      el('dt', null, 'Billed'),
      el('dd', { className: 'font-mono' }, formatAmountMicro(data.billed_micro ?? 0)),
      el('dt', null, 'AR aging'),
      el('dd', { className: 'font-mono' }, formatAmountMicro(data.ar_aging_micro ?? 0)),
      el('dt', null, 'Dispute exposure'),
      el('dd', { className: 'font-mono' }, formatAmountMicro(data.dispute_exposure_micro ?? 0)),
    );
  }

  function renderAccountant() {
    if (!data) return el('p', null, t('status.loading'));
    const close = data.close ?? {};
    return el('div', null,
      el('dl', null,
        el('dt', null, 'Billing month'),
        el('dd', null, close.billing_month ?? '—'),
        el('dt', null, 'Invariant OK'),
        el('dd', null, close.invariant_ok ? 'Yes' : 'No'),
        el('dt', null, 'Tax country'),
        el('dd', null, data.tax_country ?? '—'),
        el('dt', null, 'Tax scheme'),
        el('dd', null, data.tax_scheme ?? data.tax_vat_id ?? '—'),
      ),
      Array.isArray(data.export_jobs) && data.export_jobs.length > 0
        ? el('ul', null, data.export_jobs.map((j) => el('li', null, `${j.id}: ${j.status}`)))
        : el('p', { className: 'text-muted' }, 'No export jobs.'),
    );
  }

  function renderFraud() {
    if (!data) return el('p', null, t('status.loading'));
    return el('dl', null,
      el('dt', null, 'Ghost IVT campaigns'),
      el('dd', null, String(data.ghost_ivt_campaigns ?? 0)),
      el('dt', null, 'Labels pending (7d)'),
      el('dd', null, String(data.labels_pending ?? 0)),
      el('dt', null, 'Edge blocked (fraud tier)'),
      el('dd', null, String(data.edge_blocked_fraud ?? 0)),
    );
  }

  const renderers = {
    adops: renderAdOps,
    cfo: renderCFO,
    accountant: renderAccountant,
    fraud: renderFraud,
  };

  function render() {
    if (destroyed) return;
    if (blockError) {
      replaceChildren(container, renderErrorBlock(blockError));
      return;
    }
    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, titles[role] ?? 'Dashboard'),
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
            htmlFor: 'role-dash-customer',
            error: customerError,
            children: el('input', {
              id: 'role-dash-customer',
              className: 'form-input',
              value: customerInput,
              onInput: (e) => {
                customerInput = e.target.value;
                customerError = validateCustomerIdField(customerInput);
              },
            }),
          })
          : null,
        el('button', { type: 'submit', className: 'btn btn--primary', disabled: loading }, t('action.load')),
      ),
      loading ? el('p', null, t('status.loading')) : (renderers[role]?.() ?? el('p', null, 'Unknown role')),
    );
  }

  load();
  return { destroy() { destroyed = true; } };
}
