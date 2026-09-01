import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('command palette opens and lists server route entries', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/customers');

  await page.keyboard.press('Control+k');
  await expect(page.getByRole('dialog')).toBeVisible();
  await expect(page.getByLabel('Command palette search')).toBeVisible();

  const forbidden = page.getByText('forbidden', { exact: false });
  if (await forbidden.isVisible({ timeout: 3000 }).catch(() => false)) {
    test.skip(true, 'command palette forbidden for session permissions');
  }

  const routeEntry = page.getByRole('option').first();
  await expect(routeEntry).toBeVisible({ timeout: 15_000 });
});
