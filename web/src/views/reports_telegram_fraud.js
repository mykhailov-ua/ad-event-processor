import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { fetchTelegramFraud } from '../helpers/tg_report_api.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderFormField } from '../ui/form_field.js';
import { validateReportRange } from '../helpers/validators.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';

/**
 * Mount Telegram fraud report view.
 *
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
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
    const [res, err] = await to(fetchTelegramFraud({
      from,
      to,
      campaignId: campaignInput.trim() || undefined,
    }));
    if (destroyed) return;
    loading = false;
    error = err ?? null;
    data = res;
    render();
  }

  function render() {
    if (destroyed) return;
    if (error) {
      replaceChildren(container, renderErrorBlock(error));
      return;
    }
    replaceChildren(container,
      el('div', { className: 'page-header' },
        el('h1', { className: 'page-header__title' }, 'Telegram Fraud'),
      ),
      el('form', {
        onSubmit: (e) => {
          e.preventDefault();
          load();
        },
      },
        renderFormField({
          label: 'Campaign ID (Optional)',
          htmlFor: 'tg-fraud-campaign',
          children: el('input', {
            id: 'tg-fraud-campaign',
            className: 'form-input',
            value: campaignInput,
            onInput: (e) => { campaignInput = e.target.value; },
          }),
        }),
        renderFormField({
          label: 'From',
          htmlFor: 'tg-fraud-from',
          children: el('input', { id: 'tg-fraud-from', className: 'form-input', value: from, onInput: (e) => { from = e.target.value; } }),
        }),
        renderFormField({
          label: 'To',
          htmlFor: 'tg-fraud-to',
          children: el('input', { id: 'tg-fraud-to', className: 'form-input', value: to, onInput: (e) => { to = e.target.value; } }),
        }),
        el('button', { type: 'submit', className: 'btn btn--primary', disabled: loading }, 'Query'),
      ),
      loading ? el('p', null, 'Loading…') : null,
      !loading && data
        ? el('div', { style: { marginTop: '20px', display: 'flex', gap: '20px' } },
            el('div', { className: 'stat-card' }, el('div', null, 'Blocked clicks'), el('strong', null, String(data.blocked_clicks ?? 0))),
            el('div', { className: 'stat-card' }, el('div', null, 'Shadow clicks'), el('strong', null, String(data.shadow_clicks ?? 0))),
          )
        : null,
    );
  }

  load();
  return { destroy() { destroyed = true; } };
}
