import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

const CAMPAIGN = {
  id: 'camp-fraud-1',
  name: 'Fraud campaign',
  status: 'active',
  customer_id: 'cust-1',
};

test('fraud tab loads thresholds form', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns/camp-fraud-1', async (route) => {
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

  await page.route('**/api/v1/campaigns/camp-fraud-1/fraud**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        fraud_threshold_pass: 0.2,
        fraud_threshold_suspect: 0.5,
        fraud_threshold_ivt: 0.7,
        fraud_threshold_block: 0.9,
      }),
    });
  });

  await page.goto('/campaigns/camp-fraud-1?tab=fraud');
  await expect(page.getByRole('tab', { name: 'Fraud' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save fraud settings' })).toBeVisible();
});

test('fraud tab PATCH sends threshold update', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns/camp-fraud-1', async (route) => {
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

  let patchBody = null;
  await page.route('**/api/v1/campaigns/camp-fraud-1/fraud**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ fraud_threshold_block: 0.9 }),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(patchBody),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-fraud-1?tab=fraud');
  await page.getByLabel('Threshold block').fill('0.95');
  await page.getByRole('button', { name: 'Save fraud settings' }).click();
  expect(patchBody?.fraud_threshold_block).toBe(0.95);
});
