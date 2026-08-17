import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  TRAFFIC_SOURCE_TEMPLATES,
  isNetworkMacro,
  templateParamMap,
  trafficSourceById,
} from './traffic_source_templates.js';
import { buildTemplatedClickURL } from '../helpers/traffic_source_url.js';

describe('traffic_source_templates', () => {
  it('ships exactly 33 presets', () => {
    assert.equal(TRAFFIC_SOURCE_TEMPLATES.length, 33);
    const ids = new Set(TRAFFIC_SOURCE_TEMPLATES.map((t) => t.id));
    assert.equal(ids.size, 33);
  });

  it('detects network macros', () => {
    assert.equal(isNetworkMacro('{{campaign.id}}'), true);
    assert.equal(isNetworkMacro('{gclid}'), true);
    assert.equal(isNetworkMacro('__CAMPAIGN_ID__'), true);
    assert.equal(isNetworkMacro('facebook'), false);
  });

  it('pre-fills Meta Cost Sync join on sub2 + ad_campaign_id', () => {
    const fb = trafficSourceById('meta-facebook');
    assert.ok(fb);
    const map = templateParamMap(fb!);
    assert.equal(map.sub2, '{{campaign.id}}');
    assert.equal(map.ad_campaign_id, '{{campaign.id}}');
    assert.equal(fb!.cost_sync, 'meta');
  });

  it('builds paste-ready click URL without encoding macros', () => {
    const fb = trafficSourceById('meta-facebook')!;
    const url = buildTemplatedClickURL(
      'https://trk.example.com/click?campaign_id={campaign_id}&sub1={sub1}',
      '550e8400-e29b-41d4-a716-446655440000',
      templateParamMap(fb),
    );
    assert.ok(url.startsWith('https://trk.example.com/click?'));
    assert.ok(url.includes('campaign_id=550e8400-e29b-41d4-a716-446655440000'));
    assert.ok(url.includes('sub2={{campaign.id}}'));
    assert.ok(url.includes('ad_campaign_id={{campaign.id}}'));
    assert.ok(!url.includes('%7B%7Bcampaign.id%7D%7D'));
  });

  it('appends dmr=1 and UTM params (CPA-M4 fixture)', () => {
    const fb = trafficSourceById('meta-facebook')!;
    const url = buildTemplatedClickURL(
      'https://trk.example.com/click?campaign_id={campaign_id}&sub1={sub1}',
      '550e8400-e29b-41d4-a716-446655440000',
      templateParamMap(fb),
      {
        dmr: true,
        utm: { utm_source: 'facebook', utm_medium: 'cpc', utm_campaign: 'summer' },
      },
    );
    assert.ok(url.includes('dmr=1'));
    assert.ok(url.includes('utm_source=facebook'));
    assert.ok(url.includes('utm_campaign=summer'));
    assert.ok(url.indexOf('dmr=1') < url.indexOf('utm_source='));
  });
});
