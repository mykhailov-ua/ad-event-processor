import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  gotoLive,
  gotoLiveAwaitGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
  stubApiGetError,
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

test(
  'audit list GET 500 shows ErrorBlock without empty table',
  { tag: '@L3' },
  async ({ page }) => {
    await loginAsAdmin(page);
    await stubApiGetError(page, '/api/v1/audit', 500, 'audit list unavailable');

    const failedList = page.waitForResponse(
      (response) =>
        response.request().method() === 'GET' &&
        response.url().includes('/api/v1/audit') &&
        response.status() === 500,
      { timeout: 20_000 }
    );
    await gotoLive(page, '/audit');
    await failedList;

    await expect(page.getByRole('alert')).toBeVisible();
    await expect(page.getByText('Could not load audit log', { exact: true })).toBeVisible();
    await expect(page.getByText('No audit entries', { exact: true })).not.toBeVisible();
    await expect(page.getByRole('button', { name: 'Export CSV' })).not.toBeVisible();
  }
);
