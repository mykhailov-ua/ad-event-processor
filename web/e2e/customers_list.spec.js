import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  gotoLiveAwaitGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('customers directory loads rows from GET /api/v1/customers', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = await gotoLiveAwaitGet(page, '/customers', '/api/v1/customers');
  await mainHeading(page, 'Customers').waitFor({ timeout: 15_000 });

  const listBody = await listResponse.json();
  expect(Array.isArray(listBody)).toBe(true);

  await expectApiListBoundToDom(page, listBody, {
    emptyTitle: 'No customers',
    rowLabel: (row) => String(row.name ?? ''),
  });
});
