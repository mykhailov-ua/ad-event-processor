import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('settings save sends one PATCH per confirmed submit', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let patchCount = 0;

  await page.route('**/api/v1/settings/platform', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          config: {
            tracking_domain: 't.example',
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
        }),
      });
      return;
    }
    if (method === 'PATCH') {
      patchCount += 1;
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ok: true }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/settings');
  const save = page.getByRole('button', { name: 'Save' });
  await save.click();
  await page.getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => patchCount).toBe(1);

  await save.click();
  await page.getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => patchCount).toBe(2);
});

test('pause campaign sends Idempotency-Key header', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let idemKey = null;
  let status = 'ACTIVE';

  await page.route('**/api/v1/campaigns/camp-idem-1', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        id: 'camp-idem-1',
        name: 'Idem Camp',
        status,
        customer_id: 'cust-1',
      }),
    });
  });

  await page.route('**/api/v1/selfserve/campaigns/camp-idem-1/pause', async (route) => {
    idemKey = route.request().headers()['idempotency-key'] ?? null;
    status = 'PAUSED';
    await route.fulfill({
      status: 202,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ status: 'accepted' }),
    });
  });

  await page.goto('/campaigns/camp-idem-1');
  await page.getByRole('button', { name: 'Pause' }).click();
  await page.getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => idemKey).toBeTruthy();
});
