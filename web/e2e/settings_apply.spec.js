import { test, expect } from '@playwright/test';
import {
  ADMIN_USER,
  PLATFORM_VIEW,
  installDialogAutoAccept,
  mockAuthedSession,
} from './helpers.js';

test('settings apply once with destructive confirm after restart_required', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/settings/platform**', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          ...PLATFORM_VIEW,
          restart_required: ['tracker'],
        }),
      });
      return;
    }
    await route.continue();
  });

  let applied = false;
  await page.route('**/api/v1/settings/platform/apply**', async (route) => {
    applied = true;
    await route.fulfill({ status: 204, body: '' });
  });

  await page.goto('/settings');
  await expect(page.getByText('Restart required')).toBeVisible();
  await page.getByRole('button', { name: 'Apply to disk' }).click();
  await expect.poll(() => applied).toBe(true);
});

test('settings read-only without settings:write', async ({ page }) => {
  const readOnly = {
    ...ADMIN_USER,
    permissions: ADMIN_USER.permissions.filter((p) => p !== 'settings:write'),
  };
  await mockAuthedSession(page, readOnly);

  await page.route('**/api/v1/settings/platform**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(PLATFORM_VIEW),
    });
  });

  await page.goto('/settings');
  await expect(page.getByText('Read-only session')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save' })).toHaveCount(0);
});
