import { test, expect } from '@playwright/test';
import { TENANT_USER, mockAuthedSession } from './helpers.js';

test('role U campaigns load without manual customer uuid', async ({ page }) => {
  await mockAuthedSession(page, TENANT_USER);

  await page.route('**/api/v1/campaigns?**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [{ id: 'c-1', name: 'Scoped', status: 'active', customer_id: 'cust-own' }],
        total: 1,
      }),
    });
  });

  await page.goto('/campaigns?customer_id=cust-own');
  await expect(page.getByText('Scoped')).toBeVisible();
});
