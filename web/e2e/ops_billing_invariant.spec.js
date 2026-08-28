import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('ops hub shows billing summary strip when summary API returns data', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/dashboard/summary**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ outbox_pending: 0, rps_estimate: 100 }),
    });
  });

  await page.goto('/ops');
  await expect(page.getByTestId('ops-hub-page')).toBeVisible();
  await expect(page.getByText('RPS estimate')).toBeVisible();
});

test('ops hub stub banner when summary returns 501', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/dashboard/summary**', async (route) => {
    await route.fulfill({
      status: 501,
      headers: { 'content-type': 'application/json', 'X-API-Stub': 'true' },
      body: JSON.stringify({ error: { code: 'NOT_IMPLEMENTED', message: 'stub' } }),
    });
  });

  await page.goto('/ops');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('stub')).toBeVisible();
});

test('ops reload RBAC button posts roles reload', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/dashboard/summary**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ outbox_pending: 0 }),
    });
  });

  let reloaded = false;
  await page.route('**/api/v1/ops/roles/reload**', async (route) => {
    reloaded = true;
    await route.fulfill({ status: 204, body: '' });
  });

  await page.goto('/ops');
  await page.getByRole('button', { name: 'Reload RBAC' }).click();
  await expect.poll(() => reloaded).toBe(true);
});
