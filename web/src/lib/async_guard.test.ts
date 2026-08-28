import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  createGenerationGuard,
  shouldCommitAsyncResult,
  createInFlightGuard,
  guardTelemetryReport,
  guardTelemetryReset,
} from './async_guard.js';
import { getOrCreate, clearScope, clearAll } from '../helpers/idempotency.js';

describe('createGenerationGuard', () => {
  it('generates strictly monotonic ids', () => {
    const guard = createGenerationGuard();
    const seq = [guard.next(), guard.next(), guard.next()];
    assert.deepEqual(seq, [1, 2, 3]);
    assert.ok(seq[0] < seq[1] && seq[1] < seq[2]);
  });

  it('invalidate() supersedes all prior generations', () => {
    const guard = createGenerationGuard();
    const stale = guard.next();
    guard.invalidate();
    const current = guard.next();
    assert.equal(guard.isCurrent(stale), false);
    assert.equal(guard.isCurrent(current), true);
  });
});

describe('shouldCommitAsyncResult', () => {
  it('rejects stale generations (for all i < G: commit(i, G) = false)', () => {
    const G = 5;
    for (let i = 1; i < G; i++) {
      assert.equal(shouldCommitAsyncResult(i, G), false);
    }
  });

  it('accepts only the current generation', () => {
    assert.equal(shouldCommitAsyncResult(3, 3), true);
    assert.equal(shouldCommitAsyncResult(3, 4), false);
  });

  it('rejects all commits when destroyed', () => {
    assert.equal(shouldCommitAsyncResult(3, 3, true), false);
  });

  it('models out-of-order completion: only last writer wins', () => {
    const guard = createGenerationGuard();
    const opA = guard.next();
    const opB = guard.next();
    const current = guard.current();
    assert.equal(shouldCommitAsyncResult(opA, current), false);
    assert.equal(shouldCommitAsyncResult(opB, current), true);
  });
});

describe('createInFlightGuard', () => {
  it('allows exactly one concurrent acquisition', () => {
    guardTelemetryReset();
    const gate = createInFlightGuard();
    assert.equal(gate.tryAcquire(), true);
    assert.equal(gate.tryAcquire(), false);
    assert.equal(guardTelemetryReport().in_flight_rejected, 1);
    gate.release();
    assert.equal(gate.tryAcquire(), true);
    gate.release();
  });
});

describe('guardTelemetryReport', () => {
  it('counts stale write prevention', () => {
    guardTelemetryReset();
    assert.equal(shouldCommitAsyncResult(1, 2), false);
    assert.equal(guardTelemetryReport().stale_write_prevented, 1);
  });
});

describe('idempotency scope lifecycle', () => {
  it('clearScope enables a fresh key for the same scope', () => {
    clearAll();
    const scope = 'campaign-pause:test';
    const k1 = getOrCreate(scope);
    clearScope(scope);
    const k2 = getOrCreate(scope);
    assert.notEqual(k1, k2);
    clearAll();
  });
});
