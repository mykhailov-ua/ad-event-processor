import assert from 'node:assert/strict';
import test from 'node:test';

import {
  formatSettingsDisplayValue,
  humanizeSettingsSlug,
} from '../../lib/settings_display_values.ts';

test('formatSettingsDisplayValue maps deployment profiles', () => {
  assert.equal(formatSettingsDisplayValue('single_vps', 'profile'), 'Single VPS');
  assert.equal(formatSettingsDisplayValue('compose_dev', 'profile'), 'Compose dev');
});

test('formatSettingsDisplayValue maps ingress schemas', () => {
  assert.equal(
    formatSettingsDisplayValue('ad_event_processor_native', 'ingress_schema'),
    'Native ingest',
  );
  assert.equal(formatSettingsDisplayValue('openrtb_3', 'ingress_schema'), 'OpenRTB 3');
});

test('humanizeSettingsSlug title-cases unknown slugs', () => {
  assert.equal(humanizeSettingsSlug('edge_expose_openrtb'), 'Edge Expose OpenRTB');
});
