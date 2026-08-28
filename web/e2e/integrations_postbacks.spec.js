import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('postbacks DLQ retry POST before toast', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/postbacks/campaign-status**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([]),
    });
  });

  await page.route('**/api/v1/postbacks/dlq**', async (route) => {
    const url = route.request().url();
    if (route.request().method() === 'GET' && !url.includes('/retry')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([
          {
            id: 42,
            campaign_id: 'c-1',
            event_type: 'conversion',
            failures_count: 2,
            status: 'PENDING',
          },
        ]),
      });
      return;
    }
    await route.continue();
  });

  let retried = false;
  await page.route('**/api/v1/postbacks/dlq/42/retry**', async (route) => {
    retried = true;
    await route.fulfill({ status: 204, body: '' });
  });

  await page.goto('/integrations/postbacks');
  await expect(page.getByTestId('integrations-postbacks-page')).toBeVisible();
  await page.getByTestId('postback-dlq-retry-42').click();
  await expect.poll(() => retried).toBe(true);
});
