import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CAMPAIGN = {
  id: 'camp-edit-1',
  name: 'Edit me',
  status: 'active',
  budget_limit: '100.00',
  current_spend: '0.00',
  customer_id: 'cust-1',
  pacing_mode: 'ASAP',
  daily_budget: '50.00',
  timezone: 'UTC',
  freq_limit: 3,
  freq_window: 3600,
  target_countries: ['US'],
  target_url: 'https://old.example/',
  daypart_hours: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

async function mockCampaignDetailApis(page) {
  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
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

  await page.route('**/api/v1/campaigns/camp-edit-1/**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0, kpis: {} }),
    });
  });

  await page.route('**/api/v1/dashboards/campaign/camp-edit-1**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ kpis: {} }),
    });
  });
}

async function confirmAndSave(page) {
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
}

test('campaign config PATCH sends daily_budget_micro and geo', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  let patchBody = null;

  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...CAMPAIGN, ...patchBody }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await expect(page.locator('#cfg-name')).toHaveValue('Edit me');
  await page.getByTestId('cfg-daily-budget').fill('75.50');
  await page.getByTestId('cfg-geo').fill('US,CA');
  await page.getByTestId('cfg-target-url').fill('https://new.example/landing');
  await confirmAndSave(page);

  await expect.poll(() => patchBody).not.toBeNull();
  expect(patchBody.daily_budget_micro).toBe(75_500_000);
  expect(patchBody.target_countries).toEqual(['US', 'CA']);
  expect(patchBody.target_url).toBe('https://new.example/landing');
});

test('campaign config PATCH sends freq_limit and freq_window', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  let patchBody = null;

  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...CAMPAIGN, ...patchBody }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await expect(page.locator('#cfg-name')).toHaveValue('Edit me');
  await page.getByTestId('cfg-freq-limit').fill('5');
  await page.getByTestId('cfg-freq-window').fill('7200');
  await confirmAndSave(page);

  await expect.poll(() => patchBody).not.toBeNull();
  expect(patchBody.freq_limit).toBe(5);
  expect(patchBody.freq_window).toBe(7200);
});

test('campaign config PATCH sends budget_limit_micro and status', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  let patchBody = null;

  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...CAMPAIGN, ...patchBody }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await page.getByTestId('campaign-budget-total').fill('200.00');
  await page.getByTestId('cfg-status').selectOption('PAUSED');
  await page.getByTestId('cfg-daypart').fill('9,10,11');
  await confirmAndSave(page);

  await expect.poll(() => patchBody).not.toBeNull();
  expect(patchBody.budget_limit_micro).toBe(200_000_000);
  expect(patchBody.status).toBe('paused');
  expect(patchBody.daypart_hours).toEqual([9, 10, 11]);
});

test('campaign config PATCH sends click_delivery proxy fields', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  let patchBody = null;

  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...CAMPAIGN, ...patchBody }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await page.getByTestId('cfg-click-delivery').selectOption('proxy');
  await page.getByTestId('cfg-proxy-upstream-url').fill('https://upstream.example/offer');
  await page.getByTestId('cfg-proxy-rewrite-assets').check();
  await confirmAndSave(page);

  await expect.poll(() => patchBody).not.toBeNull();
  expect(patchBody.click_delivery).toBe('proxy');
  expect(patchBody.proxy_upstream_url).toBe('https://upstream.example/offer');
  expect(patchBody.proxy_rewrite_assets).toBe(true);
});

