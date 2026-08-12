import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { deriveCampaignHealth } from './campaign_health.js';

describe('deriveCampaignHealth', () => {
  it('flags margin breach as risk', () => {
    const health = deriveCampaignHealth(
      { status: 'ACTIVE', budget_limit: '100', current_spend: '10' },
      { marginBreach: true },
    );
    assert.equal(health.level, 'risk');
    assert.match(health.title, /Margin guard breach/);
  });

  it('warns when license is in grace', () => {
    const health = deriveCampaignHealth(
      { status: 'ACTIVE', budget_limit: '100', current_spend: '10' },
      { licenseGrace: true },
    );
    assert.equal(health.level, 'warn');
    assert.match(health.title, /grace/i);
  });

  it('combines portfolio margin_breach with utilization', () => {
    const health = deriveCampaignHealth(
      { status: 'ACTIVE', budget_limit: '100', current_spend: '95' },
      { portfolioRow: { utilization_pct: 95, margin_breach: true } },
    );
    assert.equal(health.level, 'risk');
    assert.match(health.title, /Margin guard breach/);
  });
});
