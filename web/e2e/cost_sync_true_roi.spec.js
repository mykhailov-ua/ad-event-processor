import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('cost sync view loads credentials and history', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  const customerId = '550e8400-e29b-41d4-a716-446655440000';

  await page.route('**/api/v1/cost-sync/credentials**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{
        customer_id: customerId,
        network: 'facebook',
        account_id: 'act_1',
        has_access_token: true,
      }]),
    });
  });

  await page.route('**/api/v1/cost-sync/history**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{
        id: 1,
        network: 'facebook',
        status: 'SUCCESS',
        rows_imported: 3,
        total_amount_usd_micro: 1_500_000,
      }]),
    });
  });

  await page.goto(`/integrations/cost-sync?customer_id=${customerId}`);
  await expect(page.getByTestId('cost-sync-view')).toBeVisible({ timeout: 15000 });
  await expect(page.getByRole('heading', { name: 'Cost Sync' })).toBeVisible();
  await expect(page.locator('[data-testid="cost-sync-view"]').getByText('facebook').first()).toBeVisible();
  await expect(page.getByText('SUCCESS')).toBeVisible();
});

test('true roi report view mounts', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/reports/true-roi**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        rows: [{
          campaign_id: 'camp-1',
          ad_spend_micro: 100_000_000,
          revenue_micro: 150_000_000,
          true_profit_micro: 50_000_000,
          true_roi_pct: 50,
          true_cpa_micro: 10_000_000,
          conversions: 10,
        }],
        freshness: { source: 'clickhouse', lag_seconds: 0 },
      }),
    });
  });

  await page.goto('/reports/true-roi?customer_id=550e8400-e29b-41d4-a716-446655440000');
  await expect(page.getByRole('heading', { name: 'True ROI' })).toBeVisible({ timeout: 15000 });
  await page.getByRole('button', { name: 'Load' }).click();
  await expect(page.getByText('Ad Spend')).toBeVisible();
  await expect(page.getByText('True Profit')).toBeVisible();
});
