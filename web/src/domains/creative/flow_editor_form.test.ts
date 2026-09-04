import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildFlowUpdateBody,
  flowPathsEqual,
  flowPathsToJson,
  parseFlowPathsJson,
} from './flow_editor_form.ts';

test('parseFlowPathsJson rejects non-array JSON', () => {
  const result = parseFlowPathsJson('{"weight":100}');
  assert.equal(result.ok, false);
  if (!result.ok) {
    assert.match(result.error, /array/i);
  }
});

test('flowPathsEqual_holdoutIgnoresObjectKeyReorder', () => {
  const serverPaths = [{ weight: 100, landers: [], offers: [] }];
  const draftJson = '[{"offers":[],"weight":100,"landers":[]}]';
  assert.equal(flowPathsEqual(serverPaths, draftJson), true);
});

test('flowPathsEqual_holdoutDetectsWeightChange', () => {
  const serverPaths = [{ weight: 100, landers: [], offers: [] }];
  const draftJson = '[{"weight":90,"landers":[],"offers":[]}]';
  assert.equal(flowPathsEqual(serverPaths, draftJson), false);
});

test('buildFlowUpdateBody returns parsed paths', () => {
  const result = buildFlowUpdateBody('Main flow', '[{"weight":50,"landers":[],"offers":[]}]');
  assert.equal(result.ok, true);
  if (result.ok) {
    assert.equal(result.body.name, 'Main flow');
    assert.equal(result.body.paths[0]?.weight, 50);
  }
});

test('flowPathsToJson formats array paths', () => {
  const json = flowPathsToJson([{ weight: 1, landers: [], offers: [] }]);
  assert.match(json, /"weight": 1/);
});
