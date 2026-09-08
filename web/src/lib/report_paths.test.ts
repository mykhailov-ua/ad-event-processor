import test from 'node:test';
import assert from 'node:assert/strict';

import { buildReportJobsHref, reportHubPath, reportKeyToApiPath, resolveReportCatalogKey } from './report_paths.ts';

test('reportKeyToApiPath maps nested ML report keys', () => {
  assert.equal(reportKeyToApiPath('ml/score-distribution'), '/api/v1/reports/ml/score-distribution');
  assert.equal(reportKeyToApiPath('ml/feature-spikes'), '/api/v1/reports/ml/feature-spikes');
  assert.equal(reportKeyToApiPath('ml/shadow-delta'), '/api/v1/reports/ml/shadow-delta');
});

test('reportKeyToApiPath maps RTB nested paths', () => {
  assert.equal(reportKeyToApiPath('rtb-overview'), '/api/v1/reports/rtb/overview');
  assert.equal(reportKeyToApiPath('rtb-no-bid-reasons'), '/api/v1/reports/rtb/no-bid-reasons');
});

test('reportKeyToApiPath maps telegram nested paths', () => {
  assert.equal(reportKeyToApiPath('telegram'), '/api/v1/reports/telegram');
  assert.equal(reportKeyToApiPath('telegram/summary'), '/api/v1/reports/telegram/summary');
  assert.equal(reportKeyToApiPath('telegram/fraud'), '/api/v1/reports/telegram/fraud');
});

test('reportKeyToApiPath maps clicks alias to click-log API path', () => {
  assert.equal(reportKeyToApiPath('clicks'), '/api/v1/reports/clicks');
});

test('buildReportJobsHref pre-fills query params', () => {
  assert.equal(
    buildReportJobsHref({
      reportKey: 'fraud-evidence-pack-bulk',
      customerId: '00000000-0000-4000-8000-000000000001',
    }),
    '/reports/jobs?report_key=fraud-evidence-pack-bulk&customer_id=00000000-0000-4000-8000-000000000001'
  );
});

test('resolveReportCatalogKey maps clicks alias', () => {
  assert.equal(resolveReportCatalogKey('clicks'), 'click-log');
});

test('reportHubPath maps telegram and slash keys to SPA routes', () => {
  assert.equal(reportHubPath('telegram'), '/reports/telegram');
  assert.equal(reportHubPath('telegram/summary'), '/reports/telegram/summary');
  assert.equal(reportHubPath('click-log'), '/reports/click-log');
  assert.equal(reportHubPath('ml/score-distribution'), '/reports/ml%2Fscore-distribution');
});
