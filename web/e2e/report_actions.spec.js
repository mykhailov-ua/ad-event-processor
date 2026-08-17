// CPA-M3 report row actions (mock API).
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440001';
const CAMPAIGN_ID = '550e8400-e29b-41d4-a716-446655440099';

test('source-quality pause campaign invokes API before UI success', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let sawPause = false;

  await page.route('**/api/v1/reports/source-quality**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        rows: [{
          placement_id: 'facebook-feed',
          campaign_id: CAMPAIGN_ID,
          clicks: 120,
          conversions: 4,
          ivt_rate: 0.12,
          roi_pct: 15.5,
          spend_micro: 2500000,
        }],
        freshness: { as_of: '2026-08-01T00:00:00Z', consistency: 'eventual', stale: false },
      }),
    });
  });

  await page.route(`**/api/v1/selfserve/campaigns/${CAMPAIGN_ID}/pause`, async (route) => {
    sawPause = true;
    await route.fulfill({ status: 200, body: '{}' });
  });

  await page.goto(`/reports/source-quality?customer_id=${CUSTOMER_ID}`);
  await page.getByTestId('report-row-actions-toggle').click();
  await page.getByTestId('report-action-pause').click();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => sawPause).toBe(true);
});

test('traffic-sources compare sends compare=previous query param', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let sawCompare = false;
  await page.route('**/api/v1/reports/traffic-sources**', async (route) => {
    sawCompare = route.request().url().includes('compare=previous');
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        rows: [{
          channel: 'paid_search',
          impressions: 1000,
          clicks: 50,
          spend_micro: 500000,
          roi_pct: 12,
          compare: { spend_micro_delta: 50000, clicks_delta: 5 },
        }],
        freshness: { as_of: '2026-08-01T00:00:00Z', consistency: 'eventual', stale: false },
      }),
    });
  });

  await page.goto(`/reports/traffic-sources?customer_id=${CUSTOMER_ID}`);
  await page.getByLabel('Compare with previous period').check();
  await page.getByRole('button', { name: 'Load' }).click();

  await expect.poll(() => sawCompare).toBe(true);
  await expect(page.getByText('Δ spend')).toBeVisible();
});
