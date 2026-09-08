import { test, expect } from '@playwright/test';

import {
  fetchSessionCustomerId,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud decision explain calls GET /api/v1/fraud/decisions', async ({ page }, testInfo) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/decisions');
  await expect(page.getByRole('heading', { name: 'Fraud decision explain' })).toBeVisible();

  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    testInfo.skip(true, 'integration: session has no default customer_id');
    return;
  }

  await page.locator('#decision-customer-id').fill(customerId);
  await page.locator('#decision-ip-hash').fill('00000000000000000000000000000000');

  const decisionGet = page.waitForResponse(
    (response) =>
      response.request().method() === 'GET' &&
      response.url().includes('/api/v1/fraud/decisions'),
    { timeout: 30_000 }
  );
  await page.getByRole('button', { name: 'Explain', exact: true }).click();
  const response = await decisionGet;

  if (response.status() === 404) {
    await expect(page.getByText('Could not explain fraud decision')).toBeVisible({
      timeout: 15_000,
    });
    return;
  }

  expect(response.ok()).toBe(true);
  const body = await response.json();
  expect(body).toHaveProperty('ip_hash');
  await expect(page.getByText('Tier', { exact: true })).toBeVisible({ timeout: 15_000 });
});
