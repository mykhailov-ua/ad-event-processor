import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  fetchSessionCustomerId,
  isApiGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud integrations list loads from GET /api/v1/fraud/integrations', async ({
  page,
}, testInfo) => {
  await loginAsAdmin(page);
  await page.goto('/fraud/integrations');
  await expect(page.getByRole('heading', { name: 'Fraud integrations' })).toBeVisible();

  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    testInfo.skip(true, 'integration: session has no default customer_id');
    return;
  }

  await page.locator('#fraud-customer-id').fill(customerId);
  const listResponse = page.waitForResponse(isApiGet('/api/v1/fraud/integrations'), {
    timeout: 30_000,
  });
  await page.getByRole('button', { name: 'Load', exact: true }).click();
  const response = await listResponse;
  expect(response.ok()).toBe(true);

  const body = await response.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No integrations',
    rowLabel: (row) => String(row.provider ?? row.name ?? row.id ?? ''),
  });
});
