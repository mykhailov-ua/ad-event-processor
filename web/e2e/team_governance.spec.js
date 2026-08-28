import { test, expect } from '@playwright/test';
import {
  ADMIN_USER,
  TEAM_LEAD_USER,
  installDialogAutoAccept,
  mockAuthedSession,
} from './helpers.js';

test('team lead invites member via POST', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, TEAM_LEAD_USER);

  await page.route('**/api/v1/team/overview**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        customer_id: 'cust-team-1',
        customer_name: 'Team Co',
        members: [],
      }),
    });
  });

  let inviteBody = null;
  await page.route('**/api/v1/team/members**', async (route) => {
    if (route.request().method() === 'POST') {
      inviteBody = route.request().postDataJSON();
      await route.fulfill({ status: 204, body: '' });
      return;
    }
    await route.continue();
  });

  await page.goto('/team?customer_id=cust-team-1&tab=members');
  await page.getByLabel('Email').fill('new@test.local');
  await page.getByRole('button', { name: 'Invite' }).click();
  expect(inviteBody?.email).toBe('new@test.local');
});

test('team lead approves budget request', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, TEAM_LEAD_USER);

  await page.route('**/api/v1/team/overview**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ customer_id: 'cust-team-1', members: [] }),
    });
  });

  await page.route('**/api/v1/team/budget-approvals**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify([
          {
            id: 'appr-1',
            user_id: 'u-1',
            campaign_id: 'c-1',
            requested_budget_micro: 5_000_000,
            previous_budget_micro: 1_000_000,
            status: 'pending',
          },
        ]),
      });
      return;
    }
    await route.continue();
  });

  let approved = false;
  await page.route('**/api/v1/team/budget-approvals/appr-1/approve**', async (route) => {
    approved = true;
    await route.fulfill({ status: 204, body: '' });
  });

  await page.goto('/team?customer_id=cust-team-1&tab=approvals');
  await page.getByRole('button', { name: 'Approve' }).click();
  await expect.poll(() => approved).toBe(true);
});

test('admin campaign config tab saves name', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns/camp-owner-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          id: 'camp-owner-1',
          name: 'Owner campaign',
          status: 'active',
          customer_id: 'cust-1',
        }),
      });
      return;
    }
    if (route.request().method() === 'PATCH') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ id: 'camp-owner-1', name: 'Updated' }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-owner-1?tab=config');
  await page.getByLabel('Name').fill('Updated');
  await page.getByRole('button', { name: 'Save configuration' }).click();
  await expect(page.getByText('Campaign configuration updated')).toBeVisible();
});
