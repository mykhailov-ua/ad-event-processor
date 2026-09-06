import { test, expect } from '@playwright/test';

import {
  ADMIN_SMOKE_ROUTE_READS,
  fetchSessionCustomerId,
  gotoLive,
  gotoLiveAwaitGet,
  gotoLiveAwaitResponse,
  gotoLiveTeam,
  isCampaignsListResponse,
  loginAsAdmin,
  loginSignInHeading,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('install onboarding routes load headings', async ({ page }) => {
  await page.goto('/setup');

  const setupHeading = page.getByRole('heading', { name: 'Initial setup' });
  const signInHeading = loginSignInHeading(page);

  if (await setupHeading.isVisible({ timeout: 5000 }).catch(() => false)) {
    await expect(setupHeading).toBeVisible();
    return;
  }

  await expect(signInHeading).toBeVisible();
});

test('key admin routes load page headings and primary GET reads', async ({ page }) => {
  await loginAsAdmin(page);
  const customerId = await fetchSessionCustomerId(page);

  for (const route of ADMIN_SMOKE_ROUTE_READS) {
    const { path, heading, api, skipWithoutCustomer, useCampaignsList } = route;
    if (skipWithoutCustomer && !customerId) {
      continue;
    }

    let response;
    if (useCampaignsList) {
      response = await gotoLiveAwaitResponse(page, path, isCampaignsListResponse);
    } else if (path.startsWith('/team')) {
      const loaded = await gotoLiveTeam(page, customerId);
      if (!loaded) {
        test.skip(true, 'integration: team overview requires customer_id');
        return;
      }
      continue;
    } else {
      response = await gotoLiveAwaitGet(page, path, api);
    }

    await expect(mainHeading(page, heading)).toBeVisible({ timeout: 15_000 });
    expect(response.ok()).toBe(true);
    const body = await response.json();
    expect(body).toBeTruthy();
  }

  await gotoLive(page, '/fraud');
  await expect(mainHeading(page, 'Fraud')).toBeVisible({ timeout: 15_000 });
});
