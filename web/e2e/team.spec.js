import { test, expect } from '@playwright/test';

import {
  fetchSessionCustomerId,
  gotoLiveTeam,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('team page loads overview from GET /api/v1/team/overview', async ({ page }) => {
  await loginAsAdmin(page);
  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    test.skip(true, 'integration: no default_customer_id in session');
    return;
  }

  const overviewGet = page.waitForResponse(
    (response) =>
      response.request().method() === 'GET' &&
      response.url().includes('/api/v1/team/overview') &&
      response.status() === 200,
    { timeout: 20_000 }
  );
  const loaded = await gotoLiveTeam(page, customerId);
  if (!loaded) {
    test.skip(true, 'integration: team overview requires customer_id');
    return;
  }
  const response = await overviewGet;
  expect(response.ok()).toBe(true);
  await expect(mainHeading(page, 'Team')).toBeVisible();
});
