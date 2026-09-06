import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  gotoLiveAwaitGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('audit page loads rows from GET /api/v1/audit', async ({ page }) => {
  await loginAsAdmin(page);
  const listResponse = await gotoLiveAwaitGet(page, '/audit', '/api/v1/audit');

  await expect(page.getByRole('button', { name: 'Export CSV' })).toBeVisible();
  await expect(page.getByText('Redact PII in export')).toBeVisible();

  const body = await listResponse.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No audit entries',
    rowLabel: (row) => String(row.action ?? row.resource ?? row.id ?? ''),
  });
});
