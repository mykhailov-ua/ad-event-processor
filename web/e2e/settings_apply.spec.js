import { test, expect } from '@playwright/test';
import { mockAuthedSession } from './helpers.js';

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
  restart_required: ['control'],
};

test('settings apply once with destructive confirm after restart_required', async ({ page }) => {
  await mockAuthedSession(page, {
    id: 'admin-1',
    email: 'admin@test.local',
    role: 'A',
    customer_id: '',
    permissions: ['settings:read', 'settings:write'],
  });

  let applyCalled = false;

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

  await page.route('**/api/v1/settings/platform/apply', async (route) => {
    applyCalled = true;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ ok: true }),
    });
  });

  await page.goto('/settings');
  await expect(page.getByText('Service restart required: control')).toBeVisible();

  await page.getByRole('button', { name: 'Apply to disk' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => applyCalled).toBe(true);
});

test('settings read-only without settings:write', async ({ page }) => {
  await mockAuthedSession(page, {
    id: 'viewer-1',
    email: 'viewer@test.local',
    role: 'S',
    customer_id: '',
    permissions: ['settings:read'],
  });

  await page.route('**/api/v1/settings/platform', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        ...PLATFORM_VIEW,
        restart_required: [],
      }),
    });
  });

  await page.goto('/settings');
  await expect(page.getByText('Read-only access — you can view settings but cannot save or apply changes.')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Save' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Apply to disk' })).toHaveCount(0);
  await expect(page.locator('input.form-input').first()).toBeDisabled();
});
