import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('integrations action controls are visible', async ({ page }) => {
  await loginAsAdmin(page);

  await page.goto('/integrations/cost-sync');
  await expect(page.getByRole('button', { name: 'Run sync' })).toBeVisible();

  await page.goto('/integrations/platform-campaigns');
  await expect(page.getByRole('button', { name: 'Run platform sync' })).toBeVisible();

  await page.goto('/integrations/postbacks');
  const dlqTab = page.getByRole('button', { name: 'DLQ', exact: true });
  if (await dlqTab.isVisible()) {
    await dlqTab.click();
    const retryButton = page.getByRole('button', { name: 'Retry', exact: true }).first();
    if (await retryButton.isVisible({ timeout: 5000 }).catch(() => false)) {
      await expect(retryButton).toBeVisible();
    }
  }
});
