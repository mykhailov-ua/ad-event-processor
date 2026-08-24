import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CAMPAIGN_ID = '550e8400-e29b-41d4-a716-446655440000';
const DLQ_ID = 42;

test.describe('Integrations postbacks page', () => {
  test('postback DLQ retry POST before toast', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    let retryPosted = false;

    await page.route('**/api/v1/postbacks/dlq', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([
          {
            id: DLQ_ID,
            campaign_id: CAMPAIGN_ID,
            event_type: 'conversion',
            failures_count: 2,
            status: 'FAILED',
            last_error: 'timeout',
          },
        ]),
      });
    });

    await page.route('**/api/v1/postbacks/campaign-status', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([
          {
            campaign_id: CAMPAIGN_ID,
            provider: 'webhook',
            dlq_pending_count: 1,
          },
        ]),
      });
    });

    await page.route(`**/api/v1/postbacks/dlq/${DLQ_ID}/retry`, async (route) => {
      retryPosted = route.request().method() === 'POST';
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ status: 'ok' }),
      });
    });

    await page.goto('/integrations/postbacks');
    await expect(page.getByTestId('integrations-postbacks-page')).toBeVisible();
    await expect(page.getByTestId(`postback-dlq-row-${DLQ_ID}`)).toBeVisible();

    await page.getByTestId(`postback-dlq-retry-${DLQ_ID}`).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
    await expect.poll(() => retryPosted).toBe(true);
    await expect(page.getByText('Retry queued')).toBeVisible();
  });
});
