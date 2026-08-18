import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { AFFILIATE_POSTBACK_PRESETS, affiliatePostbackById } from './affiliate_postback_presets.js';

describe('affiliate_postback_presets', () => {
  it('ships exactly 36 presets with unique ids', () => {
    assert.equal(AFFILIATE_POSTBACK_PRESETS.length, 36);
    const ids = new Set(AFFILIATE_POSTBACK_PRESETS.map((p) => p.id));
    assert.equal(ids.size, 36);
  });

  it('MaxBounty template uses BidShard click_id and payout macros', () => {
    const mb = affiliatePostbackById('maxbounty');
    assert.ok(mb);
    assert.ok(mb!.url_template.includes('{click_id}'));
    assert.ok(mb!.url_template.includes('{payout}'));
    assert.equal(mb!.network_click_token, '{subid}');
  });

  it('Custom preset is last and paste-ready', () => {
    const last = AFFILIATE_POSTBACK_PRESETS[AFFILIATE_POSTBACK_PRESETS.length - 1];
    assert.equal(last?.id, 'custom');
    assert.ok(last!.url_template.startsWith('https://'));
  });
});
