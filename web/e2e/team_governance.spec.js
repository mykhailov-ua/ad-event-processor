import { test, expect } from '@playwright/test';
import { mockAuthedSession, TEAM_LEAD_USER, ADMIN_USER } from './helpers.js';

const CUSTOMER_ID = 'cust-team-1';
const MEMBER_MB = {
  user_id: 'mb-1',
  email: 'buyer@teamco.test',
  role: 'MB',
  campaigns_owned: 2,
  spend_cap_micro: 50_000_000,
  created_at: '2026-01-01T00:00:00Z',
  is_blocked: false,
};
const OVERVIEW = {
  customer_id: CUSTOMER_ID,
  customer_name: 'Team Co',
  balance_micro: 120_000_000,
  currency: 'USD',
  members: [MEMBER_MB],
};

const APPROVAL = {
  id: 'appr-1',
  user_id: 'mb-1',
  campaign_id: 'camp-team-1',
  requested_budget_micro: 80_000_000,
  previous_budget_micro: 50_000_000,
  status: 'PENDING',
};

test('team lead invites member via POST', async ({ page }) => {
  await mockAuthedSession(page, TEAM_LEAD_USER);

  let invitePosted = false;

  await page.route('**/api/v1/team/overview', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(OVERVIEW),
    });
  });

  await page.route('**/api/v1/team/budget-approvals', async (route) => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify([]),
    });
  });

  await page.route('**/api/v1/team/members', async (route) => {
    if (route.request().method() === 'POST') {
      invitePosted = true;
      await route.fulfill({
        status: 201,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          user_id: 'mb-new',
          email: 'newbuyer@teamco.test',
          role: 'MB',
          campaigns_owned: 0,
        }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/team');
  await page.getByTestId('team-invite-email').fill('newbuyer@teamco.test');
  await page.getByTestId('team-invite-submit').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => invitePosted).toBe(true);
});

test('team lead blocks member via PATCH', async ({ page }) => {
  await mockAuthedSession(page, TEAM_LEAD_USER);

  let patchBody = null;

  await page.route('**/api/v1/team/overview', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(OVERVIEW),
    });
  });

  await page.route('**/api/v1/team/budget-approvals', async (route) => {
    await route.fulfill({ status: 200, body: JSON.stringify([]) });
  });

  await page.route('**/api/v1/team/members/mb-1', async (route) => {
    if (route.request().method() === 'PATCH') {
      patchBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ ...MEMBER_MB, is_blocked: true }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/team');
  await page.getByTestId('team-member-block-mb-1').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => patchBody).not.toBeNull();
  expect(patchBody.is_blocked).toBe(true);
});

test('team lead approves budget request', async ({ page }) => {
  await mockAuthedSession(page, TEAM_LEAD_USER);

  let approvePosted = false;

  await page.route('**/api/v1/team/overview', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(OVERVIEW),
    });
  });

  await page.route('**/api/v1/team/budget-approvals', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([APPROVAL]),
    });
  });

  await page.route(`**/api/v1/customers/${CUSTOMER_ID}/billing/forecast`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ customer_id: CUSTOMER_ID }),
    });
  });

  await page.route(`**/api/v1/team/budget-approvals/${APPROVAL.id}/approve`, async (route) => {
    approvePosted = route.request().method() === 'POST';
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ status: 'ok' }),
    });
  });

  await page.goto('/team');
  await expect(page.getByTestId('team-budget-approvals')).toBeVisible();
  await expect
    .poll(async () => {
      return await page.getByTestId('team-approval-row-appr-1').isVisible();
    })
    .toBe(true);
  await page.getByTestId('team-approval-approve-appr-1').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => approvePosted).toBe(true);
});

test('admin assigns campaign owner from config tab', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  const CAMPAIGN_ID = 'camp-owner-1';

  let ownerPut = null;

  await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}`, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          id: CAMPAIGN_ID,
          name: 'Owner test',
          status: 'active',
          budget_limit: '100.00',
          current_spend: '0.00',
          customer_id: CUSTOMER_ID,
          pacing_mode: 'ASAP',
          daily_budget: '50.00',
          timezone: 'UTC',
          freq_limit: 3,
          freq_window: 3600,
          target_countries: ['US'],
          target_url: 'https://example.com/',
          daypart_hours: [],
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        }),
      });
      return;
    }
    await route.continue();
  });

  await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}/**`, async (route) => {
    await route.fulfill({ status: 200, body: JSON.stringify({ items: [], total: 0 }) });
  });

  await page.route('**/api/v1/dashboards/campaign/**', async (route) => {
    await route.fulfill({ status: 200, body: JSON.stringify({ kpis: {} }) });
  });

  await page.route(`**/api/v1/team/overview?customer_id=${CUSTOMER_ID}`, async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(OVERVIEW),
    });
  });

  await page.route(`**/api/v1/campaigns/${CAMPAIGN_ID}/owner`, async (route) => {
    ownerPut = route.request().postDataJSON();
    await route.fulfill({ status: 200, body: JSON.stringify({ status: 'ok' }) });
  });

  await page.goto(`/campaigns/${CAMPAIGN_ID}?tab=config`);
  await page.getByTestId('campaign-owner-select').selectOption('mb-1');
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => ownerPut).not.toBeNull();
  expect(ownerPut.user_id).toBe('mb-1');
});
