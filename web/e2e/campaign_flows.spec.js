import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('campaign flows page lists landers and offers', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/landers', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{ id: 'lander-1', name: 'Lander A', url: 'https://lp.example/a' }]),
    });
  });

  await page.route('**/api/v1/offers', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([{ id: 'offer-1', name: 'Offer X', url: 'https://offer.example/x' }]),
    });
  });

  await page.route('**/api/v1/flows', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([
        {
          id: 'flow-1',
          name: 'Main',
          paths: [
            {
              weight: 100,
              landers: [{ lander_id: 'lander-1', weight: 100 }],
              offers: [{ offer_id: 'offer-1', weight: 100 }],
            },
          ],
        },
      ]),
    });
  });

  await page.goto('/campaigns/flows');
  await expect(page.getByTestId('flow-landers-table')).toContainText('Lander A');
  await page.getByRole('tab', { name: 'Offers' }).click();
  await expect(page.getByTestId('flow-offers-table')).toContainText('Offer X');
  await page.getByRole('tab', { name: 'Flows' }).click();
  await expect(page.getByTestId('flow-flows-table')).toContainText('Main');
});
