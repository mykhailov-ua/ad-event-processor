import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

import { resolveDirectoryLoadPhase } from '@/shell/directory_load_state';

const err = new Error('boom');

const shellDir = path.dirname(fileURLToPath(import.meta.url));

// Grep contract (manual): rg "from '@/shell/directory_load_state'" \
//   web/src/shell/customer_tab_shell.tsx web/src/shell/editor_page_shell.tsx
test('customer_tab_shell and editor_page_shell share directory_load_state phase helpers', () => {
  for (const file of ['customer_tab_shell.tsx', 'editor_page_shell.tsx']) {
    const source = readFileSync(path.join(shellDir, file), 'utf8');
    assert.match(source, /from '@\/shell\/directory_load_state'/);
    assert.match(source, /resolveDirectoryLoadPhase/);
    assert.match(source, /shouldShowDirectoryRefreshError/);
  }
});

test('resolveDirectoryLoadPhase fetching with error and no snapshot is blocking-error', () => {
  assert.equal(
    resolveDirectoryLoadPhase({ fetching: true, error: err, hasSnapshot: false }),
    'blocking-error'
  );
});
