import assert from 'node:assert/strict';
import test from 'node:test';

import { mergeOverlayPosition } from '@/lib/overlay_position_state';

test('mergeOverlayPosition returns same reference when coords unchanged (OV-3)', () => {
  const prev = { top: 12, left: 8, visibility: 'visible' as const };
  const next = { top: 12, left: 8, visibility: 'visible' as const };
  assert.equal(mergeOverlayPosition(prev, next), prev);
});

test('mergeOverlayPosition_holdout returns next when a coord changes', () => {
  const prev = { top: 12, left: 8 };
  const next = { top: 13, left: 8 };
  assert.equal(mergeOverlayPosition(prev, next), next);
});
