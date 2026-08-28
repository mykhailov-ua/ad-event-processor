import { test, expect } from '@playwright/test';
import {
  ADMIN_USER,
  PLATFORM_VIEW,
  installDialogAutoAccept,
  mockAuthedSession,
  mockLoginSuccess,
  mockPlatformSettings,
} from './helpers.js';

test('login shows session expired reason', async ({ page }) => {
  await page.goto('/login?reason=session');
  await expect(page.getByText('Your session expired. Sign in again.')).toBeVisible();
});

test('CSRF from cookie after full page reload', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockLoginSuccess(page);
  await mockPlatformSettings(page);

  await page.goto('/login');
  await page.fill('#login-email', 'admin@test.local');
  await page.fill('#login-password', 'secret');
  await Promise.all([
    page.waitForURL(/\/customers/, { timeout: 15_000 }),
    page.getByRole('button', { name: 'Sign in' }).click(),
  ]);

  await page.context().addCookies([
    {
      name: 'csrfToken',
      value: 'cookie-csrf-token',
      url: 'http://127.0.0.1:4173',
    },
  ]);

  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: ADMIN_USER.id,
        email: ADMIN_USER.email,
        role: ADMIN_USER.role,
        customer_id: ADMIN_USER.customer_id,
        permissions: ADMIN_USER.permissions,
      }),
    });
  });

  let patchHeaders = null;
  await page.route('**/api/v1/settings/platform**', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(PLATFORM_VIEW),
      });
      return;
    }
    if (method === 'PATCH') {
      patchHeaders = route.request().headers();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(PLATFORM_VIEW),
      });
      return;
    }
    await route.continue();
  });

  await page.reload();
  await page.goto('/settings');
  await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();
  await page.getByRole('button', { name: 'Save' }).click();

  expect(patchHeaders?.['x-csrf-token']).toBe('cookie-csrf-token');
});
