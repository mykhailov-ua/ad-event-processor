import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('fraud labels page loads manual label form', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/fraud/labels**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0 }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/fraud/labels?customer_id=cust-1');
  await expect(page.getByTestId('fraud-labels-page')).toBeVisible();
});

test('fraud decisions explain lookup renders form', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.goto('/fraud/decisions?customer_id=cust-1');
  await expect(page.getByTestId('fraud-decisions-page')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Explain' })).toBeVisible();
});
