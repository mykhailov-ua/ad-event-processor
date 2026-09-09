import assert from 'node:assert/strict';
import test from 'node:test';

import { copyTextToClipboard } from './copy_text_to_clipboard.ts';

test('copyTextToClipboard_holdoutRejectsEmptyValue', async () => {
  await assert.rejects(() => copyTextToClipboard(''), /empty value/);
  await assert.rejects(() => copyTextToClipboard('   '), /empty value/);
});
