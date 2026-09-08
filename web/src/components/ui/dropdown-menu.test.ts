import assert from 'node:assert/strict';
import test from 'node:test';

import { DROPDOWN_MENU_SCROLL_BODY_CLASS } from './dropdown_menu_scroll.ts';

test('DropdownMenu scroll body caps list height (class B / VL-13b)', () => {
  assert.match(DROPDOWN_MENU_SCROLL_BODY_CLASS, /\bmax-h-60\b/);
  assert.match(DROPDOWN_MENU_SCROLL_BODY_CLASS, /\boverflow-y-auto\b/);
  assert.match(DROPDOWN_MENU_SCROLL_BODY_CLASS, /\bui-scrollbar\b/);
});

test('DropdownMenu scroll body_holdout includes overscroll containment', () => {
  assert.ok(DROPDOWN_MENU_SCROLL_BODY_CLASS.includes('overscroll-y-contain'));
});
