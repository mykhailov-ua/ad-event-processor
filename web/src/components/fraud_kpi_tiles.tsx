import { can } from '../helpers/permissions.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId } from '../helpers/buyer_session.js';
import { Icon } from './icon.js';
import { FraudMlHealthTile } from './fraud_ml_health.js';

export type FraudKpiGeoHint = {
  ivt_rate?: number;
};

export type FraudKpiPayload = {
  silent_reject_campaigns?: number;
  edge_blocked_fraud?: number;
  geo_hints?: FraudKpiGeoHint[];
};

export type FraudKpiTilesProps = {
  loading?: boolean;
  fraud?: FraudKpiPayload | null;
  customerId?: string | null;
};

function LinkedMetric({
  label,
  value,
  icon,
  href,
  testId,
}: {
  label: string;
  value: string;
  icon: string;
  href: string;
  testId: string;
}) {
  return (
    <a href={href} className="metric-card metric-card--link" data-testid={testId}>
      <div className="metric-card__head">
        <div className="metric-card__label">{label}</div>
        <Icon name={icon} size={16} className="text-muted" />
      </div>
      <div className="metric-card__value font-mono">{value}</div>
    </a>
  );
}

export function FraudKpiTiles({ loading = false, fraud, customerId }: FraudKpiTilesProps) {
  const qs = customerId ? `?customer_id=${encodeURIComponent(customerId)}` : '';
  const highIvt = fraud?.geo_hints
    ? fraud.geo_hints.filter((h) => Number(h.ivt_rate ?? 0) >= 0.1).length
    : null;

  const silentReject = loading ? '...' : String(fraud?.silent_reject_campaigns ?? '-');
  const blocked = loading ? '...' : String(fraud?.edge_blocked_fraud ?? '-');
  const geo = loading ? '...' : highIvt != null ? String(highIvt) : '-';

  return (
    <section className="fraud-kpi-section section-block" data-testid="fraud-kpi-tiles">
      <h2 className="subsection-title">Fraud & IVT</h2>
      <p className="text-muted text-sm">
        {customerId
          ? 'Customer-scoped signals (7d). Open dashboards for detail.'
          : 'Open customer-scoped fraud views (enter customer on destination).'}
      </p>
      <div className="grid-stats">
        <LinkedMetric
          label="Silent reject campaigns"
          value={silentReject}
          icon="alert-triangle"
          href={`/dashboards/fraud${qs}`}
          testId="fraud-kpi-silent-reject-campaigns"
        />
        <LinkedMetric
          label="Edge blocked (fraud)"
          value={blocked}
          icon="shield"
          href={`/dashboards/fraud${qs}`}
          testId="fraud-kpi-edge-blocked-fraud"
        />
        <LinkedMetric
          label="High-IVT geo hints"
          value={geo}
          icon="globe"
          href={`/reports/ivt-by-source${qs}`}
          testId="fraud-kpi-high-ivt-geo-hints"
        />
        <FraudMlHealthTile customerId={customerId} />
      </div>
    </section>
  );
}

export function BuyerFraudKpiTiles() {
  const user = auth.getUser();
  const perms = user?.permissions ?? [];
  const customerId = boundCustomerId(user);
  if (!can(perms, 'audit:read')) return null;
  return <FraudKpiTiles loading={false} fraud={null} customerId={customerId} />;
}
