import assert from 'node:assert/strict';
import test from 'node:test';

import { buildExportHubHref, resolveExportHubReturnHref } from '@/lib/export_hub_paths';

test('buildExportHubHref encodes return_to for round-trip navigation', () => {
  const href = buildExportHubHref({
    returnTo: '/campaigns?customer_id=abc&status=ACTIVE',
  });
  assert.equal(href, '/exports?return_to=%2Fcampaigns%3Fcustomer_id%3Dabc%26status%3DACTIVE');
});

// E2-D9: Export Hub return_to round-trip (see resolveExportHubReturnHref in export_hub_paths.ts).
test('resolveExportHubReturnHref rejects external targets', () => {
  const params = new URLSearchParams('return_to=https%3A%2F%2Fevil.example');
  assert.equal(resolveExportHubReturnHref(params), '/campaigns');
});

test('resolveExportHubReturnHref restores in-app return path', () => {
  const params = new URLSearchParams(
    'return_to=%2Fcampaigns%3Fcustomer_id%3Dabc%26status%3DACTIVE'
  );
  assert.equal(
    resolveExportHubReturnHref(params),
    '/campaigns?customer_id=abc&status=ACTIVE'
  );
});
