import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildDateAxisTicks,
  buildMoneyAxisScale,
  buildVolumeAxisScale,
  microToUsd,
  pickKeitaroStep,
  usdToMicro,
} from './dashboard_chart_scale.ts';

test('pickKeitaroStep uses 10000 steps for volume near 85k like Keitaro', () => {
  assert.equal(pickKeitaroStep(85_000, 9), 10_000);
});

test('pickKeitaroStep uses 200 USD steps for 600-2400 band like Keitaro', () => {
  assert.equal(pickKeitaroStep(1_800, 9), 200);
});

test('buildVolumeAxisScale builds 0..90000 ticks by 10000 for 85k max', () => {
  const scale = buildVolumeAxisScale(85_000);
  assert.deepEqual(scale.ticks, [
    0, 10_000, 20_000, 30_000, 40_000, 50_000, 60_000, 70_000, 80_000, 90_000,
  ]);
  assert.deepEqual(scale.domain, [0, 90_000]);
});

test('buildMoneyAxisScale builds 600..2400 USD ticks by 200 for Keitaro-like money band', () => {
  const scale = buildMoneyAxisScale(usdToMicro(600), usdToMicro(2_400));
  assert.deepEqual(scale.ticks.map(microToUsd), [
    600, 800, 1_000, 1_200, 1_400, 1_600, 1_800, 2_000, 2_200, 2_400,
  ]);
  assert.deepEqual(scale.domain.map(microToUsd), [600, 2_400]);
});

test('buildDateAxisTicks keeps major labels without daily subdivisions', () => {
  const labels = Array.from({ length: 62 }, (_, index) => `2026-07-${String(index + 1).padStart(2, '0')}`);
  const ticks = buildDateAxisTicks(labels, 8);
  assert.ok(ticks.length <= 10);
  assert.equal(ticks[0], labels[0]);
  assert.equal(ticks.at(-1), labels.at(-1));
  assert.ok(ticks.length < labels.length);
});
