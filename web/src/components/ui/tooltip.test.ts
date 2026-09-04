import assert from 'node:assert/strict';
import test from 'node:test';

import { computeTooltipCoords } from './tooltip_position.ts';

const rect = {
  left: 100,
  right: 132,
  top: 8,
  bottom: 40,
  width: 32,
  height: 32,
} as DOMRect;

test('computeTooltipCoords centers bottom tooltips below trigger', () => {
  const coords = computeTooltipCoords(rect, 'bottom', 'center', 4);

  assert.equal(coords.left, 116);
  assert.equal(coords.top, 44);
  assert.equal(coords.transform, 'translateX(-50%)');
});

test('computeTooltipCoords_holdout does not reuse top transform for bottom placement', () => {
  const coords = computeTooltipCoords(rect, 'bottom', 'center', 4);

  assert.notEqual(coords.transform, 'translate(-50%, -100%)');
});

test('computeTooltipCoords aligns end placement to trigger right edge', () => {
  const coords = computeTooltipCoords(rect, 'top', 'end', 4);

  assert.equal(coords.left, 132);
  assert.equal(coords.top, 4);
  assert.equal(coords.transform, 'translate(-100%, -100%)');
});
