export type PacingHealthProps = {
  status?: string;
  pacingMode?: string;
  impressions7d?: number;
  deliveryPct?: number | null;
};

/**
 * Buyer pacing health summary (no financial fields).
 */
export function PacingHealth({
  status = '',
  pacingMode = 'even',
  impressions7d = 0,
  deliveryPct = null,
}: PacingHealthProps) {
  const statusUpper = String(status).toUpperCase();
  const impr = Number(impressions7d ?? 0);

  let health = 'on-track';
  let detail = 'Delivery within expected range for the period.';
  if (statusUpper === 'PAUSED') {
    health = 'paused';
    detail = 'Campaign is paused; no delivery expected.';
  } else if (pacingMode !== 'even' && pacingMode !== '') {
    health = 'drift';
    detail = `Non-even pacing mode (${pacingMode}); monitor delivery closely.`;
  } else if (impr === 0 && statusUpper === 'ACTIVE') {
    health = 'underspend';
    detail = 'No impressions in the last 7 days.';
  }

  return (
    <section data-testid="pacing-panel">
      <h3>Pacing health</h3>
      <dl>
        <dt>Status</dt>
        <dd>{health}</dd>
        <dt>Pacing mode</dt>
        <dd>{pacingMode}</dd>
        <dt>Impressions (7d)</dt>
        <dd>{String(impr)}</dd>
        {deliveryPct != null ? (
          <>
            <dt>Delivery vs expected</dt>
            <dd>{`${deliveryPct}%`}</dd>
          </>
        ) : null}
      </dl>
      <p>{detail}</p>
      <p>
        <a href="/campaigns/portfolio">Portfolio (pacing drift)</a>
        {' · '}
        <a href="/reports/placements">Placements</a>
      </p>
    </section>
  );
}
