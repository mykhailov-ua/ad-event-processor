import { test, expect } from '@playwright/test';

import {
  applyCustomerScopeIfPrompted,
  expectApiListBoundToDom,
  fetchSessionCustomerId,
  gotoLive,
  gotoLiveTeam,
  isApiGet,
  loginAsAdmin,
  mainContent,
  skipUnlessIntegrationReady,
  stubApiRoute,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('team members load from GET /api/v1/team/members', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(isApiGet('/api/v1/team/members'), { timeout: 20_000 });
  await gotoLive(page, '/team');
  await expect(page.getByRole('heading', { name: 'Team' })).toBeVisible();
  await applyCustomerScopeIfPrompted(page);

  const response = await listResponse;
  const body = await response.json();
  expect(body).toHaveProperty('items');
  expect(Array.isArray(body.items)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No members',
    rowLabel: (row) => String(row.email ?? ''),
  });

  if (body.items.length > 0) {
    await expect(page.getByRole('columnheader', { name: 'Email' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Role' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save', exact: true }).first()).toBeVisible();
  }
});

test('team member PATCH 500 shows mutation error', { tag: '@L3' }, async ({ page }) => {
  await loginAsAdmin(page);
  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    test.skip(true, 'integration: no default_customer_id in session');
    return;
  }

  const loaded = await gotoLiveTeam(page, customerId);
  if (!loaded) {
    test.skip(true, 'integration: team overview unavailable for session customer');
    return;
  }

  const membersResponse = await page.waitForResponse(isApiGet('/api/v1/team/members'), {
    timeout: 20_000,
  });
  const membersBody = await membersResponse.json();
  const firstMember = membersBody.items?.[0];
  if (!firstMember?.user_id) {
    test.skip(true, 'integration: no team members to patch');
    return;
  }

  await stubApiRoute(page, 'PATCH', `/api/v1/team/members/${firstMember.user_id}`, 500, {
    code: 'INTERNAL_ERROR',
    message: 'team member update failed',
  });

  await page.getByRole('button', { name: 'Save', exact: true }).first().click();

  await expect(mainContent(page).getByRole('alert')).toBeVisible({ timeout: 15_000 });
  await expect(
    mainContent(page).getByText('The server encountered an error. Try again later.', {
      exact: true,
    })
  ).toBeVisible();
});
