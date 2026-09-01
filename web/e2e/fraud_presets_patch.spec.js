import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud presets patch table is visible without saving', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/presets');

  await expect(page.getByRole('heading', { name: 'Fraud presets' })).toBeVisible();

  const emptyState = page.getByText('No presets');
  const nameHeader = page.getByRole('columnheader', { name: 'Name' });
  await expect(emptyState.or(nameHeader)).toBeVisible({ timeout: 15_000 });

  if (await emptyState.isVisible()) {
    return;
  }

  await expect(nameHeader).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Pass' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Block' })).toBeVisible();

  const saveButton = page.getByRole('button', { name: 'Save', exact: true }).first();
  await expect(saveButton).toBeVisible();
});
