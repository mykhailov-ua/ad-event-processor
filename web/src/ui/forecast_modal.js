import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
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
  /** @type {{ destroy: () => void }|null} */
  let curveChart = null;
  /** @type {ReturnType<typeof setInterval>|null} */
  let retryTimer = null;

  const body = el('div', null, el('p', null, 'Loading forecast…'));

  function clearRetryTimer() {
    if (retryTimer) {
      clearInterval(retryTimer);
      retryTimer = null;
    }
  }

  let modal = mountModal({
    title: 'Campaign forecast',
    body: [body],
    onClose: () => {
      clearRetryTimer();
      curveChart?.destroy();
      modal.destroy();
    },
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
    const { api } = await import('../helpers/api_client.js');
    const [res, err] = await to(api('/api/v1/forecast/campaign', {
      method: 'POST',
      body: JSON.stringify(payload),
    }));
    if (err) {
      const retryAfter = Number(err?.retryAfter ?? 0);
      if (err?.status === 503 && retryAfter > 0) {
        countdown = retryAfter;
        clearRetryTimer();
        retryTimer = setInterval(() => {
          countdown -= 1;
          if (countdown <= 0) {
            clearRetryTimer();
            load();
          } else {
            body.replaceChildren(el('p', null, `Service busy — retry in ${countdown}s`));
          }
        }, 1000);
        return;
      }
      error = err?.message ?? 'Forecast failed';
      body.replaceChildren(el('p', null, error));
      return;
    }
    forecast = res?.data ?? null;
    renderBody();
  }

  function renderBody() {
    curveChart?.destroy();
    curveChart = null;

    if (!forecast) {
      body.replaceChildren(el('p', null, 'No forecast data.'));
      return;
    }

    const chartMount = el('div');
    const children = [
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
    ];

    if (Array.isArray(forecast.spend_curve) && forecast.spend_curve.length > 0) {
      children.push(
        el('h3', { className: 'subsection-title' }, 'Projected delivery'),
        chartMount,
      );
    }

    body.replaceChildren(el('div', { className: 'stack' }, ...children));

    if (Array.isArray(forecast.spend_curve) && forecast.spend_curve.length > 0) {
      import('../charts/campaign_stats_chart.js').then((mod) => {
        curveChart = mod.mountSpendCurveChart(chartMount, forecast.spend_curve, 'impressions');
      });
    }
  }

  load();
}
