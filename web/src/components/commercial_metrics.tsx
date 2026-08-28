import { formatMicro } from '../helpers/money.js';
import { FreshnessBadge } from './freshness_badge.js';
import type { MetricsBlockDTO } from '../types/metrics.js';

export type CommercialMetricsProps = {
  kpis?: MetricsBlockDTO | null;
  masked?: boolean;
};

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric-card">
      <div className="metric-card__head">
        <div className="metric-card__label">{label}</div>
      </div>
      <div className="metric-card__value font-mono">{value}</div>
    </div>
  );
}

export function CommercialMetrics({ kpis, masked = false }: CommercialMetricsProps) {
  if (!kpis) return null;

  const cards: Array<{ label: string; value: string }> = [];

  if (kpis.spend_micro != null && !masked) {
    cards.push({ label: 'Spend', value: `$${formatMicro(kpis.spend_micro)}` });
  }
  if (kpis.revenue_micro != null && kpis.revenue_micro > 0) {
    cards.push({ label: 'Revenue', value: `$${formatMicro(kpis.revenue_micro)}` });
  }
  if (kpis.profit_micro != null && kpis.profit_micro !== 0) {
    cards.push({ label: 'Profit', value: `$${formatMicro(kpis.profit_micro)}` });
  }
  if (kpis.conversions != null) {
    cards.push({ label: 'Conversions', value: String(kpis.conversions) });
  }
  if (kpis.cpa_micro != null && kpis.cpa_micro > 0 && !masked) {
    cards.push({ label: 'CPA', value: `$${formatMicro(kpis.cpa_micro)}` });
  }
  if (kpis.roi_pct != null && kpis.roi_pct !== 0) {
    cards.push({ label: 'ROI', value: `${kpis.roi_pct.toFixed(1)}%` });
  }

  if (cards.length === 0 && !kpis.freshness) return null;

  const lagSeconds = kpis.freshness?.lagSeconds ?? kpis.freshness?.ch_lag_seconds;

  return (
    <div className="grid-stats" data-testid="commercial-metrics">
      {cards.map((card) => (
        <MetricCard key={card.label} label={card.label} value={card.value} />
      ))}
      {kpis.freshness ? (
        <div className="metric-card metric-card--freshness">
          <FreshnessBadge
            stale={kpis.freshness.stale}
            lagSeconds={lagSeconds}
            sources={kpis.freshness.sources}
          />
        </div>
      ) : null}
    </div>
  );
}
