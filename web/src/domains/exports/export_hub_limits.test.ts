import assert from 'node:assert/strict';
import test from 'node:test';

import {
  clampExportHubRowLimit,
  normalizeExportHubRowLimitDraft,
  parseExportHubRowLimitDraft,
  resolveExportHubRowLimitBounds,
} from '@/domains/exports/export_hub_limits';

test('resolveExportHubRowLimitBounds caps license-gated exports', () => {
  const gated = resolveExportHubRowLimitBounds({ licenseGated: true });
  assert.equal(gated.max, 1000);
  const open = resolveExportHubRowLimitBounds({ licenseGated: false });
  assert.equal(open.max, 5_000_000);
  assert.equal(open.default, 100_000);
});

test('clampExportHubRowLimit enforces min and max', () => {
  const bounds = resolveExportHubRowLimitBounds();
  assert.equal(clampExportHubRowLimit(0, bounds), bounds.min);
  assert.equal(clampExportHubRowLimit(9_999_999, bounds), bounds.max);
  assert.equal(parseExportHubRowLimitDraft('2500', bounds), 2500);
});

test('normalizeExportHubRowLimitDraft clamps over tier max', () => {
  const bounds = resolveExportHubRowLimitBounds({ licenseGated: true });
  const result = normalizeExportHubRowLimitDraft('99999', bounds);
  assert.equal(result.value, bounds.max);
  assert.equal(result.wasClamped, true);
});
