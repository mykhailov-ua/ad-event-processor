import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('automation rules list loads on seed stack', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/automation/rules');
  await expect(page.getByRole('heading', { name: 'Automation rules' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Automation sections' })).toBeVisible();

  const customerRequired = page.getByText('Customer required', { exact: true });
  if (await customerRequired.isVisible({ timeout: 3000 }).catch(() => false)) {
    await page.getByRole('button', { name: 'Apply' }).click();
  }

  const table = page.getByRole('table');
  const empty = page.getByText('No automation rules', { exact: true });
  const stub = page.getByText('unavailable', { exact: false });
  await expect(table.or(empty).or(stub)).toBeVisible({ timeout: 15_000 });
});
