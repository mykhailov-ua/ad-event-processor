import assert from 'node:assert/strict';
import test from 'node:test';

import { resolvePopoverAlign } from './popover_align.ts';

test('resolvePopoverAlign opens left when trigger is on the right', () => {
  const element = {
    getBoundingClientRect: () => ({
      left: 900,
      right: 1180,
      top: 0,
      bottom: 0,
      width: 280,
      height: 36,
      x: 900,
      y: 0,
      toJSON: () => ({}),
    }),
  } as HTMLElement;

  assert.equal(resolvePopoverAlign(element, 1280), 'end');
});

test('resolvePopoverAlign opens right when trigger is on the left', () => {
  const element = {
    getBoundingClientRect: () => ({
      left: 24,
      right: 304,
      top: 0,
      bottom: 0,
      width: 280,
      height: 36,
      x: 24,
      y: 0,
      toJSON: () => ({}),
    }),
  } as HTMLElement;

  assert.equal(resolvePopoverAlign(element, 1280), 'start');
});
