import { test, expect } from '@playwright/test';

import {
  fetchSessionCustomerId,
  gotoLive,
  loginAsAdmin,
  mainContent,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.describe('frozen hub deep links', { tag: '@freeze' }, () => {
  test.beforeEach(async ({}, testInfo) => {
    await skipUnlessIntegrationReady(testInfo);
  });

  test('fraud presets deep link redirects to exports hub', async ({ page }) => {
    await loginAsAdmin(page);
    await gotoLive(page, '/fraud/presets');
    await expect(page).toHaveURL(/\/exports(?:\?|$)/);
  });

  test('rtb deals deep link redirects to exports hub', async ({ page }) => {
    await loginAsAdmin(page);
    await gotoLive(page, '/rtb/deals');
    await expect(page).toHaveURL(/\/exports(?:\?|$)/);
  });

  test('creative hub deep link redirects to campaigns', async ({ page }) => {
    await loginAsAdmin(page);
    await gotoLive(page, '/creative');
    await expect(page).toHaveURL(/\/campaigns(?:\?|$)/);
  });

  test('click log report stays on export stub instead of blind exports redirect', async ({
    page,
  }) => {
    await loginAsAdmin(page);
    const customerId = await fetchSessionCustomerId(page);
    if (!customerId) {
      test.skip(true, 'integration: click-log export stub requires session customer_id');
      return;
    }

    await gotoLive(page, `/reports/click-log?customer_id=${encodeURIComponent(customerId)}`);
    await expect(page).toHaveURL(/\/reports\/click-log(?:\?|$)/);
    await expect(
      mainContent(page).getByText('Control Plane export mode', { exact: true })
    ).toBeVisible({ timeout: 15_000 });
  });
});
