import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('campaign flows page lists landers and offers', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/landers**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{ id: 'l-1', name: 'Lander A' }]),
    });
  });
  await page.route('**/api/v1/offers**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{ id: 'o-1', name: 'Offer A' }]),
    });
  });
  await page.route('**/api/v1/flows**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{ id: 'f-1', name: 'Flow A' }]),
    });
  });

  await page.goto('/campaigns/flows');
  await expect(page.getByRole('heading', { name: 'Campaign flows' })).toBeVisible();
  await expect(page.getByText('Lander A')).toBeVisible();
  await page.getByRole('tab', { name: 'Offers' }).click();
  await expect(page.getByText('Offer A')).toBeVisible();
  await page.getByRole('tab', { name: 'Flows' }).click();
  await expect(page.getByText('Flow A')).toBeVisible();
});
