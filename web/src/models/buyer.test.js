import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  mapBuyerDashboard,
  buyerCampaignStat,
  buyerCampaignIndex,
  sortPortfolioByDrift,
  portfolioDriftPct,
} from './buyer.js';

describe('mapBuyerDashboard', () => {
  it('does not materialize statsById (zero duplicate structs)', () => {
    const vm = mapBuyerDashboard({
      active: 1,
      campaigns: [
        { id: 'c-1', impressions_7d: 10, clicks_7d: 2 },
        { id: 'c-2', impressions_7d: 20, clicks_7d: 4 },
      ],
    });
    assert.equal('statsById' in vm, false);
    assert.equal(vm.campaigns.length, 2);
  });
});

describe('buyerCampaignStat', () => {
  it('reads stats from campaign row in O(1)', () => {
    const stat = buyerCampaignStat({ impressions_7d: 42, clicks_7d: 7 });
    assert.deepEqual(stat, { impressions: 42, clicks: 7 });
  });
});

describe('sortPortfolioByDrift', () => {
  it('sorts by server pacing_drift_pct when present', () => {
    const rows = sortPortfolioByDrift([
      { id: 'low', pacing_drift_pct: 5 },
      { id: 'high', pacing_drift_pct: 42 },
      { id: 'mid', pacing_drift_pct: -18 },
    ]);
    assert.equal(rows[0].row.id, 'high');
    assert.equal(rows[1].row.id, 'mid');
    assert.equal(rows[2].row.id, 'low');
    assert.equal(portfolioDriftPct(rows[1].row), 18);
  });

  it('falls back to heuristic score when pacing_drift_pct is missing', () => {
    const rows = sortPortfolioByDrift([
      { id: 'active', status: 'ACTIVE', impressions_7d: 5000 },
      { id: 'paused', status: 'PAUSED', impressions_7d: 0 },
    ]);
    assert.equal(rows[0].row.id, 'paused');
  });
});

describe('buyerCampaignIndex', () => {
  it('reuses cache when source array unchanged', () => {
    const campaigns = [{ id: 'a' }, { id: 'b' }];
    const first = buyerCampaignIndex(campaigns);
    const second = buyerCampaignIndex(campaigns, first);
    assert.equal(first, second);
    assert.equal(second.a.id, 'a');
  });

  it('lookup path is O(1) per row after index build O(n)', () => {
    const campaigns = [{ id: 'x', impressions_7d: 5 }];
    const index = buyerCampaignIndex(campaigns);
    const stat = buyerCampaignStat(index.x);
    assert.equal(stat.impressions, 5);
  });
});
