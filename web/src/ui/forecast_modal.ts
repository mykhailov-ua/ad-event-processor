import { el } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { ApiError } from '../helpers/api_client.js';
import { mountModal, type ModalHandle } from './modal.js';

export type ForecastModalOpts = {
  campaignId: string;
  customerId?: string;
  budgetMicro?: number;
  startAt: string;
  endAt: string;
};

type ForecastAdvisory = {
  message?: string;
};

type ForecastPayload = {
  impressions_p50?: number;
  impressions_p90?: number;
  low_confidence?: boolean;
  advisory?: ForecastAdvisory;
  spend_curve?: Array<{ hour: string; spend_micro?: number; impressions?: number }>;
};

type ChartHandle = {
  destroy: () => void;
};

/**
 * Open campaign forecast modal with P50/P90 and retry on 503.
 */
export async function openForecastModal(opts: ForecastModalOpts): Promise<void> {
  let countdown = 0;
  let forecast: ForecastPayload | null = null;
  let error: string | null = null;
  let curveChart: ChartHandle | null = null;
  let retryTimer: ReturnType<typeof setInterval> | null = null;

  const body = el('div', null, el('p', null, 'Loading forecast…'));

  function clearRetryTimer(): void {
    if (retryTimer) {
      clearInterval(retryTimer);
      retryTimer = null;
    }
  }

  const modal: ModalHandle = mountModal({
    title: 'Campaign forecast',
    body: [body],
    onClose: () => {
      clearRetryTimer();
      curveChart?.destroy();
      modal.destroy();
    },
  });

  async function load(): Promise<void> {
    error = null;
    body.replaceChildren(el('p', null, countdown > 0 ? `Retrying in ${countdown}s…` : 'Loading forecast…'));
    const payload: Record<string, unknown> = {
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
      const retryAfter = err instanceof ApiError
        ? Number(err.responseHeaders?.get('Retry-After') ?? (err as Error & { retryAfter?: number }).retryAfter ?? 0)
        : Number((err as Error & { retryAfter?: number }).retryAfter ?? 0);
      const status = err instanceof ApiError ? err.status : (err as Error & { status?: number }).status;
      if (status === 503 && retryAfter > 0) {
        countdown = retryAfter;
        clearRetryTimer();
        retryTimer = setInterval(() => {
          countdown -= 1;
          if (countdown <= 0) {
            clearRetryTimer();
            void load();
          } else {
            body.replaceChildren(el('p', null, `Service busy — retry in ${countdown}s`));
          }
        }, 1000);
        return;
      }
      error = err.message ?? 'Forecast failed';
      body.replaceChildren(el('p', null, error));
      return;
    }
    forecast = (res?.data as ForecastPayload | null) ?? null;
    renderBody();
  }

  function renderBody(): void {
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
      const curve = forecast.spend_curve;
      void import('../charts/campaign_stats_chart.js').then((mod) => {
        curveChart = mod.mountSpendCurveChart(chartMount, curve, 'impressions');
      });
    }
  }

  void load();
}
