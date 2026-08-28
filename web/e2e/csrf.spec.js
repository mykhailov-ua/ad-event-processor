import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession, mockPlatformSettings } from './helpers.js';

test('settings PATCH includes CSRF header', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  let patchHeaders = null;
  await page.route('**/api/v1/settings/platform**', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          config: { tracking_domain: 'track.example', default_currency: 'USD', timezone: 'UTC' },
          bootstrap_complete: true,
          restart_required: [],
        }),
      });
      return;
    }
    if (method === 'PATCH') {
      patchHeaders = route.request().headers();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          config: { tracking_domain: 'track.example', default_currency: 'USD', timezone: 'UTC' },
          bootstrap_complete: true,
          restart_required: [],
        }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/settings');
  await page.getByRole('button', { name: 'Save' }).click();
  expect(patchHeaders?.['x-csrf-token']).toBe('e2e-csrf-token');
});
