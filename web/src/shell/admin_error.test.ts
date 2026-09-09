import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const shellDir = path.dirname(fileURLToPath(import.meta.url));

test('admin_error delegates to panel_error', () => {
  const source = readFileSync(path.join(shellDir, 'admin_error.tsx'), 'utf8');
  assert.match(source, /from '@\/shell\/panel_error'/);
  assert.match(source, /return panelError\(error, title, options\)/);
  assert.match(source, /variant="mutation"/);
});

test('directory_page_shell delegates DirectoryMutationError to AdminMutationError', () => {
  const source = readFileSync(path.join(shellDir, 'directory_page_shell.tsx'), 'utf8');
  assert.match(source, /from '@\/shell\/admin_error'/);
  assert.match(source, /AdminMutationError/);
});
