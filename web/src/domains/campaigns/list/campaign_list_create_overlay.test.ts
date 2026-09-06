import assert from 'node:assert/strict';
import test from 'node:test';

import {
  openCampaignCreateDialog,
  openCampaignWizardSheet,
  setCampaignCreateDialogOpen,
  setCampaignWizardSheetOpen,
} from './campaign_list_create_overlay.ts';

test('openCampaignCreateDialog closes wizard before opening create', () => {
  let createOpen = false;
  let wizardOpen = true;
  openCampaignCreateDialog(
    (open) => {
      createOpen = open;
    },
    (open) => {
      wizardOpen = open;
    }
  );
  assert.equal(createOpen, true);
  assert.equal(wizardOpen, false);
});

test('openCampaignWizardSheet_holdout closes create before opening wizard', () => {
  let createOpen = true;
  let wizardOpen = false;
  openCampaignWizardSheet(
    (open) => {
      createOpen = open;
    },
    (open) => {
      wizardOpen = open;
    }
  );
  assert.equal(createOpen, false);
  assert.equal(wizardOpen, true);
});

test('setCampaignWizardSheetOpen does not leave create open', () => {
  let createOpen = true;
  let wizardOpen = false;
  setCampaignWizardSheetOpen(
    true,
    (open) => {
      createOpen = open;
    },
    (open) => {
      wizardOpen = open;
    }
  );
  assert.equal(createOpen, false);
  assert.equal(wizardOpen, true);
});

test('setCampaignCreateDialogOpen does not leave wizard open', () => {
  let createOpen = false;
  let wizardOpen = true;
  setCampaignCreateDialogOpen(
    true,
    (open) => {
      createOpen = open;
    },
    (open) => {
      wizardOpen = open;
    }
  );
  assert.equal(createOpen, true);
  assert.equal(wizardOpen, false);
});
