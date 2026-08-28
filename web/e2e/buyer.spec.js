import { test, expect } from '@playwright/test';
import { BUYER_USER, mockAuthedSession } from './helpers.js';

const CAMPAIGN = {
  id: 'camp-buyer-1',
  name: 'Masked campaign',
  status: 'active',
  customer_id: 'cust-1',
  target_url: 'https://secret.example/track',
};

test('buyer sees masked campaign tabs only', async ({ page }) => {
  await mockAuthedSession(page, BUYER_USER);

  await page.route('**/api/v1/campaigns/camp-buyer-1', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(CAMPAIGN),
    });
  });

  await page.goto('/campaigns/camp-buyer-1');
  await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible();
  await expect(page.getByRole('tab', { name: 'Configuration' })).toBeVisible();
  await expect(page.getByRole('tab', { name: 'Fraud' })).toHaveCount(0);
  await expect(page.getByRole('tab', { name: 'Postbacks' })).toHaveCount(0);
});
