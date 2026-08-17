import type { BillingForecastDTO } from '../types/api/billing.js';
import { formatAmountMicro } from '../helpers/money.js';
import {
  isPageBlockingError,
  mapServiceError,
} from '../helpers/service_error.js';
import { useResource } from '../hooks/use_resource.js';
import { AlertBanner } from './alert_banner.js';

export type BillingForecastWidgetProps = {
  customerId: string;
};

/**
 * Customer billing forecast from GET /customers/{id}/billing/forecast.
 */
export function BillingForecastWidget({ customerId }: BillingForecastWidgetProps) {
  const forecastUrl = customerId
    ? `/api/v1/customers/${encodeURIComponent(customerId)}/billing/forecast`
    : null;
  const { data, loading, error } = useResource<BillingForecastDTO>(forecastUrl);

  if (!customerId) return null;

  if (loading) {
    return <p className="text-muted text-sm" data-testid="billing-forecast-loading">Loading forecast…</p>;
  }

  if (error) {
    const view = mapServiceError(error);
    const message = view.message || 'Forecast unavailable.';
    if (isPageBlockingError(view)) {
      return (
        <AlertBanner variant="warning" message={message} />
      );
    }
    return (
      <p className="text-muted text-sm" data-testid="billing-forecast-empty">
        {message}
      </p>
    );
  }

  if (!data) {
    return (
      <p className="text-muted text-sm" data-testid="billing-forecast-empty">
        No forecast data for this period.
      </p>
    );
  }

  const hasNumbers = (data.ledger_mtd_micro ?? 0) > 0
    || (data.projected_month_end_micro ?? 0) > 0
    || (data.ledger_run_rate_micro_per_day ?? 0) > 0;

  if (!hasNumbers && !data.low_confidence && !data.ch_unavailable) {
    return (
      <p className="text-muted text-sm" data-testid="billing-forecast-empty">
        No spend recorded this month yet.
      </p>
    );
  }

  return (
    <section className="section-card stack" data-testid="billing-forecast-widget">
      <h2 className="subsection-title">
        Billing forecast
        {data.month ? ` (${data.month})` : ''}
      </h2>
      {data.ch_unavailable ? (
        <AlertBanner
          variant="info"
          message="ClickHouse telemetry unavailable; projection uses ledger run rate only."
        />
      ) : null}
      {data.low_confidence ? (
        <AlertBanner variant="info" message="Low confidence — limited billing history." />
      ) : null}
      <div className="grid-stats">
        <div className="metric-card">
          <div className="metric-card__label">Ledger MTD</div>
          <div className="metric-card__value font-mono" data-testid="forecast-mtd">
            {formatAmountMicro(data.ledger_mtd_micro ?? 0)}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Run rate / day</div>
          <div className="metric-card__value font-mono" data-testid="forecast-run-rate">
            {formatAmountMicro(data.ledger_run_rate_micro_per_day ?? 0)}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Projected month-end</div>
          <div className="metric-card__value font-mono" data-testid="forecast-projected">
            {formatAmountMicro(data.projected_month_end_micro ?? 0)}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Days remaining</div>
          <div className="metric-card__value font-mono" data-testid="forecast-days-remaining">
            {String(data.days_remaining ?? '—')}
          </div>
        </div>
      </div>
    </section>
  );
}
