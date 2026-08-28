import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildHomeAlerts,
  OUTBOX_PENDING_WARN_THRESHOLD,
  LOW_BUDGET_UTIL_PCT,
} from './home_alerts.js';

describe('buildHomeAlerts', () => {
  it('merges ops and buyer alerts without duplicates', () => {
    const alerts = buildHomeAlerts({
      canOps: true,
      buyerMode: true,
      summary: {
        outbox_pending: OUTBOX_PENDING_WARN_THRESHOLD + 10,
        drift_alert: true,
        emergency_breaker: 'open',
      },
      meta: { license: { state: 'expired' } },
      doctor: {
        checks: [
          { id: 'license', status: 'fail', message: 'expired JWT' },
          { id: 'slotmap', status: 'warn', message: 'version drift' },
        ],
      },
      incidents: {
        emergency_breaker: 'open',
        breaker_states: { redis_shard_2: 'open' },
        shards: [
          { shard_id: 1, ping_ok: false, ping_error: 'timeout' },
          { shard_id: 2, ping_ok: true, config_version_synced: false },
        ],
      },
      buyerPortfolio: {
        overspendCount: 2,
        campaigns: [{ id: 'c1', status: 'ACTIVE', utilization_pct: LOW_BUDGET_UTIL_PCT + 5 }],
        alerts: [{ id: 'buyer-custom', level: 'warning', title: 'Budget', detail: 'Near limit' }],
      },
    });
    const ids = alerts.map((a) => a.id);
    assert.ok(ids.includes('outbox-pending'));
    assert.ok(ids.includes('license-state'));
    assert.ok(ids.includes('doctor-license'));
    assert.ok(ids.includes('doctor-slotmap'));
    assert.ok(ids.includes('emergency-breaker'));
    assert.ok(ids.includes('breaker-open-redis_shard_2'));
    assert.ok(ids.includes('shard-down-1'));
    assert.ok(ids.includes('shard-circuit-2'));
    assert.ok(ids.includes('buyer-low-budget'));
    assert.ok(ids.includes('buyer-overspend'));
    assert.equal(new Set(ids).size, ids.length);
  });

  it('ignores small outbox backlog below threshold', () => {
    const alerts = buildHomeAlerts({
      canOps: true,
      summary: { outbox_pending: 3 },
    });
    assert.ok(!alerts.some((a) => a.id === 'outbox-pending'));
  });
});
