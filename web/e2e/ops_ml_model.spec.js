import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const ML_MODEL_FIXTURE = {
  active_version: {
    id: 'v2',
    artifact_hash: 'abcd1234efgh5678abcd1234efgh5678',
    status: 'ACTIVE',
    created_at: '2026-08-18T08:00:00Z',
  },
  syncing_version: null,
  redis: {
    version_id: 'v2',
    hash: 'abcd1234efgh5678abcd1234efgh5678',
    applied_at: '2026-08-18T08:05:00Z',
    shards_reporting: 2,
    shards_consistent: true,
  },
  shard_sync: [
    {
      shard_id: 0,
      model_version: 'v2',
      phase: 'COMPLETE',
      started_at: '2026-08-18T08:01:00Z',
    },
  ],
  drift_detected: false,
  precision: 0.91,
  recall: 0.42,
  importance: [
    { name: 'events', value: 0.31 },
    { name: 'ctr', value: 0.22 },
    { name: 'clicks', value: 0.18 },
    { name: 'unique_users', value: 0.12 },
    { name: 'spend_ratio', value: 0.09 },
  ],
};

test.describe('Ops ML model page', () => {
  test('renders status from fixture API', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    await page.route('**/api/v1/ops/ml-model**', async (route) => {
      const url = route.request().url();
      if (url.includes('/labels') || url.includes('/eval')) {
        await route.continue();
        return;
      }
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(ML_MODEL_FIXTURE),
      });
    });

    await page.goto('/ops/ml-model');
    await expect(page.getByTestId('ops-ml-model-page')).toBeVisible();
    await expect(page.getByTestId('ops-ml-eval-panel')).toBeVisible();
    await expect(page.getByText('0.9100')).toBeVisible();
    await expect(page.getByTestId('ops-ml-importance-chart')).toBeVisible();
    await expect(page.getByText('events')).toBeVisible();
  });

  test('lazy loads labels tab', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    await page.route('**/api/v1/ops/ml-model', async (route) => {
      if (route.request().url().includes('/labels')) {
        await route.fulfill({
          status: 200,
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify([
            {
              ip_hash: '0123456789abcdef0123456789abcdef',
              label: 1,
              reason: 'bot farm',
              source: 'ops',
              created_at: '2026-08-18T09:00:00Z',
            },
          ]),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(ML_MODEL_FIXTURE),
      });
    });

    await page.goto('/ops/ml-model');
    await page.getByTestId('ops-ml-labels-tab').click();
    await expect(page.getByTestId('ops-ml-labels-panel')).toBeVisible();
    await expect(page.getByText('bot farm')).toBeVisible();
  });

  test('lazy loads eval quality tab', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    await page.route('**/api/v1/ops/ml-model**', async (route) => {
      const url = route.request().url();
      if (url.includes('/eval')) {
        await route.fulfill({
          status: 200,
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify({
            status: 'ok',
            generated_at: '2026-08-18T10:00:00Z',
            hours: 168,
            threshold: 0.6,
            proxy_metrics: {
              status: 'ok',
              label_method: 'proxy',
              labeled_rows: 100,
              precision: 0.91,
              recall: 0.42,
            },
            audited_metrics: {
              status: 'empty',
              label_method: 'manual',
              labeled_rows: 0,
              confidence: 'low',
            },
          }),
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
        body: JSON.stringify(ML_MODEL_FIXTURE),
      });
    });

    await page.goto('/ops/ml-model');
    await page.getByTestId('ops-ml-eval-tab').click();
    await expect(page.getByTestId('ops-ml-eval-quality-panel')).toBeVisible();
    await expect(page.getByText('Proxy metrics')).toBeVisible();
    await expect(page.getByText('Audited metrics')).toBeVisible();
    await expect(page.getByText('not reported as accuracy')).toBeVisible();
  });
});
