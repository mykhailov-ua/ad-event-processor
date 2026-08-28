import { test, expect } from '@playwright/test';

test('billing exports surface removed from invoice directory', async ({ page }) => {
  const { ADMIN_USER, mockAuthedSession } = await import('./helpers.js');
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/invoices**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto('/billing');
  await expect(page.getByRole('heading', { name: 'Billing' })).toBeVisible();
  await expect(page.getByText('Exports')).toHaveCount(0);
});
