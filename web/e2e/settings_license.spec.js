import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('settings license page shows status panel', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/license/status**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        state: 'active',
        plan_code: 'enterprise',
        host_fingerprint: 'host-abc',
        valid_until: '2027-01-01T00:00:00Z',
        days_to_expiry: 120,
      }),
    });
  });

  await page.goto('/settings/license');
  await expect(page.getByRole('heading', { name: 'License' })).toBeVisible();
  await expect(page.getByText('host-abc')).toBeVisible();
});
