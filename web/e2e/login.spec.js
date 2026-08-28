import { test, expect } from '@playwright/test';
import { installDialogAutoAccept, mockLoginSuccess } from './helpers.js';

test('valid login navigates to customers directory', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockLoginSuccess(page);

  await page.goto('/login');
  await page.fill('#login-email', 'admin@test.local');
  await page.fill('#login-password', 'secret');
  await Promise.all([
    page.waitForURL(/\/customers/, { timeout: 15_000 }),
    page.getByRole('button', { name: 'Sign in' }).click(),
  ]);
  await expect(page.getByRole('heading', { name: 'Customers' })).toBeVisible();
});

test('logout returns to login page', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockLoginSuccess(page);

  await page.goto('/login');
  await page.fill('#login-email', 'admin@test.local');
  await page.fill('#login-password', 'secret');
  await Promise.all([
    page.waitForURL(/\/customers/, { timeout: 15_000 }),
    page.getByRole('button', { name: 'Sign in' }).click(),
  ]);

  await page.route('**/api/v1/auth/logout', async (route) => {
    await route.fulfill({ status: 204, body: '' });
  });

  await page.getByRole('button', { name: 'Logout' }).click();
  await page.waitForURL('/login');
  await expect(page.getByText('Admin Control Plane')).toBeVisible();
});