test('campaign creative tab PATCH links brand_id on first create', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  let patchBody = null;

  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...CAMPAIGN, brand_id: patchBody.brand_id }),
      });
      return;
    }
    await route.continue();
  });

  await page.route('**/api/v1/brands', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ id: 'brand-link-1' }),
      });
      return;
    }
    await route.continue();
  });

  await page.route('**/api/v1/brands/brand-link-1/creatives', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ id: 'creative-1' }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=creative');
  await page.getByLabel('Landing URL').fill('https://landing.example/');
  await page.getByRole('button', { name: 'Create brand & add' }).click();

  const confirmDialog = () => page.getByRole('dialog');
  const confirmBtn = () => confirmDialog().getByRole('button', { name: 'Confirm' });

  await expect(confirmDialog()).toBeVisible();
  await confirmBtn().click();
  await expect(confirmDialog()).toBeVisible();
  await confirmBtn().click();
  await expect(confirmDialog()).toBeVisible();
  await confirmBtn().click();

  await expect.poll(() => patchBody).not.toBeNull();
  expect(patchBody.brand_id).toBe('brand-link-1');
});

test('campaign config PATCH sends GMA fields', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  let patchBody = null;

  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...CAMPAIGN, ...patchBody }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await page.getByTestId('cfg-l1-cidr-block').uncheck();
  await page.getByTestId('cfg-l15-proxy-vpn-block').uncheck();
  await page.getByTestId('cfg-tls-fp-block').uncheck();
  await page.getByTestId('cfg-conn-type-policy').selectOption('mobile_only');
  await page.getByTestId('cfg-link-signing').check();
  await page.getByTestId('cfg-link-signing-ttl').fill('1200');
  await confirmAndSave(page);

  await expect.poll(() => patchBody).not.toBeNull();
  expect(patchBody.l1_cidr_block_enabled).toBe(false);
  expect(patchBody.l15_proxy_vpn_block_enabled).toBe(false);
  expect(patchBody.tls_fingerprint_block_enabled).toBe(false);
  expect(patchBody.conn_type_policy).toBe('mobile_only');
  expect(patchBody.link_signing_enabled).toBe(true);
  expect(patchBody.link_signing_ttl_sec).toBe(1200);
});

test('campaign config rejects proxy mode without upstream URL', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  let patchCalled = false;

  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify(CAMPAIGN) });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await page.getByTestId('cfg-click-delivery').selectOption('proxy');
  await page.getByRole('button', { name: 'Save changes' }).click();

  await expect(
    page.getByText('Proxy upstream URL is required when click delivery is reverse proxy')
  ).toBeVisible();
  expect(patchCalled).toBe(false);
});

test('campaign config rejects empty name before PATCH', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  let patchCalled = false;

  await page.route('**/api/v1/campaigns/camp-edit-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      patchCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify(CAMPAIGN) });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-edit-1?tab=config');
  await expect(page.locator('#cfg-name')).toBeVisible();
  await page.locator('#cfg-name').fill('');
  await page.getByRole('button', { name: 'Save changes' }).click();

  await expect(page.getByText('Name is required')).toBeVisible();
  expect(patchCalled).toBe(false);
});

test('campaign tracking apply-templates POST sends traffic_source', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockCampaignDetailApis(page);

  await page.route('**/api/v1/ops/doctor', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ tracking_domain: 'trk.example', rtb_mode: 'off', rtb_enabled: false }),
    });
  });

  await page.route('**/api/v1/settings/platform', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        config: { tracking_domain: 'trk.example' },
        click_url_template: 'https://trk.example/click',
      }),
    });
  });

  let applyBody = null;

  await page.route('**/api/v1/campaigns/camp-edit-1/apply-templates', async (route) => {
    applyBody = route.request().postDataJSON();
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        campaign_id: 'camp-edit-1',
        traffic_source: { target_url: 'https://partner.example/' },
      }),
    });
  });

  await page.goto('/campaigns/camp-edit-1?tab=tracking');
  await page.getByTestId('apply-traffic-source').selectOption('traffic_propellerads');
  await page
    .getByTestId('apply-campaign-templates')
    .getByRole('button', { name: 'Apply templates to campaign' })
    .click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();

  await expect.poll(() => applyBody).not.toBeNull();
  expect(applyBody.traffic_source).toBe('traffic_propellerads');
  expect(applyBody.tracking_domain).toBe('trk.example');
});
