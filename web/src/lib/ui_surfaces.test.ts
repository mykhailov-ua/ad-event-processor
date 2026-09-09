import assert from 'node:assert/strict';
import test from 'node:test';

import { uiMessageSurfaceClass, uiSurfaces } from './ui_surfaces.ts';

test('uiSurfaces message bases include shared padding contract', () => {
  assert.ok(uiSurfaces.message.includes('p-4'));
  assert.ok(uiSurfaces.messageError.includes('border-destructive'));
  assert.ok(uiSurfaces.messageMuted.includes('text-muted-foreground'));
  assert.ok(uiSurfaces.panel.includes('border-border'));
  assert.ok(uiSurfaces.control.includes('min-h-7'));
  assert.ok(uiSurfaces.toolbarBand.includes('flex-wrap'));
  assert.ok(uiSurfaces.chip.includes('whitespace-nowrap'));
  assert.ok(uiSurfaces.summaryBand.includes('bg-primary/5'));
  assert.ok(uiSurfaces.tableHost.includes('border-border'));
  assert.ok(uiSurfaces.tableHostFill.includes('overflow-hidden'));
  assert.ok(uiSurfaces.directoryStack.includes('flex-col'));
  assert.ok(uiSurfaces.metaLinksBand.includes('text-muted-foreground'));
  assert.ok(uiSurfaces.actionLinksBand.includes('flex-wrap'));
});

test('uiMessageSurfaceClass maps tones to modifier classes', () => {
  assert.equal(uiMessageSurfaceClass('error'), uiSurfaces.messageError);
  assert.equal(uiMessageSurfaceClass('warning'), uiSurfaces.messageWarning);
});
