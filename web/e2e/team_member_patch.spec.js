import { test, expect } from '@playwright/test';

import {
  applyCustomerScopeIfPrompted,
  expectApiListBoundToDom,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
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
