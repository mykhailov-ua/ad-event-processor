import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { apiPathTemplate, recordApiTiming, apiTimingReport, apiTimingReset } from './api_timing.js';

describe('apiPathTemplate', () => {
  it('replaces UUID segments with {id}', () => {
    const tpl = apiPathTemplate('/api/v1/campaigns/550e8400-e29b-41d4-a716-446655440000/stats');
    assert.equal(tpl, '/api/v1/campaigns/{id}/stats');
  });
});

describe('apiTimingReport', () => {
  it('flags paths with p95 >= 500ms', () => {
    apiTimingReset();
    const path = '/api/v1/campaigns/{id}';
    for (let i = 0; i < 10; i++) recordApiTiming(path, 600);
    const report = apiTimingReport();
    assert.ok(report.slowPaths.includes(path));
    assert.equal(report.paths[path].p95Ms, 600);
    apiTimingReset();
  });
});
