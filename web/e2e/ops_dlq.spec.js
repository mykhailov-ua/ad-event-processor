import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('ops DLQ tab lists entries and retries with confirm', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/dashboard/summary**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ outbox_pending: 0 }),
    });
  });

  let retried = false;
  await page.route('**/api/v1/ops/dlq**', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          items: [
            {
              id: 'dlq-1',
              shard_id: 0,
              stream_id: 'events',
              entry_id: '1-0',
              error: 'timeout',
              retry_count: 1,
            },
          ],
          partial: false,
        }),
      });
      return;
    }
    if (method === 'POST' && route.request().url().includes('/retry')) {
      retried = true;
      await route.fulfill({ status: 204, body: '' });
      return;
    }
    await route.continue();
  });

  await page.goto('/ops?tab=dlq');
  await expect(page.getByText('dlq-1')).toBeVisible();
  await page.getByRole('button', { name: 'Retry' }).first().click();
  await expect.poll(() => retried).toBe(true);
});

test('ops DLQ partial snapshot shows banner', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/dashboard/summary**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ outbox_pending: 0 }),
    });
  });

  await page.route('**/api/v1/ops/dlq**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], partial: true }),
    });
  });

  await page.goto('/ops?tab=dlq');
  await expect(page.getByText('Partial DLQ snapshot')).toBeVisible();
});

test('ops DLQ inbox page loads', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/dlq**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (url.includes('/inbox')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], next_cursor: null }),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], partial: false }),
    });
  });

  await page.goto('/ops/dlq');
  await expect(page.getByTestId('ops-dlq-page')).toBeVisible();
});
