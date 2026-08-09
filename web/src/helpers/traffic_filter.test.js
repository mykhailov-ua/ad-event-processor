import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { parseTrafficFilter, serializeTrafficFilter } from './traffic_filter.js';

describe('traffic_filter', () => {
  it('round-trips structured rules', () => {
    const raw = serializeTrafficFilter({
      allowReferrers: ['partner.com'],
      blockReferrers: ['spam.net'],
      blockEmptyReferrer: true,
    });
    const parsed = parseTrafficFilter(raw);
    assert.deepEqual(parsed.allowReferrers, ['partner.com']);
    assert.deepEqual(parsed.blockReferrers, ['spam.net']);
    assert.equal(parsed.blockEmptyReferrer, true);
  });
});
