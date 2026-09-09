import { test, expect } from '@playwright/test';

import { gotoLive, loginAsAdmin, mainHeading, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings page shows license state and apply form', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/settings');
  await expect(mainHeading(page, 'Settings')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Apply license' })).toBeVisible();
});
