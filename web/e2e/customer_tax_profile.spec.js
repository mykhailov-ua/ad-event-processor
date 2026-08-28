import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('customer tax profile tab loads for admin', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/customers/cust-1', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ id: 'cust-1', name: 'Acme', currency: 'USD' }),
    });
  });

  await page.route('**/api/v1/customers/cust-1/tax-profile**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ country: 'US', vat_id: 'US123' }),
    });
  });

  await page.goto('/customers/cust-1?tab=tax');
  await expect(page.getByLabel('Country code')).toBeVisible();
});

test('tenant tax profile is read-only', async ({ page }) => {
  const { TENANT_USER } = await import('./helpers.js');
  await mockAuthedSession(page, TENANT_USER);

  await page.route('**/api/v1/customers/cust-own', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ id: 'cust-own', name: 'Own', currency: 'USD' }),
    });
  });

  await page.route('**/api/v1/customers/cust-own/tax-profile**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ country: 'DE' }),
    });
  });

  await page.goto('/customers/cust-own?tab=tax');
  await expect(page.getByRole('button', { name: 'Save tax profile' })).toHaveCount(0);
});
