import assert from 'node:assert/strict';
import test from 'node:test';

import { CLEAR_UUID, resolveOptionalUuidPatchValue } from './clear_uuid.ts';

test('resolveOptionalUuidPatchValue_holdoutClearsWithNilUuid', () => {
  assert.equal(
    resolveOptionalUuidPatchValue('', '11111111-1111-4111-8111-111111111111'),
    CLEAR_UUID,
  );
});

test('resolveOptionalUuidPatchValue_holdoutRejectsEmptyStringPatch', () => {
  assert.equal(resolveOptionalUuidPatchValue('', undefined), undefined);
  assert.equal(resolveOptionalUuidPatchValue('', ''), undefined);
});

test('resolveOptionalUuidPatchValue_holdoutUnchangedSkipsPatch', () => {
  const id = '11111111-1111-4111-8111-111111111111';
  assert.equal(resolveOptionalUuidPatchValue(id, id), undefined);
});
