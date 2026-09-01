import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('team member edit table is visible', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/team');

  await expect(page.getByRole('heading', { name: 'Team' })).toBeVisible();

  const emptyMembers = page.getByText('Team roster is empty for this customer.');
  const emailHeader = page.getByRole('columnheader', { name: 'Email' });
  await expect(emptyMembers.or(emailHeader)).toBeVisible({ timeout: 15_000 });

  if (await emptyMembers.isVisible()) {
    return;
  }

  await expect(emailHeader).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Role' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Spend cap' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: 'Blocked' })).toBeVisible();

  const saveButton = page.getByRole('button', { name: 'Save', exact: true }).first();
  await expect(saveButton).toBeVisible();
});
