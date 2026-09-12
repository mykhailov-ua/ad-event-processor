import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  expectErrorBlockVisible,
  gotoLive,
  gotoLiveAwaitGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
  stubApiGetError,
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

test(
  'customers list GET 500 shows ErrorBlock without empty table',
  { tag: '@L3' },
  async ({ page }) => {
    await loginAsAdmin(page);
    await stubApiGetError(page, '/api/v1/customers', 500, 'customers unavailable');

    const failedList = page.waitForResponse(
      (response) =>
        response.request().method() === 'GET' &&
        response.url().includes('/api/v1/customers') &&
        response.status() === 500,
      { timeout: 20_000 }
    );
    await gotoLive(page, '/customers');
    await failedList;

    await expectErrorBlockVisible(page, 'Could not load customers');
    await expect(page.getByText('No customers', { exact: true })).not.toBeVisible();
  }
);
