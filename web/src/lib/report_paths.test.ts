import test from 'node:test';
import assert from 'node:assert/strict';

import {
  buildReportJobsHref,
  reportKeyToApiPath,
  reportTitleFromKey,
  resolveReportCatalogKey,
  resolveReportDisplayTitle,
} from './report_paths.ts';

test('reportKeyToApiPath maps nested ML report keys', () => {
  assert.equal(
    reportKeyToApiPath('ml/score-distribution'),
    '/api/v1/reports/ml/score-distribution'
  );
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

test('buildReportJobsHref routes to export hub', () => {
  const href = buildReportJobsHref({
    reportKey: 'fraud-evidence-pack-bulk',
    customerId: '00000000-0000-4000-8000-000000000001',
  });
  assert.equal(href.startsWith('/exports?'), true);
  const params = new URLSearchParams(href.slice('/exports?'.length));
  assert.equal(params.get('kind'), 'report');
  assert.equal(params.get('report_key'), 'fraud-evidence-pack-bulk');
  assert.equal(params.get('customer_id'), '00000000-0000-4000-8000-000000000001');
});

test('resolveReportCatalogKey maps clicks alias', () => {
  assert.equal(resolveReportCatalogKey('clicks'), 'click-log');
  assert.equal(
    resolveReportCatalogKey('ghost-impression-funnel'),
    'silent-reject-impression-funnel'
  );
});

test('reportTitleFromKey humanizes export catalog keys', () => {
  assert.equal(reportTitleFromKey('true-roi'), 'True ROI');
  assert.equal(reportTitleFromKey('campaign-stats'), 'Campaign stats');
  assert.equal(reportTitleFromKey('fraud-evidence-pack-bulk'), 'Fraud evidence pack bulk');
  assert.equal(reportTitleFromKey('customer-portfolio'), 'Customer portfolio');
  assert.equal(reportTitleFromKey('placements'), 'Placements');
});

test('resolveReportDisplayTitle prefers catalog title', () => {
  assert.equal(resolveReportDisplayTitle('true-roi', 'True ROI'), 'True ROI');
  assert.equal(resolveReportDisplayTitle('true-roi', ''), 'True ROI');
  assert.equal(resolveReportDisplayTitle('true-roi', 'true-roi'), 'True ROI');
});
