import { test, expect } from '@playwright/test';
import { mockAuthedSession, TENANT_USER } from './helpers.js';

test('role U sees forbidden on other customer detail', async ({ page }) => {
  await mockAuthedSession(page, TENANT_USER);

  await page.route('**/api/v1/customers/other-customer', async (route) => {
    await route.fulfill({
      status: 403,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        error: { code: 'FORBIDDEN', message: 'tenant boundary' },
      }),
    });
  });

  await page.goto('/customers/other-customer');
  await expect(page.getByText('Access denied')).toBeVisible();
  await expect(page.locator('.error-page__code')).toHaveText('403');
});

test('role U customers list redirects to own customer', async ({ page }) => {
  await mockAuthedSession(page, TENANT_USER);

  await page.route('**/api/v1/customers/cust-own', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: 'cust-own',
        name: 'Own Org',
        currency: 'USD',
      }),
    });
  });

  await page.goto('/customers');
  await page.waitForURL('/customers/cust-own');
  await expect(page.getByRole('heading', { name: 'Own Org' })).toBeVisible();
});
