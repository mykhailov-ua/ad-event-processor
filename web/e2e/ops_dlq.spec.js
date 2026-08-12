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

test('ops DLQ tab lists entries and retries with confirm', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let retryCalled = false;

  await page.route('**/api/v1/ops/dlq**', async (route) => {
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

  await page.goto('/ops');
  await page.getByRole('tab', { name: 'DLQ' }).click();
  await expect(page.getByTestId('ops-dlq-tab')).toBeVisible();
  await expect(page.getByText('timeout')).toBeVisible();

  await page.getByTestId('ops-dlq-retry-shard-1-1700000000000-0').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => retryCalled).toBe(true);
});
