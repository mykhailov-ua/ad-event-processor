/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('settings license page shows status fields', async ({ page }) => {
  await mockAuthedSession(page, {
    ...ADMIN_USER,
    permissions: [...ADMIN_USER.permissions, 'customers:read'],
  });

  await page.route('**/api/v1/license/status', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        deployment_id: 'dep-e2e-1',
        state: 'active',
        valid_until: '2027-01-01T00:00:00Z',
      }),
    });
  });

  await page.goto('/settings/license');
  await expect(page.getByTestId('license-status-panel')).toBeVisible();
  await expect(page.getByText('dep-e2e-1')).toBeVisible();
  await expect(page.getByTestId('license-apply-link')).toBeVisible();
});
