import { test, expect } from '@playwright/test';

import {
  fetchSessionCustomerId,
  gotoLive,
  isApiPost,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('apply fraud override posts to API', async ({ page }) => {
  try {
    await loginAsAdmin(page);
  } catch {
    test.skip(true, 'integration: admin login unavailable');
    return;
  }
  const testIp = `203.0.113.${Math.floor(Math.random() * 200) + 10}`;

  await gotoLive(page, '/fraud/overrides');
  await expect(mainHeading(page, 'Fraud overrides')).toBeVisible();

  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    test.skip(true, 'integration: no default_customer_id in session');
    return;
  }

  await page.locator('#override-customer-id').fill(customerId);
  await page.getByRole('button', { name: 'Set customer', exact: true }).click();

  await page.locator('#override-ip').fill(testIp);

  const overridePost = page.waitForResponse(
    (response) =>
      isApiPost('/api/v1/fraud/overrides', 200)(response) &&
      response.url().includes(`customer_id=${encodeURIComponent(customerId)}`),
    { timeout: 20_000 },
  );

  await page.getByRole('button', { name: 'Apply override' }).click();

  await overridePost;

  await expect(page.getByText('Override accepted by the API.', { exact: true })).toBeVisible({
    timeout: 15_000,
  });
});
