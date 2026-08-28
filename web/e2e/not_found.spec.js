import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('unknown route shows React 404 page', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.goto('/definitely-not-a-real-admin-route');
  await expect(page.getByText('404')).toBeVisible();
  await expect(page.getByText('Page not found')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Home' })).toBeVisible();
});
