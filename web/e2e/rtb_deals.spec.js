import { test, expect } from '@playwright/test';
import { mockAuthedSession } from './helpers.js';

const RTB_USER = {
  id: 'rtb-admin-1',
  email: 'rtb@test.local',
  role: 'A',
  customer_id: '',
  permissions: ['rtb:read', 'rtb:write', 'settings:write', 'campaigns:read'],
};

const DEALS = [
  {
    id: 1,
    deal_id: 'deal-alpha',
    floor_micro: 1_000_000,
    geo_mask: 0,
    cat_mask: 0,
    pacing: 'even',
    seats: 2,
    customer_id: 'cust-1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-02T00:00:00Z',
  },
  {
    id: 2,
    deal_id: 'deal-beta',
    floor_micro: 2_000_000,
    geo_mask: 0,
    cat_mask: 0,
    pacing: 'asap',
    seats: 1,
    customer_id: 'cust-2',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-03T00:00:00Z',
  },
];

test('RTB deals lists rows, creates, and deletes', async ({ page }) => {
  await mockAuthedSession(page, RTB_USER);

  let deals = [...DEALS];
  let createCalled = false;
  let deleteCalled = false;

  await page.route('**/api/v1/rtb/deals**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    if (method === 'GET' && !url.match(/\/deals\/\d+/)) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(deals),
      });
      return;
    }

    if (method === 'POST') {
      createCalled = true;
      const body = route.request().postDataJSON();
      deals = [...deals, {
        id: 3,
        deal_id: body.deal_id,
        floor_micro: body.floor_micro,
        geo_mask: 0,
        cat_mask: 0,
        pacing: body.pacing ?? 'even',
        seats: body.seats ?? 1,
        customer_id: body.customer_id,
        created_at: '2026-01-04T00:00:00Z',
        updated_at: '2026-01-04T00:00:00Z',
      }];
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(deals[deals.length - 1]),
      });
      return;
    }

    if (method === 'DELETE' && url.includes('/deals/1')) {
      deleteCalled = true;
      deals = deals.filter((d) => d.id !== 1);
      await route.fulfill({ status: 204, body: '' });
      return;
    }

    await route.continue();
  });

  await page.goto('/rtb/deals');
  await expect(page.getByTestId('rtb-deals-table')).toBeVisible();
  await expect(page.getByText('deal-alpha')).toBeVisible();
  await expect(page.getByText('deal-beta')).toBeVisible();

  await page.getByTestId('rtb-deal-create-btn').click();
  await expect(page.getByTestId('rtb-deal-modal')).toBeVisible();
  await page.locator('#deal-id').fill('deal-new');
  await page.locator('#deal-customer').fill('cust-3');
  await page.locator('#deal-floor').fill('500000');
  await page.getByTestId('rtb-deal-modal').getByRole('button', { name: 'Save' }).click();
  await page.getByRole('dialog').filter({ hasText: 'Confirm action' }).getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => createCalled).toBe(true);
  await expect(page.getByTestId('rtb-deals-table').getByRole('cell', { name: 'deal-new' })).toBeVisible();

  await page.getByTestId('rtb-deal-delete-1').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByLabel('Type DELETE to confirm').fill('DELETE');
  await page.getByLabel('I understand the consequences').check();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => deleteCalled).toBe(true);
  await expect(page.getByTestId('rtb-deals-table').getByText('deal-alpha')).toHaveCount(0);
});
