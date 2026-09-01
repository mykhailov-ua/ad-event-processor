import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('audit page shows CSV export control', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/audit');

  await expect(page.getByRole('button', { name: 'Export CSV' })).toBeVisible();
  await expect(page.getByText('Redact PII in export')).toBeVisible();
});
