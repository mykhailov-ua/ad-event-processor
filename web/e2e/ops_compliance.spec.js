import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('ops ml model page renders status', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/ml-model**', async (route) => {
    const url = route.request().url();
    if (url.includes('/eval')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ status: 'ok', generated_at: '2026-01-01T00:00:00Z' }),
      });
      return;
    }
    if (url.includes('/labels')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([]),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        active_version: { id: 'v1' },
        syncing_version: null,
        redis: { version_id: 'v1', shards_consistent: true },
        drift_detected: false,
      }),
    });
  });

  await page.goto('/ops/ml-model');
  await expect(page.getByTestId('ops-ml-model-page')).toBeVisible();
  await expect(page.getByText('Active: v1')).toBeVisible();
});

test('consent page is read-only list', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/consent/proofs**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [
          {
            id: 'proof-1',
            user_id_hash: 'hash-proof-1',
            source: 'web',
            recorded_at: '2026-01-01T00:00:00Z',
          },
        ],
      }),
    });
  });

  await page.goto('/ops/consent');
  await expect(page.getByTestId('ops-consent-page')).toBeVisible();
  await expect(page.getByText('hash-proof-1')).toBeVisible();
});

test('support feedback form submits POST', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/support/feedback/meta**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ deployment_id: 'dep-1', binary_version: '1.0.0' }),
    });
  });

  let posted = false;
  await page.route('**/api/v1/support/feedback', async (route) => {
    if (route.request().method() === 'POST') {
      posted = true;
      await route.fulfill({ status: 204, body: '' });
      return;
    }
    await route.continue();
  });

  await page.goto('/support/feedback');
  await page.locator('input[type="email"]').fill('ops@test.local');
  await page.locator('textarea').fill('E2E feedback');
  await page.getByRole('button', { name: 'Submit feedback' }).click();
  await expect.poll(() => posted).toBe(true);
});

test('settings domains probe calls probe endpoint', async ({ page }) => {
  installDialogAutoAccept(page);
  const adminWithWrite = {
    ...ADMIN_USER,
    permissions: [...ADMIN_USER.permissions, 'settings:write'],
  };
  await mockAuthedSession(page, adminWithWrite);

  await page.route('**/api/v1/domains**', async (route) => {
    const url = route.request().url();
    if (route.request().method() === 'GET' && url.endsWith('/api/v1/domains')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([{ hostname: 'track.example', health_status: 'ok' }]),
      });
      return;
    }
    await route.continue();
  });

  let probed = false;
  await page.route('**/api/v1/domains/track.example/probe**', async (route) => {
    if (route.request().method() === 'POST') {
      probed = true;
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ hostname: 'track.example', health_status: 'ok' }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/settings/domains');
  await page.getByRole('button', { name: 'Probe' }).click();
  await expect.poll(() => probed).toBe(true);
});
