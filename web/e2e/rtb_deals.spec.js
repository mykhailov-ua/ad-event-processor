import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('RTB deals lists rows, creates, and deletes', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  let deals = [{ deal_id: 'deal-1', customer_id: 'cust-1', floor_micro: 1000 }];

  await page.route('**/api/v1/rtb/deals**', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(deals),
      });
      return;
    }
    if (method === 'POST') {
      deals = [...deals, { deal_id: 'deal-2', customer_id: 'cust-1', floor_micro: 2000 }];
      await route.fulfill({
        status: 201,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(deals[deals.length - 1]),
      });
      return;
    }
    if (method === 'DELETE') {
      deals = deals.filter((d) => d.deal_id !== 'deal-1');
      await route.fulfill({ status: 204, body: '' });
      return;
    }
    await route.continue();
  });

  await page.goto('/rtb/deals');
  await expect(page.getByRole('heading', { name: 'RTB deals' })).toBeVisible();
  await expect(page.getByText('deal-1')).toBeVisible();

  await page.getByRole('button', { name: 'Create deal' }).click();
  await page.getByLabel('Deal ID').fill('deal-2');
  await page.getByLabel('Customer ID').fill('cust-1');
  await page.getByRole('button', { name: 'Create', exact: true }).click();
  await expect(page.getByRole('gridcell', { name: 'deal-2' })).toBeVisible();
});
