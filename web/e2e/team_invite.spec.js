import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('team invite form is visible', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/team');

  await expect(page.getByRole('heading', { name: 'Team' })).toBeVisible();
  await expect(page.getByLabel('Email', { exact: true })).toBeVisible();
  await expect(page.getByLabel('Role', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Send invite' })).toBeVisible();
});
