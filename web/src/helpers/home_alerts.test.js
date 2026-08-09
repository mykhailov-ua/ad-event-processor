import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { buildHomeAlerts } from './home_alerts.js';

describe('buildHomeAlerts', () => {
  it('merges ops and buyer alerts without duplicates', () => {
    const alerts = buildHomeAlerts({
      canOps: true,
      buyerMode: true,
      summary: { outbox_pending: 3, drift_alert: true },
      meta: { license: { state: 'expired' } },
      incidents: { shards: [{ shard_id: 1, ping_ok: false, ping_error: 'timeout' }] },
      buyerPortfolio: {
        overspendCount: 2,
        alerts: [{ id: 'buyer-custom', level: 'warning', title: 'Budget', detail: 'Near limit' }],
      },
    });
    const ids = alerts.map((a) => a.id);
    assert.ok(ids.includes('outbox-pending'));
    assert.ok(ids.includes('license-state'));
    assert.ok(ids.includes('shard-down-1'));
    assert.ok(ids.includes('buyer-overspend'));
    assert.equal(new Set(ids).size, ids.length);
  });
});
