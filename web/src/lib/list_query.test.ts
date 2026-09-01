import test from 'node:test';
import assert from 'node:assert/strict';
import {
  clampListLimit,
  DEFAULT_LIST_LIMIT,
  parseListLimit,
  parseListOffset,
} from './list_query.ts';

test('parseListLimit returns default when raw is missing or invalid', () => {
  assert.equal(parseListLimit(null), DEFAULT_LIST_LIMIT);
  assert.equal(parseListLimit(''), DEFAULT_LIST_LIMIT);
  assert.equal(parseListLimit('0'), DEFAULT_LIST_LIMIT);
  assert.equal(parseListLimit('-1'), DEFAULT_LIST_LIMIT);
  assert.equal(parseListLimit('abc'), DEFAULT_LIST_LIMIT);
});

test('parseListLimit parses valid values', () => {
  assert.equal(parseListLimit('1'), 1);
  assert.equal(parseListLimit('50'), 50);
});

test('parseListLimit caps at default max of 500', () => {
  assert.equal(parseListLimit('500'), 500);
  assert.equal(parseListLimit('999'), 500);
});

test('parseListLimit caps at custom max', () => {
  assert.equal(parseListLimit('200', 200), 200);
  assert.equal(parseListLimit('500', 200), 200);
  assert.equal(parseListLimit('100', 100), 100);
  assert.equal(parseListLimit('150', 100), 100);
});

test('parseListOffset returns zero when raw is missing or invalid', () => {
  assert.equal(parseListOffset(null), 0);
  assert.equal(parseListOffset(''), 0);
  assert.equal(parseListOffset('-1'), 0);
  assert.equal(parseListOffset('abc'), 0);
});

test('parseListOffset parses valid values', () => {
  assert.equal(parseListOffset('0'), 0);
  assert.equal(parseListOffset('100'), 100);
});

test('clampListLimit clamps to optimal UI range', () => {
  assert.equal(clampListLimit(0), DEFAULT_LIST_LIMIT);
  assert.equal(clampListLimit(50), 50);
  assert.equal(clampListLimit(100), 100);
  assert.equal(clampListLimit(250), 100);
});
