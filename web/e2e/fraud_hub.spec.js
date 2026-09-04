import { test, expect } from '@playwright/test';

import { hubCardLink, loginAsAdmin, mainHeading, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud hub links are visible', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/fraud');
  await expect(mainHeading(page, 'Fraud')).toBeVisible();
  await expect(hubCardLink(page, '/fraud/integrations')).toBeVisible();
  await expect(hubCardLink(page, '/fraud/labels')).toBeVisible();
});
