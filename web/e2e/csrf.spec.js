import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const PLATFORM_VIEW = {
  config: {
    tracking_domain: 'track.example',
    default_currency: 'USD',
    timezone: 'UTC',
    ingress_schema: 'espx_native',
    telemetry_enabled: true,
    profile: 'single_vps',
    edge_xdp: false,
    network_interface: 'eth0',
    stripe: { enabled: false },
  },
  bootstrap_complete: true,
  restart_required: [],
};

test('settings PATCH sends X-CSRF-Token header', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let patchHeaders = null;

  await page.route('**/api/v1/settings/platform', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(PLATFORM_VIEW),
      });
      return;
    }
    if (method === 'PATCH') {
      patchHeaders = route.request().headers();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(PLATFORM_VIEW),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/settings');
  await page.getByRole('button', { name: 'Save' }).click();
  await page.getByRole('button', { name: 'Confirm' }).click();

  expect(patchHeaders?.['x-csrf-token']).toBe('e2e-csrf-token');
});
