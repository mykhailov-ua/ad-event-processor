import assert from 'node:assert/strict';
import test from 'node:test';

import { resolveCommandPaletteHref } from '@/lib/command_palette_href';

const CAMPAIGN_ID = '00000000-0000-4000-8000-000000000101';

test('resolveCommandPaletteHref appends edit to bare campaign detail routes', () => {
  assert.equal(
    resolveCommandPaletteHref(`/campaigns/${CAMPAIGN_ID}`),
    `/campaigns/${CAMPAIGN_ID}/edit`
  );
  assert.equal(
    resolveCommandPaletteHref(`/campaigns/${CAMPAIGN_ID}?tab=ops`),
    `/campaigns/${CAMPAIGN_ID}/edit?tab=ops`
  );
});

test('resolveCommandPaletteHref maps legacy report routes to export hub', () => {
  const href = resolveCommandPaletteHref('/reports/campaign-stats?customer_id=abc');
  assert.equal(href.startsWith('/exports?'), true);
  const params = new URLSearchParams(href.slice('/exports?'.length));
  assert.equal(params.get('kind'), 'report');
  assert.equal(params.get('report_key'), 'campaign-stats');
  assert.equal(params.get('customer_id'), 'abc');
  assert.equal(params.get('entry'), 'report-campaign-stats');
});

test('resolveCommandPaletteHref maps nested report keys and platform campaigns', () => {
  const mlHref = resolveCommandPaletteHref('/reports/ml%2Ffeature-spikes');
  const mlParams = new URLSearchParams(mlHref.slice('/exports?'.length));
  assert.equal(mlParams.get('report_key'), 'ml/feature-spikes');

  assert.equal(
    resolveCommandPaletteHref('/platform-campaigns'),
    '/integrations/platform-campaigns'
  );
});
