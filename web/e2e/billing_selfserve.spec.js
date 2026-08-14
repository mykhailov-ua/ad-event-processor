/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, BUYER_USER } from './helpers.js';

test('buyer billing wallet top-up creates payment intent', async ({ page }) => {
  await mockAuthedSession(page, BUYER_USER);

  await page.route('**/api/v1/customers/*/wallet', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        balance_micro: 5_000_000,
        currency: 'USD',
        payment_provider: 'stripe',
        payment_provider_configured: true,
      }),
    });
  });

  let intentBody = null;
  await page.route('**/api/v1/selfserve/payment-intents', async (route) => {
    intentBody = route.request().postDataJSON();
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        intent_id: 'pi-e2e',
        status: 'pending',
        checkout_url: 'https://checkout.example/pi-e2e',
      }),
    });
  });

  await page.goto('/billing');
  await expect(page.getByTestId('billing-selfserve-panel')).toBeVisible();
  await page.getByTestId('billing-topup-amount').fill('50.00');
  await page.getByTestId('billing-topup-submit').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('button', { name: 'Confirm' }).click();

  await expect(page.getByTestId('billing-topup-checkout-link')).toBeVisible();
  expect(intentBody?.amount_micro).toBe(50_000_000);
});

test('buyer billing statement loads summary', async ({ page }) => {
  await mockAuthedSession(page, BUYER_USER);

  await page.route('**/api/v1/customers/*/wallet', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ balance_micro: 0, currency: 'USD' }),
    });
  });

  await page.route('**/api/v1/selfserve/billing/statement*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        opening_balance_micro: 1_000_000,
        closing_balance_micro: 2_500_000,
        currency: 'USD',
        period: { from: '2026-08-01', to: '2026-08-31' },
      }),
    });
  });

  await page.goto('/billing');
  await page.getByTestId('billing-statement-load').click();
  await expect(page.getByText('1.00 USD')).toBeVisible();
  await expect(page.getByText('2.50 USD')).toBeVisible();
});
