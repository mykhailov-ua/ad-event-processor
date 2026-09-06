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

test('fraud presets page loads from GET /api/v1/fraud/presets', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = await gotoLiveAwaitGet(page, '/fraud/presets', '/api/v1/fraud/presets');
  await expect(mainHeading(page, 'Fraud presets')).toBeVisible();

  const body = await listResponse.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No presets',
    rowLabel: (row) => String(row.name ?? row.id ?? ''),
  });
});
