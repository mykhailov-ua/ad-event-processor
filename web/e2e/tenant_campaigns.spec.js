/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, TENANT_USER } from './helpers.js';

test('role U campaigns load without manual customer uuid', async ({ page }) => {
  await mockAuthedSession(page, TENANT_USER);

  await page.route('**/api/v1/campaigns*', async (route) => {
    const url = route.request().url();
    expect(url).toContain('customer_id=cust-own');
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [{ id: 'c1', name: 'Tenant Camp', status: 'ACTIVE', customer_id: 'cust-own' }],
        total: 1,
      }),
    });
  });

  await page.goto('/campaigns');
  await expect(page.getByText('Tenant Camp')).toBeVisible();
  await expect(page.locator('input[placeholder="Customer UUID…"]')).toHaveCount(0);
});
