/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const UNSYNCED_SHARDS = {
  emergency_breaker: 'closed',
  shards: [
    {
      shard_id: 0,
      ping_ok: true,
      ping_latency_ms: 1.2,
      config_version_lag: 3,
      config_version_synced: false,
    },
    {
      shard_id: 1,
      ping_ok: true,
      ping_latency_ms: 0.8,
      config_version_lag: 0,
      config_version_synced: true,
    },
  ],
};

const SYNCED_SHARDS = {
  ...UNSYNCED_SHARDS,
  shards: [
    {
      shard_id: 0,
      ping_ok: true,
      ping_latency_ms: 1.2,
      config_version_lag: 0,
      config_version_synced: true,
    },
    UNSYNCED_SHARDS.shards[1],
  ],
};

test('shard 0 catch-up button clears unsynced banner after success', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let synced = false;
  let catchupCalled = false;

  await page.route('**/api/v1/ops/shards', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(synced ? SYNCED_SHARDS : UNSYNCED_SHARDS),
    });
  });

  await page.route('**/api/v1/ops/shards/0/catchup', async (route) => {
    catchupCalled = true;
    synced = true;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ status: 'ok' }),
    });
  });

  await page.route('**/api/v1/ops/dashboard/metrics**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        points: [{ ts: '2026-08-12T10:00:00Z', value: 1_700_000_000 }],
      }),
    });
  });

  await page.goto('/ops/shards');
  await expect(page.getByTestId('shard0-catchup-banner')).toBeVisible();
  await expect(page.getByTestId('shard0-catchup-btn')).toBeVisible();

  await page.getByTestId('shard0-catchup-btn').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByLabel('Type DELETE to confirm').fill('DELETE');
  await page.getByRole('checkbox', { name: 'I understand the consequences' }).check();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => catchupCalled).toBe(true);
  await expect(page.getByText('Catch-up started')).toBeVisible();
  await expect(page.getByTestId('shard0-catchup-banner')).toHaveCount(0);
  await expect(page.getByTestId('shard0-catchup-btn')).toHaveCount(0);
  await expect(page.getByTestId('shard0-catchup-metric')).toBeVisible();
});

test('shard 0 catch-up surfaces 503 when worker not configured', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/shards', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(UNSYNCED_SHARDS),
    });
  });

  await page.route('**/api/v1/ops/shards/0/catchup', async (route) => {
    await route.fulfill({
      status: 503,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        error: { code: 'UNAVAILABLE', message: 'shard 0 catch-up not configured' },
      }),
    });
  });

  await page.route('**/api/v1/ops/dashboard/metrics**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ points: [] }),
    });
  });

  await page.goto('/ops/shards');
  await page.getByTestId('shard0-catchup-btn').click();
  await page.getByLabel('Type DELETE to confirm').fill('DELETE');
  await page.getByRole('checkbox', { name: 'I understand the consequences' }).check();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();

  await expect(page.getByText('shard 0 catch-up not configured')).toBeVisible();
  await expect(page.getByTestId('shard0-catchup-banner')).toBeVisible();
});
