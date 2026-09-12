import { test, expect } from '@playwright/test';

import {
  baseURL,
  getAdminCredentials,
  loginWithCredentials,
  mainNav,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test(
  'nav updates after admin grants role without re-login',
  { tag: '@L3' },
  async ({ browser }) => {
    const mbEmail = process.env.ADMIN_E2E_MB_EMAIL || process.env.ADMIN_E2E_MEDIA_BUYER_EMAIL;
    const mbPassword =
      process.env.ADMIN_E2E_MB_PASSWORD || process.env.ADMIN_E2E_MEDIA_BUYER_PASSWORD;
    if (!mbEmail || !mbPassword) {
      test.skip(true, 'integration: set ADMIN_E2E_MB_EMAIL and ADMIN_E2E_MB_PASSWORD');
      return;
    }

    const { email: adminEmail, password: adminPassword } = getAdminCredentials();
    const mbContext = await browser.newContext();
    const adminContext = await browser.newContext();
    const mbPage = await mbContext.newPage();
    const adminPage = await adminContext.newPage();

    try {
      await loginWithCredentials(mbPage, mbEmail, mbPassword);
      await expect(mainNav(mbPage).getByRole('link', { name: 'Audit' })).toHaveCount(0);

      const auditRes = await mbPage.request.get(new URL('/api/v1/audit', baseURL).toString());
      expect(auditRes.status()).toBe(403);

      const bootstrapRes = await mbPage.request.get(
        new URL('/api/v1/session/bootstrap', baseURL).toString()
      );
      expect(bootstrapRes.ok()).toBeTruthy();
      const bootstrap = await bootstrapRes.json();
      const memberId = bootstrap.user?.id;
      const customerId = bootstrap.user?.customer_id;
      if (!memberId || !customerId) {
        throw new Error('session bootstrap missing user id or customer_id');
      }

      await loginWithCredentials(adminPage, adminEmail, adminPassword);
      const grantRes = await adminPage.request.patch(
        new URL(
          `/api/v1/team/members/${encodeURIComponent(memberId)}?customer_id=${encodeURIComponent(customerId)}`,
          baseURL
        ).toString(),
        { data: { role: 'M' } }
      );
      expect(grantRes.ok()).toBeTruthy();

      await mbPage.evaluate(() => {
        document.dispatchEvent(new Event('visibilitychange'));
      });

      await expect(mainNav(mbPage).getByRole('link', { name: 'Audit' })).toBeVisible({
        timeout: 15_000,
      });
    } finally {
      const bootstrapRes = await mbPage.request.get(
        new URL('/api/v1/session/bootstrap', baseURL).toString()
      );
      if (bootstrapRes.ok()) {
        const bootstrap = await bootstrapRes.json();
        const memberId = bootstrap.user?.id;
        const customerId = bootstrap.user?.customer_id;
        if (memberId && customerId) {
          await adminPage.request.patch(
            new URL(
              `/api/v1/team/members/${encodeURIComponent(memberId)}?customer_id=${encodeURIComponent(customerId)}`,
              baseURL
            ).toString(),
            { data: { role: 'MB' } }
          );
        }
      }
      await mbContext.close();
      await adminContext.close();
    }
  }
);
