import { test, expect } from '@playwright/test';

import {
  applyCustomerScopeIfPrompted,
  expectApiListBoundToDom,
  gotoLiveAwaitGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud labels page loads list from GET /api/v1/fraud/labels', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = await gotoLiveAwaitGet(page, '/fraud/labels', '/api/v1/fraud/labels');
  await expect(mainHeading(page, 'Fraud labels')).toBeVisible();
  await applyCustomerScopeIfPrompted(page);

  const body = await listResponse.json();
  expect(body).toHaveProperty('items');
  expect(Array.isArray(body.items)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No labels',
    rowLabel: (row) => String(row.ip_hash ?? row.reason ?? ''),
  });
});
