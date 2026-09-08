import assert from 'node:assert/strict';
import test from 'node:test';

import { uiMessageSurfaceClass, uiSurfaces } from './ui_surfaces.ts';

test('uiSurfaces message bases include shared padding contract', () => {
  assert.ok(uiSurfaces.message.includes('ui-message-surface'));
  assert.ok(uiSurfaces.messageError.includes('ui-message-surface--error'));
  assert.ok(uiSurfaces.messageMuted.includes('ui-message-surface--muted'));
  assert.ok(uiSurfaces.panel.includes('ui-panel-surface'));
  assert.ok(uiSurfaces.control.includes('ui-control-surface'));
  assert.ok(uiSurfaces.toolbarBand.includes('ui-toolbar-band'));
  assert.ok(uiSurfaces.chip.includes('ui-chip-surface'));
  assert.ok(uiSurfaces.summaryBand.includes('ui-summary-band'));
  assert.ok(uiSurfaces.tableHost.includes('ui-table-host'));
  assert.ok(uiSurfaces.tableHostFill.includes('ui-table-host--fill'));
  assert.ok(uiSurfaces.directoryStack.includes('ui-directory-stack'));
  assert.ok(uiSurfaces.metaLinksBand.includes('ui-meta-links-band'));
  assert.ok(uiSurfaces.actionLinksBand.includes('ui-action-links-band'));
});

test('uiMessageSurfaceClass maps tones to modifier classes', () => {
  assert.equal(uiMessageSurfaceClass('error'), uiSurfaces.messageError);
  assert.equal(uiMessageSurfaceClass('warning'), uiSurfaces.messageWarning);
});
