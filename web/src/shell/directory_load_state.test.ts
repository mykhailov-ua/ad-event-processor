import assert from 'node:assert/strict';
import test from 'node:test';

import {
  resolveDirectoryLoadPhase,
  shouldShowDirectoryRefreshError,
} from '@/shell/directory_load_state';

const err = new Error('boom');

test('resolveDirectoryLoadPhase loading without snapshot', () => {
  assert.equal(
    resolveDirectoryLoadPhase({ fetching: true, error: undefined, hasSnapshot: false }),
    'loading'
  );
});

test('resolveDirectoryLoadPhase blocking error without snapshot', () => {
  assert.equal(
    resolveDirectoryLoadPhase({ fetching: false, error: err, hasSnapshot: false }),
    'blocking-error'
  );
});

test('resolveDirectoryLoadPhase ready with snapshot despite error', () => {
  assert.equal(
    resolveDirectoryLoadPhase({ fetching: false, error: err, hasSnapshot: true }),
    'ready'
  );
});

test('shouldShowDirectoryRefreshError only when snapshot exists', () => {
  assert.equal(shouldShowDirectoryRefreshError({ fetching: false, error: err, hasSnapshot: true }), true);
  assert.equal(
    shouldShowDirectoryRefreshError({ fetching: false, error: err, hasSnapshot: false }),
    false
  );
});
