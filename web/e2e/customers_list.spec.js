import { test, expect } from '@playwright/test';

import { gotoCustomers, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('customers directory shows table or empty state', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCustomers(page);

  const emptyState = page.getByText('No customers');
  const tableHeader = page.getByRole('columnheader', { name: 'Name' });

  await expect(emptyState.or(tableHeader)).toBeVisible({ timeout: 15_000 });
});
