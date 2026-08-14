/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';

test('bootstrap page submits with strong confirm and install token', async ({ page }) => {
  let bootstrapCalled = false;
  /** @type {Record<string, string>|null} */
  let bootstrapHeaders = null;

  await page.route('**/api/v1/settings/platform/bootstrap', async (route) => {
    bootstrapCalled = true;
    bootstrapHeaders = route.request().headers();
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ bootstrap_complete: true }),
    });
  });

  await page.goto('/bootstrap');
  await page.getByLabel('Install token').fill('install-token');
  await page.getByLabel('Admin email').fill('admin@bootstrap.local');
  await page.getByLabel('Admin password').fill('secret-bootstrap');
  await page.getByLabel('Tracking domain').fill('track.local');

  await page.getByRole('button', { name: 'Initialize platform' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();

  await page.getByLabel('Type DELETE to confirm').fill('DELETE');
  await page.getByRole('checkbox', { name: 'I understand the consequences' }).check();
  await page.getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => bootstrapCalled).toBe(true);
  expect(bootstrapHeaders?.['x-install-token']).toBe('install-token');
  await page.waitForURL('/install/done');
});

test('login redirects to bootstrap when platform not initialized', async ({ page }) => {
  await page.route('**/api/v1/meta', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ bootstrap_complete: false, version: 'e2e' }),
    });
  });

  await page.goto('/login');
  await page.waitForURL('/bootstrap');
  await expect(page.getByRole('heading', { name: 'Bootstrap' })).toBeVisible();
});

test('shell shows bootstrap banner when bootstrap incomplete', async ({ page }) => {
  const { mockAuthedSession, ADMIN_USER } = await import('./helpers.js');

  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/meta', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        version: 'e2e',
        bootstrap_complete: false,
      }),
    });
  });

  await page.route('**/api/v1/ops/doctor', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ overall: 'ok', checks: [] }),
    });
  });

  await page.route('**/api/v1/ops/incidents', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ partial: false, outbox: { pending: 0 } }),
    });
  });

  await page.route('**/api/v1/ops/dashboard/summary', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ outbox_pending: 0, services: [] }),
    });
  });

  await page.goto('/');
  await expect(page.getByText('Platform bootstrap is not complete')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Run bootstrap' })).toHaveAttribute('href', '/bootstrap');
});
