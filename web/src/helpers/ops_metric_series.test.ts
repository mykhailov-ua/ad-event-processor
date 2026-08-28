import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  copyMetricPoints,
  copySnapshotSeriesSoA,
  recordSnapshot,
  rangeMsFromHours,
} from './ops_metric_series.js';

describe('copyMetricPoints', () => {
  it('returns flat pair when fewer than two in-range points', () => {
    const outTs = new Float64Array(8);
    const outVal = new Float64Array(8);
    const now = 1_000_000;
    const rangeMs = rangeMsFromHours(1);
    const n = copyMetricPoints(null, 42, rangeMs, outTs, outVal, now);
    assert.equal(n, 2);
    assert.equal(outVal[0], 42);
    assert.equal(outVal[1], 42);
    assert.equal(outTs[1], now);
  });

  it('copies in-range points without allocation', () => {
    const outTs = new Float64Array(8);
    const outVal = new Float64Array(8);
    const now = 10_000;
    const rangeMs = 5000;
    const points = [
      { ts: 6000, value: 1 },
      { ts: 8000, value: 3 },
      { ts: 9500, value: 5 },
    ];
    const n = copyMetricPoints(points, 0, rangeMs, outTs, outVal, now);
    assert.equal(n, 3);
    assert.equal(outVal[2], 5);
  });
});

describe('recordSnapshot SoA ring', () => {
  it('coalesces updates within 5s', () => {
    const outTs = new Float64Array(8);
    const outVal = new Float64Array(8);
    const t0 = 50_000;
    recordSnapshot('test-metric', 10);
    recordSnapshot('test-metric', 20);
    const n = copySnapshotSeriesSoA('test-metric', 0, rangeMsFromHours(24), outTs, outVal, t0);
    assert.equal(n, 2);
    assert.equal(outVal[1], 20);
  });
});
