import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession, mockOpsDashboard } from './helpers.js';

test('operations hub shows KPI metrics and doctor', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockOpsDashboard(page);

  await page.goto('/ops');
  await expect(page.getByRole('heading', { name: 'Operations' })).toBeVisible();
  await expect(page.getByText('Outbox pending')).toBeVisible();
  await expect(page.getByText('1200')).toBeVisible();
  await expect(page.getByText('Doctor (ok)')).toBeVisible();
  await expect(page.getByText('Management')).toBeVisible();
});
