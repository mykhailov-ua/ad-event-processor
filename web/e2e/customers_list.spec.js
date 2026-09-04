import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  gotoLive,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('customers directory loads rows from GET /api/v1/customers', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/customers');
  await mainHeading(page, 'Customers').waitFor({ timeout: 15_000 });

  const listBody = await page.evaluate(async () => {
    const response = await fetch('/api/v1/customers', { credentials: 'include' });
    if (!response.ok) {
      throw new Error(`customers list failed: ${response.status}`);
    }
    return response.json();
  });

  await expectApiListBoundToDom(page, listBody, {
    emptyTitle: 'No customers',
    rowLabel: (row) => String(row.name ?? ''),
  });
});
