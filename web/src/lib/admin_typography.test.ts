import assert from 'node:assert/strict';
import test from 'node:test';

import {
  ADMIN_MONO_CLASS,
  ADMIN_MONO_DATA_KINDS,
  ADMIN_NUMERIC_CLASS,
  ADMIN_TABULAR_CLASS,
  ADMIN_TABULAR_DATA_KINDS,
  formatAdminEnumLabel,
  formatCampaignStatusLabel,
} from './admin_typography.ts';

test('admin typography separates tabular metrics from mono wire data', () => {
  assert.equal(ADMIN_TABULAR_CLASS, 'tabular-nums');
  assert.equal(ADMIN_NUMERIC_CLASS, 'tabular-nums');
  assert.equal(ADMIN_MONO_CLASS, 'font-mono tabular-nums');
  assert.ok(ADMIN_TABULAR_DATA_KINDS.includes('money'));
  assert.ok(ADMIN_TABULAR_DATA_KINDS.includes('display_id'));
  assert.ok(ADMIN_MONO_DATA_KINDS.includes('uuid'));
  assert.ok(!ADMIN_TABULAR_DATA_KINDS.includes('uuid'));
});

test('formatAdminEnumLabel title-cases enums and preserves abbreviations', () => {
  assert.equal(formatCampaignStatusLabel('ACTIVE'), 'Active');
  assert.equal(formatCampaignStatusLabel('PAUSED', 'PAUSED'), 'Paused');
  assert.equal(formatAdminEnumLabel('EVEN'), 'Even');
  assert.equal(formatAdminEnumLabel('meta_social_funnel'), 'Meta social funnel');
});
