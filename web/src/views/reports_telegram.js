import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderFormField } from '../ui/form_field.js';
import { validateReportRange } from '../helpers/validators.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';

/**
 * Mount Telegram Mini Apps report view.
 *
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const user = auth.getUser();
  const sessionScoped = hasBoundCustomer(user?.role);
  let campaignInput = ctx.query.get('campaign_id') || '';
  const preset = REPORT_DATE_PRESETS[1] ?? REPORT_DATE_PRESETS[0];
  let from = ctx.query.get('from') || preset.from();
  let to = ctx.query.get('to') || preset.to();
  let loading = false;
  let data = null;
  /** @type {Error|null} */
  let error = null;

  async function load() {
    const rangeErr = validateReportRange(from, to);
    if (rangeErr) {
      error = new Error(rangeErr);
      render();
      return;
    }
    loading = true;
    error = null;
    render();

    const params = new URLSearchParams({ from, to });
    if (campaignInput.trim()) {
      params.append('campaign_id', campaignInput.trim());
    }

    const [res, err] = await to(api(`/api/v1/reports/telegram?${params.toString()}`));
    if (destroyed) return;
    loading = false;
    if (err) {
      error = err;
    } else {
      data = res?.data ?? null;
    }
    render();
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error));
      return;
    }

    const funnelItems = data?.funnel
      ? [
          { label: 'Total Clicks', val: data.clicks },
          { label: 'Total Impressions', val: data.impressions },
          { label: 'Premium Users', val: data.premium },
          { label: 'Motivated Clicks', val: data.motivated },
        ]
      : [];

    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, 'Telegram Mini Apps Analytics'),
      ),
      el('form', {
        onSubmit: (e) => {
          e.preventDefault();
          load();
        },
      },
        renderFormField({
          label: 'Campaign ID (Optional)',
          htmlFor: 'tg-campaign',
          children: el('input', {
            id: 'tg-campaign',
            className: 'form-input',
            value: campaignInput,
            placeholder: 'All campaigns',
            onInput: (e) => { campaignInput = e.target.value; },
          }),
        }),
        renderFormField({
          label: 'From',
          htmlFor: 'tg-from',
          children: el('input', { id: 'tg-from', className: 'form-input', value: from, onInput: (e) => { from = e.target.value; } }),
        }),
        renderFormField({
          label: 'To',
          htmlFor: 'tg-to',
          children: el('input', { id: 'tg-to', className: 'form-input', value: to, onInput: (e) => { to = e.target.value; } }),
        }),
        el('button', { type: 'submit', className: 'btn btn--primary', disabled: loading }, 'Query'),
      ),
      loading ? el('p', null, 'Loading report…') : null,
      !loading && data
        ? el('div', { style: { marginTop: '20px' } },
            el('h2', null, 'Attribution Funnel'),
            el('div', { style: { display: 'flex', gap: '20px', flexWrap: 'wrap', marginTop: '10px' } },
              funnelItems.map((item) =>
                el('div', {
                  style: {
                    border: '1px solid var(--border-color, #ccc)',
                    padding: '15px 25px',
                    borderRadius: '8px',
                    minWidth: '150px',
                    background: 'var(--bg-secondary, #fafafa)',
                    textAlign: 'center',
                  },
                },
                  el('div', { style: { fontSize: '14px', color: 'var(--text-muted, #666)' } }, item.label),
                  el('div', { style: { fontSize: '28px', fontWeight: 'bold', marginTop: '5px' } }, String(item.val ?? 0)),
                )
              )
            )
          )
        : null,
      !loading && !data ? el('p', null, 'Perform a query to load report.') : null,
    );
  }

  load();
  return { destroy() { destroyed = true; } };
}
