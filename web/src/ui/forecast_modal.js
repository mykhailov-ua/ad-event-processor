import { el } from '../lib/dom.js';
import { mountModal } from './modal.js';

/**
 * Open campaign forecast modal with P50/P90 and retry on 503.
 *
 * @param {{ campaignId: string, customerId?: string, budgetMicro?: number, startAt: string, endAt: string }} opts
 * @returns {Promise<void>}
 */
export async function openForecastModal(opts) {
  let countdown = 0;
  /** @type {object|null} */
  let forecast = null;
  /** @type {string|null} */
  let error = null;

  const body = el('div', null, el('p', null, 'Loading forecast…'));
  let modal = mountModal({
    title: 'Campaign forecast',
    body: [body],
    onClose: () => modal.destroy(),
  });

  async function load() {
    error = null;
    body.replaceChildren(el('p', null, countdown > 0 ? `Retrying in ${countdown}s…` : 'Loading forecast…'));
    const payload = {
      budget_limit_micro: opts.budgetMicro ?? 0,
      start_at: opts.startAt,
      end_at: opts.endAt,
      pacing_mode: 'even',
      timezone: 'UTC',
    };
    if (opts.customerId) payload.customer_id = opts.customerId;
    try {
      const { api } = await import('../helpers/api_client.js');
      const res = await api('/api/v1/forecast/campaign', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
      forecast = res?.data ?? null;
      renderBody();
    } catch (err) {
      const retryAfter = Number(err?.retryAfter ?? 0);
      if (err?.status === 503 && retryAfter > 0) {
        countdown = retryAfter;
        const tick = setInterval(() => {
          countdown -= 1;
          if (countdown <= 0) {
            clearInterval(tick);
            load();
          } else {
            body.replaceChildren(el('p', null, `Service busy — retry in ${countdown}s`));
          }
        }, 1000);
        return;
      }
      error = err?.message ?? 'Forecast failed';
      body.replaceChildren(el('p', null, error));
    }
  }

  function renderBody() {
    if (!forecast) {
      body.replaceChildren(el('p', null, 'No forecast data.'));
      return;
    }
    body.replaceChildren(
      el('dl', null,
        el('dt', null, 'Impressions P50'),
        el('dd', { className: 'font-mono' }, String(forecast.impressions_p50 ?? '—')),
        el('dt', null, 'Impressions P90'),
        el('dd', { className: 'font-mono' }, String(forecast.impressions_p90 ?? '—')),
        el('dt', null, 'Low confidence'),
        el('dd', null, forecast.low_confidence ? 'Yes' : 'No'),
      ),
      forecast.advisory?.message
        ? el('p', { className: 'text-muted' }, forecast.advisory.message)
        : null,
      Array.isArray(forecast.spend_curve) && forecast.spend_curve.length > 0
        ? el('pre', { style: { fontSize: 12, maxHeight: 200, overflow: 'auto' } },
          JSON.stringify(forecast.spend_curve.slice(0, 12), null, 2),
        )
        : null,
    );
  }

  load();
}
