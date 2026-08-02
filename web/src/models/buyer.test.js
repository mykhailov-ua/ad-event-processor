import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  mapBuyerDashboard,
  buyerCampaignStat,
  buyerCampaignIndex,
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
