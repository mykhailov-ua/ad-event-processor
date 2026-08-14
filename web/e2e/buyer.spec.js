/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, BUYER_USER } from './helpers.js';

const CAMPAIGN = {
  id: 'camp-buyer-1',
  name: 'Buyer Campaign',
  status: 'ACTIVE',
  customer_id: 'cust-1',
  budget_limit: '$100.00',
  current_spend: '$10.00',
  daily_budget: '$5.00',
  pacing_mode: 'even',
  target_url: 'https://secret.example/click',
  creative_payload: { title: 'Ad' },
};

test('buyer campaign detail hides creative tab', async ({ page }) => {
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
  await expect(page.getByRole('tab', { name: 'Statistics' })).toBeVisible();
  await expect(page.getByRole('tab', { name: 'Configuration' })).toBeVisible();
  await expect(page.getByRole('tab', { name: 'Creative' })).toHaveCount(0);
  await expect(page.getByText('https://secret.example/click')).toHaveCount(0);
});
