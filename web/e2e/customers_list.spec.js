import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('customers list renders API rows', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/customers**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (/\/api\/v1\/customers\/[^/?]/.test(url)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [{ id: 'cust-1', name: 'Acme Corp', currency: 'USD', created_at: '2026-01-01T00:00:00Z' }],
        total: 1,
      }),
    });
  });

  await page.goto('/customers');
  await expect(page.getByRole('heading', { name: 'Customers' })).toBeVisible();
  await expect(page.getByRole('grid')).toBeVisible();
  await expect(page.getByText('Acme Corp')).toBeVisible();
});

test('customers list 503 shows error block', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/customers**', async (route) => {
    const url = route.request().url();
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    if (/\/api\/v1\/customers\/[^/?]/.test(url)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ error: { code: 'UNAVAILABLE', message: 'upstream down' } }),
    });
  });

  await page.goto('/customers');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('upstream down')).toBeVisible();
});
