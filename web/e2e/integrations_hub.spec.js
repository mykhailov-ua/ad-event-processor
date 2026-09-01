import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

const INTEGRATIONS_SECTION_PATHS = [
  '/integrations/cost-sync',
  '/integrations/postbacks',
  '/integrations/schemas',
  '/integrations/platform-campaigns',
  '/integrations/affiliate-presets',
];

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('integrations hub shows section navigation', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/integrations');
  await expect(page.getByRole('heading', { name: 'Integrations' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Integrations sections' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Cost sync' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Affiliate presets' })).toBeVisible();
});

test('integrations section links resolve without 404', async ({ page }) => {
  await loginAsAdmin(page);

  for (const path of INTEGRATIONS_SECTION_PATHS) {
    await page.goto(path);
    await expect(page.getByRole('navigation', { name: 'Integrations sections' })).toBeVisible();
    await expect(page.getByText('404')).toHaveCount(0);
    await expect(page.getByText('Page not found')).toHaveCount(0);
  }
});

test('affiliate presets read returns table or empty state', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/integrations/affiliate-presets');
  await expect(page.getByRole('heading', { name: 'Affiliate status presets' })).toBeVisible();

  const table = page.getByRole('table');
  const empty = page.getByText('No presets', { exact: true });
  await expect(table.or(empty)).toBeVisible({ timeout: 15_000 });
});
