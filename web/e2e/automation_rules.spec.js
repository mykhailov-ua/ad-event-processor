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

test('automation rules list loads from GET /api/v1/automation/rules', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(isApiGet('/api/v1/automation/rules'), { timeout: 20_000 });
  await gotoLive(page, '/automation/rules');
  await expect(page.getByRole('heading', { name: 'Automation rules' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Automation sections' })).toBeVisible();

  const response = await listResponse;
  const body = await response.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No automation rules',
    rowLabel: (row) => String(row.name ?? row.id ?? ''),
  });
});
