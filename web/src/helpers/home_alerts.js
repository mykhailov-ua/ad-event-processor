/**
 * Build home-page alert cards from operator and buyer dashboard payloads.
 *
 * @param {{
 *   summary?: { outbox_pending?: number, drift_alert?: boolean }|null,
 *   doctor?: { checks?: Array<{ name?: string, id?: string, status?: string }>, services?: Array<{ name?: string, status?: string }> }|null,
 *   incidents?: { shards?: Array<{ shard_id?: number, ping_ok?: boolean, ping_error?: string }> }|null,
 *   meta?: { license?: { state?: string, valid_until?: string } }|null,
 *   buyerPortfolio?: { alerts?: Array<{ id: string, level: string, title: string, detail: string, route?: string }>, overspendCount?: number }|null,
 *   canOps?: boolean,
 *   buyerMode?: boolean,
 * }} input
 * @returns {Array<{ id: string, level: string, title: string, detail: string, route?: string }>}
 */
export function buildHomeAlerts(input) {
  /** @type {Array<{ id: string, level: string, title: string, detail: string, route?: string }>} */
  const alerts = [];
  const seen = new Set();

  /**
   * @param {{ id: string, level: string, title: string, detail: string, route?: string }} card
   */
  function push(card) {
    if (!card?.id || seen.has(card.id)) return;
    seen.add(card.id);
    alerts.push(card);
  }

  const license = input.meta?.license;
  const licenseState = String(license?.state ?? '').toLowerCase();
  if (licenseState && licenseState !== 'valid' && licenseState !== 'active') {
    push({
      id: 'license-state',
      level: 'critical',
      title: 'License',
      detail: `${license.state}${license.valid_until ? ` · until ${license.valid_until}` : ''}`,
      route: '/settings',
    });
  }

  if (input.canOps) {
    const summary = input.summary;
    if ((summary?.outbox_pending ?? 0) > 0) {
      push({
        id: 'outbox-pending',
        level: 'warning',
        title: 'Outbox backlog',
        detail: `${summary.outbox_pending} pending events`,
        route: '/ops',
      });
    }
    if (summary?.drift_alert) {
      push({
        id: 'drift-alert',
        level: 'critical',
        title: 'Pacing drift',
        detail: 'Campaign pacing drift detected',
        route: '/campaigns/portfolio',
      });
    }

    const chCheck = input.doctor?.checks?.find((c) =>
      String(c.name ?? c.id ?? '').toLowerCase().includes('clickhouse'),
    );
    const chService = input.doctor?.services?.find((s) =>
      String(s.name ?? '').toLowerCase().includes('clickhouse'),
    );
    const chBad = (chCheck?.status && chCheck.status !== 'ok' && chCheck.status !== 'pass')
      || (chService?.status && chService.status !== 'ok' && chService.status !== 'disabled');
    if (chBad) {
      push({
        id: 'ch-lag',
        level: 'warning',
        title: 'ClickHouse lag',
        detail: 'Analytics may be stale',
        route: '/ops',
      });
    }

    const shards = input.incidents?.shards ?? [];
    for (let i = 0; i < shards.length; i++) {
      const shard = shards[i];
      if (shard.ping_ok === false) {
        push({
          id: `shard-down-${shard.shard_id ?? i}`,
          level: 'critical',
          title: `Redis shard ${shard.shard_id ?? i} down`,
          detail: shard.ping_error ?? 'Ping failed',
          route: '/ops/shards',
        });
      }
    }
  }

  if (input.buyerMode && input.buyerPortfolio) {
    const buyerAlerts = input.buyerPortfolio.alerts ?? [];
    for (let i = 0; i < buyerAlerts.length; i++) {
      const card = buyerAlerts[i];
      push({
        id: card.id ?? `buyer-alert-${i}`,
        level: card.level ?? 'warning',
        title: card.title ?? 'Alert',
        detail: card.detail ?? '',
        route: card.route,
      });
    }
    if ((input.buyerPortfolio.overspendCount ?? 0) > 0) {
      push({
        id: 'buyer-overspend',
        level: 'warning',
        title: 'Overspend risk',
        detail: `${input.buyerPortfolio.overspendCount} campaign(s) need attention`,
        route: '/campaigns/portfolio',
      });
    }
  }

  return alerts;
}
