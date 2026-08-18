export const OUTBOX_PENDING_WARN_THRESHOLD = 50;

export const OUTBOX_PENDING_CRITICAL_THRESHOLD = 500;

export const LOW_BUDGET_UTIL_PCT = 85;

export type HomeAlertCard = {
  id: string;
  level: string;
  title: string;
  detail: string;
  route?: string;
};

export type HomeAlertService = {
  name?: string;
  id?: string;
  status?: string;
  detail?: string;
};

export type HomeAlertInput = {
  summary?: {
    outbox_pending?: number;
    drift_alert?: boolean;
    emergency_breaker?: string;
    services?: HomeAlertService[];
  } | null;
  doctor?: {
    checks?: Array<{ id?: string; name?: string; status?: string; message?: string }>;
    services?: HomeAlertService[];
  } | null;
  incidents?: {
    emergency_breaker?: string;
    breaker_states?: Record<string, string>;
    shards?: Array<{
      shard_id?: number;
      ping_ok?: boolean;
      ping_error?: string;
      config_version_synced?: boolean;
    }>;
  } | null;
  meta?: { license?: { state?: string; valid_until?: string } } | null;
  buyerPortfolio?: {
    alerts?: HomeAlertCard[];
    overspendCount?: number;
    campaigns?: Array<{
      id?: string;
      name?: string;
      status?: string;
      utilization_pct?: number;
    }>;
  } | null;
  canOps?: boolean;
  buyerMode?: boolean;
};

export function buildHomeAlerts(input: HomeAlertInput): HomeAlertCard[] {
  const alerts: HomeAlertCard[] = [];
  const seen = new Set<string>();

  function push(card: HomeAlertCard): void {
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
      detail: `${license!.state}${license!.valid_until ? ` · until ${license!.valid_until}` : ''}`,
      route: '/settings',
    });
  }

  if (input.canOps) {
    const summary = input.summary;
    const pending = summary?.outbox_pending ?? 0;
    if (pending >= OUTBOX_PENDING_CRITICAL_THRESHOLD) {
      push({
        id: 'outbox-pending',
        level: 'critical',
        title: 'Outbox backlog',
        detail: `${pending} pending events (>${OUTBOX_PENDING_CRITICAL_THRESHOLD})`,
        route: '/ops',
      });
    } else if (pending > OUTBOX_PENDING_WARN_THRESHOLD) {
      push({
        id: 'outbox-pending',
        level: 'warning',
        title: 'Outbox backlog',
        detail: `${pending} pending events`,
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

    const breaker = String(
      summary?.emergency_breaker ?? input.incidents?.emergency_breaker ?? ''
    ).toLowerCase();
    if (breaker === 'open') {
      push({
        id: 'emergency-breaker',
        level: 'critical',
        title: 'Emergency breaker',
        detail: 'Shard emergency breaker is open',
        route: '/ops/shards',
      });
    }

    const breakerStates = input.incidents?.breaker_states ?? {};
    const openBreakers = Object.entries(breakerStates).filter(
      ([, v]) => String(v).toLowerCase() === 'open'
    );
    for (let i = 0; i < openBreakers.length; i++) {
      const [name] = openBreakers[i];
      push({
        id: `breaker-open-${name}`,
        level: 'critical',
        title: `Circuit open: ${name}`,
        detail: 'Breaker rejected hot-path traffic',
        route: '/ops',
      });
    }

    const checks = input.doctor?.checks ?? [];
    for (let i = 0; i < checks.length; i++) {
      const check = checks[i];
      const checkId = String(check.id ?? check.name ?? '').toLowerCase();
      const status = String(check.status ?? '').toLowerCase();
      const message = String(check.message ?? '');

      if (checkId === 'license' && status !== 'pass' && status !== 'ok' && status !== 'skip') {
        push({
          id: 'doctor-license',
          level: status === 'fail' ? 'critical' : 'warning',
          title: 'License probe',
          detail: message || 'License check failed',
          route: '/settings',
        });
      }

      if (checkId === 'slotmap' && status !== 'pass' && status !== 'ok' && status !== 'skip') {
        push({
          id: 'doctor-slotmap',
          level: status === 'fail' ? 'critical' : 'warning',
          title: 'Slot map drift',
          detail: message || 'Postgres vs edge slot map mismatch',
          route: '/ops/shards',
        });
      }

      if (
        message.toLowerCase().includes('registry_stale') ||
        (checkId.includes('registry') && status === 'fail')
      ) {
        push({
          id: 'registry-stale',
          level: 'critical',
          title: 'Campaign registry stale',
          detail: message || 'Tracker registry stale-serve active',
          route: '/ops',
        });
      }
    }

    const trackerSvc =
      input.doctor?.services?.find((s) =>
        String(s.name ?? '')
          .toLowerCase()
          .includes('tracker')
      ) ??
      input.summary?.services?.find((s) =>
        String(s.name ?? s.id ?? '')
          .toLowerCase()
          .includes('tracker')
      );
    const trackerDetail = String(trackerSvc?.detail ?? '').toLowerCase();
    if (trackerDetail.includes('registry_stale') || trackerDetail.includes('registry stale')) {
      push({
        id: 'registry-stale-tracker',
        level: 'critical',
        title: 'Campaign registry stale',
        detail: trackerSvc?.detail ?? 'Tracker reports stale registry',
        route: '/ops',
      });
    }

    const chCheck = checks.find((c) =>
      String(c.id ?? c.name ?? '')
        .toLowerCase()
        .includes('clickhouse')
    );
    const chService = input.doctor?.services?.find((s) =>
      String(s.name ?? '')
        .toLowerCase()
        .includes('clickhouse')
    );
    const chBad =
      (chCheck?.status && chCheck.status !== 'ok' && chCheck.status !== 'pass') ||
      (chService?.status && chService.status !== 'ok' && chService.status !== 'disabled');
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
      } else if (shard.config_version_synced === false) {
        push({
          id: `shard-circuit-${shard.shard_id ?? i}`,
          level: 'warning',
          title: `Shard ${shard.shard_id ?? i} config lag`,
          detail: 'Redis config version not synced — circuit risk',
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

    const campaigns = input.buyerPortfolio.campaigns ?? [];
    let lowBudgetCount = 0;
    for (let i = 0; i < campaigns.length; i++) {
      const c = campaigns[i];
      if (String(c.status ?? '').toUpperCase() !== 'ACTIVE') continue;
      const util = Number(c.utilization_pct ?? 0);
      if (util >= LOW_BUDGET_UTIL_PCT) lowBudgetCount += 1;
    }
    if (lowBudgetCount > 0) {
      push({
        id: 'buyer-low-budget',
        level: 'warning',
        title: 'Low budget headroom',
        detail: `${lowBudgetCount} active campaign(s) above ${LOW_BUDGET_UTIL_PCT}% utilization`,
        route: '/campaigns/portfolio',
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
