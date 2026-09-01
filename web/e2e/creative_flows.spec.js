import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('flows list loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/flows');
  await expect(page.getByRole('heading', { name: 'Flows' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Creative sections' })).toBeVisible();

  const table = page.getByRole('table');
  const empty = page.getByText('No flows', { exact: true });
  await expect(table.or(empty)).toBeVisible({ timeout: 15_000 });
  await expect(page.getByRole('button', { name: 'Create flow' })).toBeVisible();
});

test('landers list loads', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/landers');
  await expect(page.getByRole('heading', { name: 'Landers' })).toBeVisible();

  const table = page.getByRole('table');
  const empty = page.getByText('No landers', { exact: true });
  await expect(table.or(empty)).toBeVisible({ timeout: 15_000 });
});
