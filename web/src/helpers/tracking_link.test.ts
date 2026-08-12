import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { buildTrackingLink, defaultClickTemplate } from './tracking_link.js';

describe('tracking_link', () => {
  it('builds link with subs', () => {
    const tpl = defaultClickTemplate('trk.example.com');
    const url = buildTrackingLink(tpl, 'camp-1', { sub1: 'fb', sub2: 'us' });
    assert.ok(url.includes('campaign_id=camp-1'));
    assert.ok(url.includes('sub1=fb'));
    assert.ok(url.includes('sub2=us'));
  });

  it('substitutes sub30', () => {
    const tpl = 'https://trk/click?campaign_id={campaign_id}&sub30={sub30}';
    const url = buildTrackingLink(tpl, 'c-1', { sub30: 'deep' });
    assert.ok(url.includes('sub30=deep'));
  });
});
