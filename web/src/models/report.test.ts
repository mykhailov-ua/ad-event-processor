import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { isRetiredReport, retiredReportAlt } from './report.js';

describe('retiredReportAlt', () => {
  it('maps source-margin to source-quality with alt text', () => {
    const alt = retiredReportAlt('source-margin');
    assert.ok(alt);
    assert.equal(alt.href, '/reports/source-quality');
    assert.equal(alt.label, 'Source quality');
    assert.match(alt.title, /Source quality/);
  });

  it('maps unit economics to placements with alt text', () => {
    const alt = retiredReportAlt('campaign-unit-economics');
    assert.ok(alt);
    assert.equal(alt.href, '/reports/placements');
    assert.match(alt.title, /Placements/);
  });

  it('returns null for live reports', () => {
    assert.equal(retiredReportAlt('placements'), null);
    assert.equal(isRetiredReport('placements'), false);
  });
});
