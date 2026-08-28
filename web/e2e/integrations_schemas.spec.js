import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('schemas page lists rows', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/integration/schemas**', async (route) => {
    const url = route.request().url();
    if (route.request().method() === 'GET' && url.endsWith('/api/v1/integration/schemas')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([{ id: 'schema-1', name: 'Custom', version: 1, kind: 'inbound_tokens' }]),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/integrations/schemas');
  await expect(page.getByTestId('integrations-schemas-page')).toBeVisible();
  await expect(page.getByTestId('schema-row-schema-1')).toBeVisible();
});

test('billing summary strip on billing page for admin', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/invoices**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.route('**/api/v1/billing/summary**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ invoice_count: 3, open_micro: 1_000_000 }),
    });
  });

  await page.goto('/billing');
  await expect(page.getByRole('heading', { name: 'Billing' })).toBeVisible();
});
