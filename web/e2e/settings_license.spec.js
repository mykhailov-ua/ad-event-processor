import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('settings license page shows status and host identity', async ({ page }) => {
  await mockAuthedSession(page, {
    ...ADMIN_USER,
    permissions: [...ADMIN_USER.permissions, 'customers:read', 'settings:write'],
  });

  await page.route('**/api/v1/license/status', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        deployment_id: 'dep-e2e-1',
        state: 'ACTIVE',
        valid_until: '2027-01-01T00:00:00Z',
        host_fingerprint: 'fp-e2e-host',
        hwid_v2: 'hwid-e2e-host',
        hwid_match: true,
        days_to_expiry: 120,
      }),
    });
  });

  await page.goto('/settings/license');
  await expect(page.getByTestId('license-status-panel')).toBeVisible();
  await expect(page.getByText('dep-e2e-1')).toBeVisible();
  await expect(page.getByTestId('license-host-fingerprint')).toContainText('fp-e2e-host');
  await expect(page.getByTestId('license-hwid-v2')).toContainText('hwid-e2e-host');
  await expect(page.getByTestId('license-apply-panel')).toBeVisible();
  await expect(page.getByTestId('license-apply-button')).toBeVisible();
});
