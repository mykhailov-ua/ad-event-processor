import { test, expect } from '@playwright/test';

import {
  baseURL,
  gotoLive,
  loginWithCredentials,
  mainContent,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test(
  'media buyer /ops shows forbidden panel and API returns 403',
  { tag: '@L3' },
  async ({ page }) => {
    const email = process.env.ADMIN_E2E_MB_EMAIL || process.env.ADMIN_E2E_MEDIA_BUYER_EMAIL;
    const password =
      process.env.ADMIN_E2E_MB_PASSWORD || process.env.ADMIN_E2E_MEDIA_BUYER_PASSWORD;
    if (!email || !password) {
      test.skip(true, 'integration: set ADMIN_E2E_MB_EMAIL and ADMIN_E2E_MB_PASSWORD');
      return;
    }

    await loginWithCredentials(page, email, password);
    await gotoLive(page, '/ops');

    await expect(mainContent(page).getByText('403', { exact: true })).toBeVisible({
      timeout: 15_000,
    });

    const opsHome = await page.request.get(new URL('/api/v1/ops/home', baseURL).toString());
    expect(opsHome.status()).toBe(403);
  }
);
