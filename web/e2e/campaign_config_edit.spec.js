import { test, expect } from '@playwright/test';
import {
  ADMIN_USER,
  installDialogAutoAccept,
  mockAuthedSession,
} from './helpers.js';

const CAMPAIGN = {
  id: 'camp-edit-1',
  name: 'Edit me',
  status: 'active',
  budget_limit: '100.00',
  current_spend: '0.00',
  customer_id: 'cust-1',
  pacing_mode: 'ASAP',
  timezone: 'UTC',
  target_url: 'https://old.example/',
  updated_at: '2026-01-01T00:00:00Z',
};

async function mockCampaignGet(page) {
  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    await route.continue();
  });
}

test('campaign config PATCH sends updated name', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignGet(page);

  let patchBody = null;
  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...CAMPAIGN, ...patchBody }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await page.getByLabel('Name').fill('Renamed campaign');
  await page.getByRole('button', { name: 'Save configuration' }).click();
  await expect(page.getByText('Campaign configuration updated')).toBeVisible();
  expect(patchBody?.name).toBe('Renamed campaign');
});

test('campaign overview tab shows budget fields', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignGet(page);

  await page.goto('/campaigns/camp-edit-1');
  await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Edit me' })).toBeVisible();
  await expect(page.getByText('$100.00')).toBeVisible();
});
