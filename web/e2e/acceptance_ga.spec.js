import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const BUYER_INTEGRATION_USER = {
  id: 'buyer-int-1',
  email: 'buyer-int@test.local',
  role: 'B',
  customer_id: 'cust-1',
  permissions: ['campaigns:read', 'audit:read'],
};

const CAMPAIGN = {
  id: 'camp-ga-1',
  name: 'GA Campaign',
  status: 'ACTIVE',
  customer_id: 'cust-1',
  budget_limit: '$100.00',
  current_spend: '$10.00',
  daily_budget: '$5.00',
  pacing_mode: 'even',
};

async function mockCampaignDetailApis(page) {
  await page.route('**/api/v1/campaigns/camp-ga-1**', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(CAMPAIGN),
    });
  });

  await page.route('**/api/v1/dashboards/campaign/camp-ga-1**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ impressions_7d: 10, clicks_7d: 2 }),
    });
  });

  await page.route('**/api/v1/ops/doctor', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        click_url_template: 'https://trk.example/click?campaign_id={campaign_id}',
        tracking_domain: 'trk.example',
        rtb_mode: 'shadow',
        rtb_enabled: true,
        checks: [],
      }),
    });
  });

  await page.route('**/api/v1/settings/platform', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        config: {
          tracking_domain: 'trk.example',
          edge_expose_click: true,
          edge_expose_openrtb: false,
        },
      }),
    });
  });
}

test('buyer integration tab surfaces click and track URLs', async ({ page }) => {
  await mockAuthedSession(page, BUYER_INTEGRATION_USER);
  await mockCampaignDetailApis(page);

  await page.goto('/campaigns/camp-ga-1');
  await page.getByRole('tab', { name: 'Integration' }).click();
  await expect(page.getByText('Click URL (campaign traffic)')).toBeVisible();
  await expect(page.getByTestId('integration-inbound-url')).toContainText('Postback URL');
  await expect(
    page.locator('.integration-copy-row code').filter({ hasText: '/click' })
  ).toBeVisible();
  await expect(
    page.locator('.integration-copy-row code').filter({ hasText: '/track' }).first()
  ).toBeVisible();
});

test('campaign list bulk pause is available when rows selected', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [
          {
            id: 'c-bulk-1',
            name: 'Bulk One',
            status: 'ACTIVE',
            customer_id: '11111111-1111-1111-1111-111111111111',
          },
          {
            id: 'c-bulk-2',
            name: 'Bulk Two',
            status: 'PAUSED',
            customer_id: '11111111-1111-1111-1111-111111111111',
          },
        ],
        total: 2,
      }),
    });
  });

  await page.goto('/campaigns?customer_id=11111111-1111-1111-1111-111111111111');
  await expect(page.getByRole('cell', { name: 'Bulk One', exact: true })).toBeVisible();
  await page.getByLabel('Select Bulk One').check();
  await expect(page.locator('#campaigns-bulk-actions')).toBeVisible();
  await expect(page.locator('#campaigns-bulk-actions [data-action="pause"]')).toBeEnabled();
  await expect(page.locator('#campaigns-bulk-actions [data-action="resume"]')).toBeEnabled();
});

test('buyer overview shows fraud KPI tiles without opening fraud dashboard', async ({ page }) => {
  await mockAuthedSession(page, BUYER_INTEGRATION_USER);

  await page.route('**/api/v1/dashboards/buyer*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        active: 1,
        paused: 0,
        archived: 0,
        impressions_7d: 100,
        clicks_7d: 10,
        overspend_count: 0,
        campaigns: [],
        attention: [],
      }),
    });
  });

  await page.route('**/api/v1/dashboards/fraud*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        customer_id: 'cust-1',
        silent_reject_campaigns: 2,
        edge_blocked_fraud: 15,
        geo_hints: [{ country: 'US', ivt_rate: 0.12, clicks: 100 }],
      }),
    });
  });

  await page.goto('/');
  await expect(page.getByTestId('fraud-kpi-tiles')).toBeVisible();
  await expect(page.getByTestId('fraud-kpi-silent-reject-campaigns')).toBeVisible();
  await expect(
    page.getByTestId('fraud-kpi-tiles').getByRole('link', { name: /High-IVT geo hints/i })
  ).toBeVisible();
});

test('live report campaign-overview does not mount stub banner', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/reports/campaign-overview*', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        rows: [{ name: 'Camp A', status: 'ACTIVE', clicks_7d: 5 }],
        freshness: { stale: false, ch_lag_seconds: 0 },
      }),
    });
  });

  await page.goto(
    '/reports/campaign-overview?customer_id=11111111-1111-1111-1111-111111111111&from=2026-01-01&to=2026-01-31'
  );
  await expect(page.getByRole('heading', { name: 'Campaign overview' })).toBeVisible();
  await expect(page.getByText('Planned API')).toHaveCount(0);
});
