import assert from 'node:assert/strict';
import test from 'node:test';

import { coalesceUserAction, DEFAULT_COALESCE_WINDOW_MS } from './coalesced_user_action.ts';

function gate(lastFiredAtMs: number, nowMs: number, inFlight = false, inFlightGuard = true) {
  return coalesceUserAction({
    lastFiredAtMs,
    nowMs,
    windowMs: DEFAULT_COALESCE_WINDOW_MS,
    inFlight,
    inFlightGuard,
  });
}

test('coalesceUserAction allows first refresh', () => {
  assert.equal(gate(0, 1_000), 'allow');
});

test('coalesceUserAction skips refresh inside leading window', () => {
  assert.equal(gate(1_000, 1_200), 'skip_window');
  assert.equal(gate(1_000, 1_499), 'skip_window');
  assert.equal(gate(1_000, 1_500), 'allow');
});

test('coalesceUserAction skips refresh while in flight', () => {
  assert.equal(gate(0, 10_000, true), 'skip_in_flight');
});

test('coalesceUserAction_holdoutWithoutInFlightGuard allows burst after window', () => {
  const withoutGuard = coalesceUserAction({
    lastFiredAtMs: 0,
    nowMs: 10_000,
    windowMs: DEFAULT_COALESCE_WINDOW_MS,
    inFlight: true,
    inFlightGuard: false,
  });
  assert.equal(withoutGuard, 'allow');
});
