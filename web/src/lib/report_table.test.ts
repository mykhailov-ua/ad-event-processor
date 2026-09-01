import test from 'node:test';
import assert from 'node:assert/strict';

import { deriveColumns, reportMapRowKey } from './report_table.ts';

test('deriveColumns unions keys across rows', () => {
  assert.deepEqual(
    deriveColumns([{ b: 1, a: 2 }, { c: 3, a: 4 }]),
    ['a', 'b', 'c'],
  );
});

test('reportMapRowKey prefers id fields over index', () => {
  const row = { click_id: 'clk-1', reason: 'budget' };
  assert.equal(reportMapRowKey(row, ['reason'], 9), 'clk-1');
});

test('reportMapRowKey uses first scalar column before index', () => {
  const row = { reason: 'timeout' };
  assert.equal(reportMapRowKey(row, ['reason'], 9), 'timeout');
});
