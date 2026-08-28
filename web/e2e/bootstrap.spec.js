import { test, expect } from '@playwright/test';
import {
  ADMIN_USER,
  PLATFORM_VIEW,
  installDialogAutoAccept,
  mockAuthedSession,
} from './helpers.js';

test('platform settings shows bootstrap incomplete badge', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/settings/platform**', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ ...PLATFORM_VIEW, bootstrap_complete: false }),
    });
  });

  await page.goto('/settings');
  await expect(page.getByText('Bootstrap incomplete')).toBeVisible();
});

test('meta bootstrap flag does not block login form', async ({ page }) => {
  await page.route('**/api/v1/meta', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ bootstrap_complete: false, version: 'e2e' }),
    });
  });

  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'ad-event-processor' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
});
