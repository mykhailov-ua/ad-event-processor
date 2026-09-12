import test from 'node:test';
import assert from 'node:assert/strict';

import {
  allTypedCatalogReportKeys,
  buildReportJobsHref,
  isTypedCatalogReportKey,
  reportKeyToApiPath,
  reportRequiresCustomerScope,
  reportStubPathFromKey,
  reportTitleFromKey,
  REPORT_CATALOG_KEY_ALIASES,
  resolveReportCatalogKey,
  resolveReportDisplayTitle,
  resolveReportStubRequiresCustomer,
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

test('reportRequiresCustomerScope scopes customer reports only', () => {
  assert.equal(reportRequiresCustomerScope('placements'), true);
  assert.equal(reportRequiresCustomerScope('rtb-overview'), false);
  assert.equal(reportRequiresCustomerScope('edge-parity'), false);
});

test('resolveReportStubRequiresCustomer uses catalog scope for typed keys only', () => {
  assert.equal(resolveReportStubRequiresCustomer('placements'), true);
  assert.equal(resolveReportStubRequiresCustomer('rtb-overview'), false);
  assert.equal(resolveReportStubRequiresCustomer('custom-export-report'), false);
});

test('unknown report keys humanize for export stub title', () => {
  const unknownKey = 'custom-export-report';
  assert.equal(isTypedCatalogReportKey(unknownKey), false);
  assert.equal(reportTitleFromKey(unknownKey), 'Custom Export Report');
});

test('reportStubPathFromKey maps catalog keys to report URL splat', () => {
  assert.equal(reportStubPathFromKey('placements'), 'placements');
  assert.equal(reportStubPathFromKey('ml/feature-spikes'), 'ml/feature-spikes');
});

test('ghost-impression-funnel alias resolves canonical stub title', () => {
  const catalogKey = resolveReportCatalogKey('ghost-impression-funnel');
  assert.equal(catalogKey, 'silent-reject-impression-funnel');
  assert.equal(reportTitleFromKey('ghost-impression-funnel'), 'Non-blocking fraud response funnel');
  assert.equal(reportStubPathFromKey(catalogKey), 'ghost-impression-funnel');
});

test('allTypedCatalogReportKeys covers catalog and legacy alias routes', () => {
  const typedCount = allTypedCatalogReportKeys().size;
  const aliasCount = Object.keys(REPORT_CATALOG_KEY_ALIASES).length;
  assert.ok(typedCount >= 47);
  assert.ok(typedCount + aliasCount >= 49);
});
