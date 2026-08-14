/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockLoginSuccess } from './helpers.js';

test('valid login navigates to shell with overview heading', async ({ page }) => {
  await mockLoginSuccess(page);

  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@test.local');
  await page.fill('input[type=password]', 'secret');
  await page.click('button[type=submit]');

  await page.waitForURL('/');
  await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible();
});

test('logout confirms and returns to login page', async ({ page }) => {
  await mockLoginSuccess(page);

  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@test.local');
  await page.fill('input[type=password]', 'secret');
  await page.click('button[type=submit]');
  await page.waitForURL('/');

  await page.route('**/api/v1/auth/logout', async (route) => {
    await route.fulfill({
      status: 204,
      headers: { 'content-type': 'application/json' },
      body: '',
    });
  });

  await page.getByRole('button', { name: 'Logout' }).click();
  await page.getByRole('button', { name: 'Confirm' }).click();
  await page.waitForURL('/login');
  await expect(page.getByText('Admin Control Plane')).toBeVisible();
});
