import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('shard 0 catch-up button posts catchup', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/shards**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ shards: [{ shard_id: 0, ping_ok: true }], partial: false }),
      });
      return;
    }
    await route.continue();
  });

  let catchupCalled = false;
  await page.route('**/api/v1/ops/shards/0/catchup**', async (route) => {
    catchupCalled = true;
    await route.fulfill({ status: 204, body: '' });
  });

  await page.goto('/ops/shards');
  await page.getByRole('button', { name: 'Run shard 0 catch-up' }).click();
  await expect.poll(() => catchupCalled).toBe(true);
});

test('shard catch-up surfaces error when worker not configured', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/shards**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ shards: [], partial: false }),
      });
      return;
    }
    await route.continue();
  });

  await page.route('**/api/v1/ops/shards/0/catchup**', async (route) => {
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'UNAVAILABLE', message: 'worker not configured' } }),
    });
  });

  await page.goto('/ops/shards');
  await page.getByRole('button', { name: 'Run shard 0 catch-up' }).click();
  await expect(page.getByText('worker not configured')).toBeVisible();
});
