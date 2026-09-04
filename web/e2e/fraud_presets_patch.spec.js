import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('fraud presets load from GET /api/v1/fraud/presets', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(isApiGet('/api/v1/fraud/presets'), { timeout: 20_000 });
  await gotoLive(page, '/fraud/presets');
  await expect(page.getByRole('heading', { name: 'Fraud presets' })).toBeVisible();

  const response = await listResponse;
  const body = await response.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No presets',
    rowLabel: (row) => String(row.name ?? ''),
  });

  if (body.length > 0) {
    await expect(page.getByRole('columnheader', { name: 'Name' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Pass' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save', exact: true }).first()).toBeVisible();
  }
});
