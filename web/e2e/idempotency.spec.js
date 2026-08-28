import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession, mockPlatformSettings } from './helpers.js';

test('settings save sends one PATCH per confirm accept', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);
  await mockPlatformSettings(page);

  let patchCount = 0;
  await page.route('**/api/v1/settings/platform**', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          config: { tracking_domain: 'a.example', default_currency: 'USD', timezone: 'UTC' },
          bootstrap_complete: true,
          restart_required: [],
        }),
      });
      return;
    }
    if (method === 'PATCH') {
      patchCount += 1;
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          config: { tracking_domain: 'b.example', default_currency: 'USD', timezone: 'UTC' },
          bootstrap_complete: true,
          restart_required: [],
        }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/settings');
  await page.getByLabel('Tracking domain').fill('b.example');
  await page.getByRole('button', { name: 'Save' }).click();
  await expect.poll(() => patchCount).toBe(1);
});

test('campaign bulk pause sends POST with ids', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (/\/api\/v1\/campaigns\/[^/?]+/.test(url)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [{ id: 'c-pause', name: 'Pause me', status: 'active', customer_id: 'cust-1' }],
        total: 1,
      }),
    });
  });

  let bulkBody = null;
  await page.route('**/api/v1/campaigns/bulk**', async (route) => {
    bulkBody = route.request().postDataJSON();
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ updated: 1 }),
    });
  });

  await page.goto('/campaigns');
  await page.locator('input[type=checkbox]').first().check();
  await page.getByRole('region', { name: 'Bulk actions' }).getByRole('button', { name: 'Pause' }).click();
  expect(bulkBody?.action).toBe('pause');
  expect(bulkBody?.campaign_ids).toContain('c-pause');
});
