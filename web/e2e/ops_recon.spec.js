import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('ops recon lists runs with pagination header', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/recon/runs**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        'content-type': 'application/json',
        'X-Total-Count': '1',
      },
      body: JSON.stringify([{ id: 'run-1', service: 'billing', status: 'ok', created_at: '2026-01-01T00:00:00Z' }]),
    });
  });

  await page.goto('/ops/recon');
  await expect(page.getByTestId('ops-recon-page')).toBeVisible();
  await expect(page.getByRole('gridcell', { name: 'billing' })).toBeVisible();
});
