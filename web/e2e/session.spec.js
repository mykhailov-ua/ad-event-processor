/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockLoginSuccess, ADMIN_USER } from './helpers.js';

const PLATFORM_VIEW = {
  config: {
    tracking_domain: 'track.example',
    default_currency: 'USD',
    timezone: 'UTC',
    ingress_schema: 'ad_event_processor_native',
    telemetry_enabled: true,
    profile: 'single_vps',
    edge_xdp: false,
    network_interface: 'eth0',
    stripe: { enabled: false },
  },
  bootstrap_complete: true,
  restart_required: [],
};

test('login shows session expired reason', async ({ page }) => {
  await page.goto('/login?reason=session');
  await expect(page.getByText('Your session expired. Sign in again.')).toBeVisible();
});

test('CSRF from cookie after full page reload', async ({ page }) => {
  await mockLoginSuccess(page);

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
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(PLATFORM_VIEW),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/login');
  await page.fill('input[type=email]', 'admin@test.local');
  await page.fill('input[type=password]', 'secret');
  await page.click('button[type=submit]');
  await page.waitForURL('/');

  await page.context().addCookies([{
    name: 'csrfToken',
    value: 'cookie-csrf-token',
    url: 'http://127.0.0.1:4173',
  }]);

  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: ADMIN_USER.id,
        email: ADMIN_USER.email,
        role: ADMIN_USER.role,
        customer_id: ADMIN_USER.customer_id,
        permissions: ADMIN_USER.permissions,
      }),
    });
  });

  await page.route('**/api/v1/meta', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ version: 'e2e', bootstrap_complete: true }),
    });
  });

  await page.reload();

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
  await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();
  await page.getByRole('button', { name: 'Save' }).click();
  await page.getByRole('button', { name: 'Confirm' }).click();

  expect(patchHeaders?.['x-csrf-token']).toBe('cookie-csrf-token');
});
