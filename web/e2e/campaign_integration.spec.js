/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CAMPAIGN_ID = '550e8400-e29b-41d4-a716-446655440000';

const CAMPAIGN = {
  id: CAMPAIGN_ID,
  name: 'Integration Camp',
  status: 'ACTIVE',
  customer_id: 'cust-1',
  budget_limit: 100,
  current_spend: 10,
  pacing_mode: 'even',
};

/**
 * @param {import('@playwright/test').Page} page
 */
async function mockCampaignApis(page) {
  await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(CAMPAIGN),
    });
  });

  await page.route('**/api/v1/ops/doctor', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        tracking_domain: 'trk.example.com',
        click_url_template: 'https://trk.example.com/click?campaign_id={campaign_id}&sub1={sub1}',
        rtb_mode: 'off',
        rtb_enabled: false,
      }),
    });
  });

  await page.route('**/api/v1/settings/platform', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        config: {
          tracking_domain: 'trk.example.com',
          edge_expose_click: true,
          edge_expose_openrtb: false,
        },
        click_url_template: 'https://trk.example.com/click?campaign_id={campaign_id}&sub1={sub1}',
      }),
    });
  });

  await page.route('**/api/v1/postbacks/config', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([]),
    });
  });

  await page.route('**/api/v1/postbacks/dlq', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([]),
    });
  });

  await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}/dashboard**`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ kpis: {} }),
    });
  });

  await page.route('**/api/v1/buyer/portfolio**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [] }),
    });
  });
}

test('Integration tab sub15 appears in click URL', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignApis(page);

  await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=tracking`);
  await page.getByTestId('integration-sub11-30').locator('summary').click();
  await page.locator('#track-sub15').fill('test');
  await expect(page.getByTestId('integration-click-url')).toContainText('sub15=test');
});

test('Integration tab shows click + inbound S2S copy rows', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignApis(page);

  await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=tracking`);
  await expect(page.getByRole('tab', { name: 'Integration' })).toBeVisible();
  await expect(page.getByTestId('campaign-integration-kit')).toBeVisible();
  await expect(page.getByTestId('integration-click-url')).toContainText(CAMPAIGN_ID);
  await expect(page.getByTestId('integration-inbound-url')).toContainText('https://trk.example.com/track');
  await expect(page.getByTestId('integration-inbound-body')).toContainText('"type": "conversion"');
  await expect(page.getByTestId('integration-macro-table')).toContainText('{sub1}…{sub30}');
  await expect(page.getByTestId('traffic-guide')).toBeVisible();

  const copyBtn = page.getByTestId('integration-click-url-copy');
  await expect(copyBtn).toBeEnabled();
  await copyBtn.click();
});

test('Integration tab shows zero-redirect snippet', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignApis(page);

  await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=tracking`);
  await expect(page.getByTestId('integration-direct-snippet')).toBeVisible();
  await expect(page.getByTestId('integration-direct-snippet')).toContainText('bidshardTrack');
  await expect(page.getByTestId('integration-direct-snippet')).toContainText(CAMPAIGN_ID);
});

test('CAPI tab loads and links back to Integration', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignApis(page);

  await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=postbacks`);
  await expect(page.getByRole('tab', { name: 'CAPI & Postbacks' })).toBeVisible();
  await expect(page.getByTestId('campaign-capi-panel')).toBeVisible();
  await expect(page.getByTestId('campaign-capi-panel').getByRole('link', { name: 'Integration' })).toBeVisible();
});

test('CAPI tab shows test event code for Meta', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignApis(page);

  await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=postbacks`);
  await page.selectOption('#pb-provider', 'facebook');
  await expect(page.getByTestId('postback-test-event-code-field')).toBeVisible();
});
