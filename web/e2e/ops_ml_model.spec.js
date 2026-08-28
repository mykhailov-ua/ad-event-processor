import { test, expect } from '@playwright/test';
import { ADMIN_USER, mockAuthedSession } from './helpers.js';

test('ops ml model page renders status from fixture API', async ({ page }) => {
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
        active_version: { id: 'model-v1' },
        drift_detected: false,
        redis: { version_id: 'model-v1', shards_consistent: true },
      }),
    });
  });

  await page.goto('/ops/ml-model');
  await expect(page.getByTestId('ops-ml-model-page')).toBeVisible();
  await expect(page.getByText('Active: model-v1')).toBeVisible();
});

test('ops ml model labels section visible', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/ml-model**', async (route) => {
    const url = route.request().url();
    if (url.includes('/eval')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ status: 'ok' }),
      });
      return;
    }
    if (url.includes('/labels')) {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([{ ip_hash: 'abc', label: 1 }]),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ active_version: { id: 'v1' } }),
    });
  });

  await page.goto('/ops/ml-model');
  await expect(page.getByText('Manual labels')).toBeVisible();
});
