import assert from 'node:assert/strict';
import test from 'node:test';

import { buildExportOnlyReportHubHref } from '@/shell/export_only_report_hub_href';
import { exportOnlyReportStubBannerMessage } from '@/shell/export_only_report_stub_message';

test('buildExportOnlyReportHubHref includes stub params in export hub link', () => {
  const href = buildExportOnlyReportHubHref(
    {
      reportKey: 'fraud-evidence-pack-bulk',
      customerId: '00000000-0000-4000-8000-000000000001',
      from: '2026-09-01T00:00:00.000Z',
      to: '2026-09-08T00:00:00.000Z',
      format: 'zip',
    },
    '/reports/fraud-evidence-pack-bulk?customer_id=abc'
  );

  const [path, query] = href.split('?');
  assert.equal(path, '/exports');
  const params = new URLSearchParams(query);
  assert.equal(params.get('kind'), 'report');
  assert.equal(params.get('report_key'), 'fraud-evidence-pack-bulk');
  assert.equal(params.get('customer_id'), '00000000-0000-4000-8000-000000000001');
  assert.equal(params.get('from'), '2026-09-01T00:00:00.000Z');
  assert.equal(params.get('to'), '2026-09-08T00:00:00.000Z');
  assert.equal(params.get('format'), 'zip');
  assert.equal(params.get('return_to'), '/reports/fraud-evidence-pack-bulk?customer_id=abc');
});

test('exportOnlyReportStubBannerMessage mentions static catalog when key is unknown', () => {
  const message = exportOnlyReportStubBannerMessage(true);
  assert.match(message, /not listed in the static catalog/i);
  assert.match(message, /report_key query parameters/i);
  assert.match(message, /async export only/i);
});

test('exportOnlyReportStubBannerMessage omits catalog note for typed keys', () => {
  const message = exportOnlyReportStubBannerMessage(false);
  assert.doesNotMatch(message, /static catalog/i);
  assert.match(message, /async export only/i);
});

test('buildExportOnlyReportHubHref prefills date range when from or to missing', () => {
  const href = buildExportOnlyReportHubHref(
    {
      reportKey: 'fraud-evidence-pack-bulk',
      defaultRange: '7d',
    },
    '/reports/fraud-evidence-pack-bulk'
  );

  const params = new URLSearchParams(href.split('?')[1] ?? '');
  const from = params.get('from');
  const to = params.get('to');
  assert.ok(from);
  assert.ok(to);
  assert.ok(!Number.isNaN(Date.parse(from)));
  assert.ok(!Number.isNaN(Date.parse(to)));
  assert.equal(params.get('report_key'), 'fraud-evidence-pack-bulk');
  assert.equal(params.get('return_to'), '/reports/fraud-evidence-pack-bulk');
});
