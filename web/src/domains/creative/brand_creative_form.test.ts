import assert from 'node:assert/strict';
import test from 'node:test';

import { buildBrandCreativeBody, parseCreativeWeight } from './brand_creative_form.ts';

test('parseCreativeWeight accepts positive integers', () => {
  const result = parseCreativeWeight('100');
  assert.equal(result.ok, true);
  if (result.ok) {
    assert.equal(result.weight, 100);
  }
});

test('parseCreativeWeight_holdoutRejectsParseIntSlop', () => {
  const result = parseCreativeWeight('100abc');
  assert.equal(result.ok, false);
  if (!result.ok) {
    assert.match(result.error, /whole number/i);
  }
  assert.equal(Number.parseInt('100abc', 10), 100);
});

test('parseCreativeWeight_holdoutRejectsZeroAndNegative', () => {
  assert.equal(parseCreativeWeight('0').ok, false);
  assert.equal(parseCreativeWeight('-5').ok, false);
});

test('parseCreativeWeight rejects decimals and empty', () => {
  assert.equal(parseCreativeWeight('1.5').ok, false);
  assert.equal(parseCreativeWeight('').ok, false);
  assert.equal(parseCreativeWeight('   ').ok, false);
});

test('buildBrandCreativeBody maps validated fields', () => {
  const result = buildBrandCreativeBody('Banner A', 'https://example.com/lp', '50', 'active');
  assert.equal(result.ok, true);
  if (result.ok) {
    assert.equal(result.body.name, 'Banner A');
    assert.equal(result.body.landing_url, 'https://example.com/lp');
    assert.equal(result.body.weight, 50);
    assert.equal(result.body.status, 'active');
  }
});

test('buildBrandCreativeBody surfaces weight errors before name', () => {
  const result = buildBrandCreativeBody('Banner A', 'https://example.com/lp', '0', 'active');
  assert.equal(result.ok, false);
  if (!result.ok) {
    assert.match(result.error, /greater than zero/i);
  }
});
