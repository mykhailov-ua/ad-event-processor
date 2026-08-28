import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('unknown route shows React 404 page', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.goto('/does-not-exist-route');
  await expect(page.getByText('Page not found')).toBeVisible();
  await expect(page.locator('.error-page').getByRole('link', { name: 'Customers' })).toBeVisible();
});
