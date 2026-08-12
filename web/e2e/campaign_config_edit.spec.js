import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CAMPAIGN = {
  id: 'camp-edit-1',
  name: 'Edit me',
  status: 'active',
  budget_limit: '100.00',
  current_spend: '0.00',
  customer_id: 'cust-1',
  pacing_mode: 'ASAP',
  daily_budget: '50.00',
  timezone: 'UTC',
  freq_limit: 3,
  freq_window: 3600,
  target_countries: ['US'],
  target_url: 'https://old.example/',
  daypart_hours: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

test('campaign config PATCH sends daily_budget_micro and geo', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

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

  await page.route('**/api/v1/campaigns/camp-edit-1/**', async (route) => {
    await route.fulfill({ status: 200, body: JSON.stringify({ items: [], total: 0 }) });
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await page.getByTestId('cfg-daily-budget').fill('75.50');
  await page.getByTestId('cfg-geo').fill('US,CA');
  await page.getByTestId('cfg-target-url').fill('https://new.example/landing');

  await page.getByRole('button', { name: 'Save changes' }).click();
  await page.getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => patchBody).not.toBeNull();
  expect(patchBody.daily_budget_micro).toBe(75_500_000);
  expect(patchBody.target_countries).toEqual(['US', 'CA']);
  expect(patchBody.target_url).toBe('https://new.example/landing');
});
