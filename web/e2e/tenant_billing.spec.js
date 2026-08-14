/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, TENANT_USER } from './helpers.js';

test('role U billing locks customer_id', async ({ page }) => {
  await mockAuthedSession(page, TENANT_USER);

  await page.route('**/api/v1/customers/cust-own/wallet', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        balance_micro: 1000000,
        currency: 'USD',
        allowed_overdraft_micro: 0,
        low_balance_threshold_micro: 0,
        payment_provider: 'stripe',
        payment_provider_configured: true,
      }),
    });
  });

  await page.goto('/billing?customer_id=other-customer');
  await expect(page.getByText('cust-own')).toBeVisible();
  await expect(page.locator('input[placeholder="customer_id (UUID)"]')).toHaveCount(0);
});
