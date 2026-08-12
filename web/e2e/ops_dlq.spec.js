import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const DLQ_ROWS = {
  items: [
    {
      id: 'shard-1-1700000000000-0',
      shard_id: 1,
      stream_id: 'events:ch',
      entry_id: '1700000000000-0',
      campaign_id: 'camp-1',
      event_type: 'click',
      error: 'timeout',
      failed_at: '2026-08-12T10:00:00Z',
      retry_count: 2,
    },
  ],
  next_cursor: '',
};

const READ_ONLY_USER = {
  ...ADMIN_USER,
  permissions: ADMIN_USER.permissions.filter((p) => p !== 'shards:write'),
};

/**
 * @param {import('@playwright/test').Page} page
 */
async function mockOpsShellApis(page) {
  await page.route('**/api/v1/ops/doctor', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ overall: 'ok', checks: [] }),
    });
  });

  await page.route('**/api/v1/ops/incidents', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ partial: false, shards: [], outbox: { pending: 0 } }),
    });
  });

  await page.route('**/api/v1/ops/dashboard/summary', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        outbox_pending: 0,
        rps_estimate: 0,
        emergency_breaker: 'closed',
        services: [],
      }),
    });
  });

  await page.route('**/api/v1/ops/rum', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ events: [] }),
    });
  });

  await page.route('**/api/v1/dashboards/operator', async (route) => {
    await route.fulfill({ status: 200, body: JSON.stringify(null) });
  });
}

test('ops DLQ tab lists entries and retries with confirm', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockOpsShellApis(page);

  let retryCalled = false;

  await page.route('**/api/v1/ops/dlq**', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(DLQ_ROWS),
    });
  });

  await page.route('**/api/v1/ops/dlq/shard-1-1700000000000-0/retry', async (route) => {
    retryCalled = true;
    await route.fulfill({ status: 202, body: '' });
  });

  await page.setViewportSize({ width: 1600, height: 900 });
  await page.goto('/ops');
  await page.getByRole('tab', { name: 'DLQ' }).click();
  await expect(page.getByTestId('ops-dlq-tab')).toBeVisible();
  await expect(page.getByRole('cell', { name: 'timeout' })).toBeVisible();
  await expect(page.getByText('camp-1')).toBeVisible();

  await page.getByTestId('ops-dlq-retry-shard-1-1700000000000-0').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => retryCalled).toBe(true);
  await expect(page.getByText('Retry queued')).toBeVisible();
});

test('ops DLQ partial 503 shows stub banner with rows', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockOpsShellApis(page);

  await page.route('**/api/v1/ops/dlq**', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: DLQ_ROWS.items,
        errors: [{ source: 'shard-2', code: 'timeout' }],
      }),
    });
  });

  await page.goto('/ops');
  await page.getByRole('tab', { name: 'DLQ' }).click();
  await expect(page.getByText('Partial shard errors: shard-2 (timeout)')).toBeVisible();
  await expect(page.getByRole('cell', { name: 'timeout' })).toBeVisible();
});

test('ops DLQ read-only hides retry column', async ({ page }) => {
  await mockAuthedSession(page, READ_ONLY_USER);
  await mockOpsShellApis(page);

  await page.route('**/api/v1/ops/dlq**', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(DLQ_ROWS),
    });
  });

  await page.goto('/ops');
  await page.getByRole('tab', { name: 'DLQ' }).click();
  await expect(page.getByTestId('ops-dlq-tab')).toBeVisible();
  await expect(page.getByRole('cell', { name: 'timeout' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Retry' })).toHaveCount(0);
});
