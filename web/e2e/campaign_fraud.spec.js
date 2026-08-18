import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CAMPAIGN_ID = 'camp-fraud-1';

const CAMPAIGN = {
  id: CAMPAIGN_ID,
  name: 'Fraud controls',
  status: 'active',
  budget_limit: '100.00',
  current_spend: '0.00',
  customer_id: 'cust-1',
  pacing_mode: 'ASAP',
  timezone: 'UTC',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const FRAUD_CONFIG = {
  campaign_id: CAMPAIGN_ID,
  fraud_threshold_pass: 30,
  fraud_threshold_suspect: 60,
  fraud_threshold_ivt: 80,
  fraud_threshold_block: 100,
  ghost_ivt_enabled: false,
};

test.describe('Campaign fraud tab', () => {
  test('loads fraud settings and applies aggressive preset', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}`, async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify(CAMPAIGN),
        });
        return;
      }
      await route.continue();
    });

    await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}/**`, async (route) => {
      const url = route.request().url();
      if (url.includes('/fraud')) {
        if (route.request().method() === 'GET') {
          await route.fulfill({
            status: 200,
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify(FRAUD_CONFIG),
          });
          return;
        }
        if (route.request().method() === 'PATCH') {
          await route.fulfill({
            status: 200,
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({
              ...FRAUD_CONFIG,
              fraud_threshold_pass: 20,
              fraud_threshold_suspect: 45,
              fraud_threshold_ivt: 65,
              fraud_threshold_block: 85,
            }),
          });
          return;
        }
        if (route.request().method() === 'POST' && url.includes('/fraud/preview')) {
          await route.fulfill({
            status: 200,
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({
              campaign_id: CAMPAIGN_ID,
              affected_ips_7d: 7,
              sample_size: 50,
              by_tier: { suspect: 3, ivt: 2, block: 2 },
              disclaimer: 'estimate based on last 7d shadow scores',
            }),
          });
          return;
        }
      }
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0, kpis: {} }),
      });
    });

    await page.route(`**/api/v1/dashboards/campaign/${CAMPAIGN_ID}**`, async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ kpis: {} }),
      });
    });

    await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=fraud`);
    await expect(page.getByTestId('campaign-fraud-section')).toBeVisible();
    await expect(page.getByTestId('fraud-threshold-fraud_threshold_pass')).toHaveValue('30');

    await page.getByTestId('fraud-preset-aggressive').click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
    await expect(page.getByText('Fraud settings saved')).toBeVisible();
    await expect(page.getByTestId('fraud-threshold-fraud_threshold_pass')).toHaveValue('20');
  });

  test('shows impact preview before save', async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}`, async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify(CAMPAIGN),
        });
        return;
      }
      await route.continue();
    });

    await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}/**`, async (route) => {
      const url = route.request().url();
      if (url.includes('/fraud')) {
        if (route.request().method() === 'GET') {
          await route.fulfill({
            status: 200,
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify(FRAUD_CONFIG),
          });
          return;
        }
        if (route.request().method() === 'POST' && url.includes('/preview')) {
          await route.fulfill({
            status: 200,
            headers: { 'content-type': 'application/json' },
            body: JSON.stringify({
              campaign_id: CAMPAIGN_ID,
              affected_ips_7d: 7,
              sample_size: 50,
              by_tier: { suspect: 3, ivt: 2, block: 2 },
              disclaimer: 'estimate based on last 7d shadow scores',
            }),
          });
          return;
        }
      }
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0, kpis: {} }),
      });
    });

    await page.route(`**/api/v1/dashboards/campaign/${CAMPAIGN_ID}**`, async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ kpis: {} }),
      });
    });

    await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=fraud`);
    await page.getByTestId('fraud-threshold-fraud_threshold_pass').fill('25');
    await page.getByTestId('fraud-preview-impact').click();
    await expect(page.getByTestId('fraud-preview-panel')).toContainText('7');
    await expect(page.getByTestId('fraud-preview-panel')).toContainText('estimate based on last 7d');
  });
});
