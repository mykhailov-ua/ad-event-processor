import { test, expect } from '@playwright/test';

import {
  expectApiListBoundToDom,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  mainContent,
  skipUnlessIntegrationReady,
  stubApiGetError,
} from './helpers.js';

const INTEGRATIONS_SECTION_READS = [
  { path: '/integrations/cost-sync', api: '/api/v1/cost-sync/snapshot' },
  { path: '/integrations/postbacks', api: '/api/v1/postbacks/snapshot' },
  { path: '/integrations/schemas', api: '/api/v1/integration/snapshot' },
  { path: '/integrations/platform-campaigns', api: '/api/v1/platform-campaigns/links' },
  { path: '/integrations/affiliate-presets', api: '/api/v1/integration/affiliate-status-presets' },
];

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('integrations hub shows section navigation', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/integrations');
  await expect(page.getByRole('heading', { name: 'Integrations' })).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Integrations sections' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Cost sync' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Affiliate presets' })).toBeVisible();
});

test('integrations sections load live read endpoints', async ({ page }) => {
  await loginAsAdmin(page);

  for (const { path, api } of INTEGRATIONS_SECTION_READS) {
    const listResponse = page.waitForResponse(isApiGet(api), { timeout: 20_000 });
    await gotoLive(page, path);
    await expect(page.getByRole('navigation', { name: 'Integrations sections' })).toBeVisible();
    const response = await listResponse;
    expect(response.ok()).toBe(true);
    const body = await response.json();
    expect(body).toBeTruthy();
  }
});

test('affiliate presets read returns rows from GET /api/v1/integration/affiliate-status-presets', async ({
  page,
}) => {
  await loginAsAdmin(page);
  const listResponse = page.waitForResponse(
    isApiGet('/api/v1/integration/affiliate-status-presets'),
    { timeout: 20_000 }
  );
  await gotoLive(page, '/integrations/affiliate-presets');
  await expect(page.getByRole('heading', { name: 'Affiliate status presets' })).toBeVisible();

  const response = await listResponse;
  const body = await response.json();
  expect(Array.isArray(body)).toBe(true);

  await expectApiListBoundToDom(page, body, {
    emptyTitle: 'No presets',
    rowLabel: (row) => String(row.name ?? ''),
  });
});

test('affiliate presets GET 500 shows ErrorBlock without empty table', async ({ page }) => {
  await loginAsAdmin(page);
  await stubApiGetError(
    page,
    '/api/v1/integration/affiliate-status-presets',
    500,
    'affiliate presets unavailable'
  );

  const failedList = page.waitForResponse(
    (response) =>
      response.request().method() === 'GET' &&
      response.url().includes('/api/v1/integration/affiliate-status-presets') &&
      response.status() === 500,
    { timeout: 20_000 }
  );
  await gotoLive(page, '/integrations/affiliate-presets');
  await failedList;

  const content = mainContent(page);
  await expect(content.getByRole('alert')).toBeVisible();
  await expect(content.getByText('Could not load affiliate presets', { exact: true })).toBeVisible();
  await expect(content.getByText('No presets', { exact: true })).not.toBeVisible();
});
