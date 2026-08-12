import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const INVOICE = {
  id: 'inv-e2e-1',
  customer_id: 'cust-1',
  status: 'open',
  total_micro: 1000000,
  currency: 'USD',
};

test('void invoice requires strong confirm dialog', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let voidCalled = false;

  await page.route('**/api/v1/billing/invoices/inv-e2e-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(INVOICE),
      });
      return;
    }
    await route.continue();
  });

  await page.route('**/api/v1/billing/invoices/inv-e2e-1/void', async (route) => {
    voidCalled = true;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ ok: true }),
    });
  });

  await page.goto('/billing/invoices/inv-e2e-1');
  await page.getByRole('button', { name: 'Void' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await expect(page.getByText('Void invoice')).toBeVisible();

  const confirmBtn = page.getByRole('button', { name: 'Confirm' });
  await expect(confirmBtn).toBeDisabled();
  expect(voidCalled).toBe(false);

  await page.getByLabel('Type DELETE to confirm').fill('DELETE');
  await page.getByRole('checkbox', { name: 'I understand the consequences' }).check();
  await page.getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => voidCalled).toBe(true);
});
