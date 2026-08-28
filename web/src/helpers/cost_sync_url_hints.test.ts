import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { COST_SYNC_URL_HINTS, costSyncHintsForNetwork } from '../helpers/cost_sync_url_hints.js';
import {
  ingressCostMacroPlaceholder,
  isIngressCostParam,
  resolveIngressCostParam,
} from '../helpers/ingress_cost_url.js';
import { buildTemplatedClickURL } from '../helpers/traffic_source_url.js';
import { templateParamMap, trafficSourceById } from '../models/traffic_source_templates.js';

describe('cost_sync_url_hints', () => {
  it('maps template cost_sync tags to API network ids', () => {
    assert.equal(COST_SYNC_URL_HINTS.meta.apiNetworkId, 'facebook');
    assert.equal(COST_SYNC_URL_HINTS.google.apiNetworkId, 'google');
    assert.equal(COST_SYNC_URL_HINTS.tiktok.apiNetworkId, 'tiktok');
  });

  it('returns hints for Meta templates', () => {
    const hints = costSyncHintsForNetwork('meta');
    assert.ok(hints);
    assert.ok(hints!.requiredKeys.some((row) => row.key === 'ad_campaign_id'));
    assert.ok(hints!.requiredKeys.some((row) => row.key === 'sub2'));
  });

  it('returns null when template has no cost_sync network', () => {
    assert.equal(costSyncHintsForNetwork(undefined), null);
  });
});

describe('ingress_cost_url', () => {
  it('accepts cost, cpc, and bid param names', () => {
    assert.equal(isIngressCostParam('cost'), true);
    assert.equal(isIngressCostParam('cpc'), true);
    assert.equal(isIngressCostParam('bid'), true);
    assert.equal(isIngressCostParam('spend'), false);
  });

  it('builds macro placeholder without encoding', () => {
    assert.equal(ingressCostMacroPlaceholder('cost'), '{cost}');
    assert.equal(resolveIngressCostParam('cpc'), 'cpc');
    assert.equal(resolveIngressCostParam(''), null);
  });

  it('appends ingress cost macro to click URL', () => {
    const url = buildTemplatedClickURL(
      'https://trk.example.com/click',
      '550e8400-e29b-41d4-a716-446655440000',
      templateParamMap(trafficSourceById('google-ads')!),
      { ingressCost: { param: 'cost', value: '{cost}' } }
    );
    assert.ok(url.includes('cost={cost}'));
    assert.ok(!url.includes('%7Bcost%7D'));
  });
});
