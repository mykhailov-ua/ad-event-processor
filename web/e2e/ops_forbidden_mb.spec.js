import { test, expect } from '@playwright/test';

import { loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('ops home shows blocking error when media buyer lacks shards:read', async ({ page }) => {
  await loginAsAdmin(page);

  const forbiddenResponse = page.waitForResponse(
    (response) =>
      response.request().method() === 'GET' &&
      response.url().includes('/api/v1/ops/home') &&
      response.status() === 403,
    { timeout: 20_000 }
  );

  await page.goto('/ops?admin_dev=1&admin_dev_role=MB');

  const response = await forbiddenResponse;
  const body = await response.json();
  expect(body.error?.code).toBe('FORBIDDEN');

  await expect(page.getByText('Could not load ops snapshot')).toBeVisible({ timeout: 15_000 });
  await expect(page.getByRole('heading', { name: 'Ops' })).toHaveCount(0);
});
