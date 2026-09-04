import assert from 'node:assert/strict';
import test from 'node:test';

import {
  isTrivialSequentialUuid,
  newRandomUuid,
  seedDeterministicUuid,
} from './seed_uuid.ts';

test('seedDeterministicUuid matches Go seed_catalog parity vectors', () => {
  const cases: Array<[string, number, string]> = [
    ['customer', 1, '9e05f63f-e1a2-5c32-9843-03bdd1cb2ff1'],
    ['customer', 2, 'f5629099-566b-573c-b618-94c478c4bd1d'],
    ['campaign', 1, 'abe62900-7466-5a77-8dac-2cf17fd1dd84'],
    ['campaign', 25, '49d5ee7a-7ba5-5f3a-a0b3-6e3172b30f3c'],
    ['user', 1, 'ebdaa0da-ebba-507e-ae1c-73a8ddd7a04b'],
  ];
  for (const [kind, seq, expected] of cases) {
    assert.equal(seedDeterministicUuid(kind, seq), expected, `${kind}:${seq}`);
  }
});

test('seedDeterministicUuid is not trivial sequential', () => {
  for (let seq = 1; seq <= 100; seq += 1) {
    const id = seedDeterministicUuid('campaign', seq);
    assert.equal(isTrivialSequentialUuid(id), false, id);
  }
});

test('isTrivialSequentialUuid rejects legacy dev_mock patterns', () => {
  assert.equal(isTrivialSequentialUuid('00000000-0000-0000-0000-000000000041'), true);
  assert.equal(isTrivialSequentialUuid('00000000-cust-4000-8000-000000000001'), true);
  assert.equal(isTrivialSequentialUuid('00000000-camp-4000-8000-000000000025'), true);
  assert.equal(isTrivialSequentialUuid('9e05f63f-e1a2-5c32-9843-03bdd1cb2ff1'), false);
});

test('newRandomUuid is not trivial sequential', () => {
  for (let index = 0; index < 20; index += 1) {
    const id = newRandomUuid();
    assert.equal(isTrivialSequentialUuid(id), false, id);
  }
});
