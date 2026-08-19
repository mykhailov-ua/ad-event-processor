import { test, expect } from '@playwright/test';
import { mockAuthedSession } from './helpers.js';

const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440000';
const FRAUD_USER = {
  id: 'fraud-1',
  email: 'fraud@test.local',
  role: 'A',
  customer_id: CUSTOMER_ID,
  permissions: ['audit:read', 'campaigns:write'],
};

const IP_HASH = '0123456789abcdef0123456789abcdef';

test.describe('Fraud dashboard labels', () => {
  test('submits manual fraud label with confirm', async ({ page }) => {
    await mockAuthedSession(page, FRAUD_USER);

    let posted = false;

    await page.route('**/api/v1/dashboards/fraud**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          customer_id: CUSTOMER_ID,
          labels_pending: 0,
          recent_labels: [],
          ml_eval_status: 'healthy',
          ml_label_method: 'proxy',
          ml_eval_generated_at: '2026-08-18T10:00:00Z',
          ml_precision: 0.88,
          ml_recall: 0.41,
          fraud_tier_thresholds: {
            scope: 'platform_default',
            pass_max: 30,
            suspect_max: 60,
            ivt_max: 80,
            block_above: 100,
          },
        }),
      });
    });

    await page.route('**/api/v1/fraud/integrations**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([
          {
            campaign_id: '660e8400-e29b-41d4-a716-446655440001',
            name: 'US Push',
            provider: 'facebook',
            configured: true,
            health_status: 'failing',
            dlq_count: 2,
            last_error: 'HTTP 401',
          },
        ]),
      });
    });

    await page.route('**/api/v1/fraud/labels**', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify([]),
        });
        return;
      }
      if (route.request().method() === 'POST') {
        posted = true;
        await route.fulfill({
          status: 202,
          headers: { 'content-type': 'application/json' },
          body: '{}',
        });
        return;
      }
      await route.continue();
    });

    await page.goto(`/dashboards/fraud?customer_id=${CUSTOMER_ID}`);
    await expect(page.getByTestId('fraud-dashboard')).toBeVisible();
    await expect(page.getByTestId('fraud-ml-trust-panel')).toBeVisible();
    await expect(page.getByTestId('fraud-integrations-panel')).toBeVisible();
    await expect(page.getByText('US Push')).toBeVisible();
    await expect(page.getByTestId('fraud-labels-panel')).toBeVisible();

    await page.getByTestId('fraud-label-ip-hash').fill(IP_HASH);
    await page.getByTestId('fraud-label-reason').fill('suspicious cluster');
    await page.getByTestId('fraud-label-submit-fraud').click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
    await expect.poll(() => posted).toBe(true);
    await expect(page.getByText('Label saved')).toBeVisible();
  });

  test('looks up fraud decision breakdown', async ({ page }) => {
    await mockAuthedSession(page, FRAUD_USER);

    await page.route('**/api/v1/dashboards/fraud**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          customer_id: CUSTOMER_ID,
          labels_pending: 0,
          recent_labels: [],
          fraud_tier_thresholds: { pass_max: 30, suspect_max: 60, ivt_max: 80, block_above: 100 },
        }),
      });
    });

    await page.route('**/api/v1/fraud/integrations**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([]),
      });
    });

    await page.route('**/api/v1/fraud/labels**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([]),
      });
    });

    await page.route('**/api/v1/fraud/decisions**', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          ip_hash: IP_HASH,
          campaign_id: '660e8400-e29b-41d4-a716-446655440001',
          window_start: '2026-08-18T09:00:00Z',
          evaluated_at: '2026-08-18T09:01:00Z',
          disclaimer: 'Decision as of last scorer window',
          tier: 'ivt',
          score: 72,
          ml_probability: 0.81,
          adjusted_probability: 0.81,
          residential_proxy: true,
          structural_fraud: false,
          fp_guard_applied: false,
          features: { events: 120, clicks: 4 },
          campaign_thresholds: {
            scope: 'campaign',
            pass_max: 30,
            suspect_max: 60,
            ivt_max: 80,
            block_above: 100,
          },
        }),
      });
    });

    await page.goto(`/dashboards/fraud?customer_id=${CUSTOMER_ID}`);
    await page.getByTestId('fraud-decision-ip-hash').fill(IP_HASH);
    await page.getByTestId('fraud-decision-hours').fill('24');
    await page.getByRole('button', { name: 'Why blocked?' }).click();
    await expect(page.getByTestId('fraud-decision-result')).toBeVisible();
    await expect(page.getByText('IVT')).toBeVisible();
    await expect(page.getByText('Residential proxy: yes')).toBeVisible();
  });
});
